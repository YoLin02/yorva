package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrInstallRejected       = errors.New("runtime install rejected")
)

type OperationStore interface {
	CreateOperation(context.Context, operation.Operation) error
	GetOperation(context.Context, string) (operation.Operation, error)
	GetOperationByIdempotencyKey(context.Context, string) (operation.Operation, bool, error)
	ActiveRuntimeInstall(context.Context, string) (operation.Operation, bool, error)
	LatestRuntimeInstall(context.Context, string) (operation.Operation, bool, error)
	ListOperations(context.Context, any) ([]operation.Operation, error)
	UpdateOperation(context.Context, operation.Operation, operation.Operation) error
	InterruptActiveInstalls(context.Context, time.Time) ([]operation.Operation, error)
}

type installOperationStore interface {
	CreateOperation(context.Context, operation.Operation) error
	GetOperation(context.Context, string) (operation.Operation, error)
	GetOperationByIdempotencyKey(context.Context, string) (operation.Operation, bool, error)
	ActiveRuntimeInstall(context.Context, string) (operation.Operation, bool, error)
	LatestRuntimeInstall(context.Context, string) (operation.Operation, bool, error)
	PreviousRuntimeInstall(context.Context, string, string) (operation.Operation, bool, error)
	UpdateOperation(context.Context, operation.Operation, operation.Operation) error
	InterruptActiveInstalls(context.Context, time.Time) ([]operation.Operation, error)
	ListOperations(context.Context, string, string, int) ([]operation.Operation, error)
}

type RuntimeInstall struct {
	discovery       *RuntimeDiscovery
	store           installOperationStore
	now             func() time.Time
	newID           func() (string, error)
	applier         HostApplier
	completer       InstallCompleter
	nodeID          string
	cancels         map[string]context.CancelFunc
	mu              sync.Mutex
	allowNonWindows bool
}

type InstallStartResult struct {
	Operation operation.Operation
	Created   bool
}

type InstallRejection struct {
	Code      yorvaruntime.ErrorCode
	Retryable bool
	ActiveID  string
}

func (e InstallRejection) Error() string {
	return fmt.Sprintf("runtime install rejected: %s", e.Code)
}

func NewRuntimeInstall(discovery *RuntimeDiscovery, store installOperationStore) *RuntimeInstall {
	return &RuntimeInstall{
		discovery: discovery,
		store:     store,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     newOperationID,
	}
}

func (s *RuntimeInstall) Start(ctx context.Context, kind yorvaruntime.Kind, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.store.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{Operation: existing}, nil
	}

	if active, ok, err := s.store.ActiveRuntimeInstall(ctx, string(kind)); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{}, InstallRejection{
			Code:      yorvaruntime.ErrorRuntimeInstallInProgress,
			Retryable: true,
			ActiveID:  active.ID,
		}
	}

	discovery, err := s.discovery.Detect(ctx, kind)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, ErrRuntimeKindNotFound) {
			return InstallStartResult{}, err
		}
		return InstallStartResult{}, InstallRejection{
			Code:      yorvaruntime.ErrorRuntimeDiscoveryTimeout,
			Retryable: true,
		}
	}
	latest, _, err := s.store.LatestRuntimeInstall(ctx, string(kind))
	if err != nil {
		return InstallStartResult{}, err
	}
	if decision := DecideInstallPreflight(discovery, latest); !decision.Allow {
		return InstallStartResult{}, InstallRejection{Code: decision.Code, Retryable: decision.Retryable}
	}

	now := s.now()
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, err
	}
	correlation, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, err
	}
	created := operation.Operation{
		ID:             id,
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       string(kind),
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  correlation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.CreateOperation(ctx, created); err != nil {
		return InstallStartResult{}, err
	}
	if s.applier != nil {
		workerCtx := s.bindWorker(context.Background(), created.ID)
		go s.execute(workerCtx, created)
	}
	return InstallStartResult{Operation: created, Created: true}, nil
}

func (s *RuntimeInstall) Get(ctx context.Context, id string) (operation.Operation, error) {
	return s.store.GetOperation(ctx, id)
}

func (s *RuntimeInstall) List(ctx context.Context, targetType, targetID string, limit int) ([]operation.Operation, error) {
	return s.store.ListOperations(ctx, targetType, targetID, limit)
}

func (s *RuntimeInstall) Cancel(ctx context.Context, id string) (operation.Operation, error) {
	current, err := s.store.GetOperation(ctx, id)
	if err != nil {
		return operation.Operation{}, err
	}
	if operation.IsTerminal(current.Status) {
		return operation.Operation{}, InstallRejection{
			Code:      yorvaruntime.ErrorOperationNotCancellable,
			Retryable: false,
		}
	}
	s.stopWorker(id)
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.ErrorCode = yorvaruntime.ErrorRuntimeInstallCancelled
	next.Retryable = true
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.store.UpdateOperation(ctx, current, next); err != nil {
		return operation.Operation{}, err
	}
	return next, nil
}

func (s *RuntimeInstall) InterruptStale(ctx context.Context) ([]operation.Operation, error) {
	return s.store.InterruptActiveInstalls(ctx, s.now())
}

type PreflightDecision struct {
	Allow     bool
	Code      yorvaruntime.ErrorCode
	Retryable bool
}

func DecideInstallPreflight(discovery yorvaruntime.Discovery, latest operation.Operation) PreflightDecision {
	switch discovery.State {
	case yorvaruntime.DiscoveryNotInstalled:
		return PreflightDecision{Allow: true}
	case yorvaruntime.DiscoverySupported:
		return PreflightDecision{Code: yorvaruntime.ErrorRuntimeInstallAlreadyPresent}
	case yorvaruntime.DiscoveryBrokenExecutable:
		if RetryEligible(latest) {
			return PreflightDecision{Allow: true}
		}
		return PreflightDecision{Code: yorvaruntime.ErrorRuntimeInstallStateConflict}
	case yorvaruntime.DiscoveryTimedOut:
		return PreflightDecision{Code: yorvaruntime.ErrorRuntimeDiscoveryTimeout, Retryable: true}
	case yorvaruntime.DiscoveryUnsupported, yorvaruntime.DiscoveryAmbiguous, yorvaruntime.DiscoveryMalformedVersion:
		return PreflightDecision{Code: yorvaruntime.ErrorRuntimeInstallStateConflict}
	default:
		return PreflightDecision{Code: yorvaruntime.ErrorRuntimeInstallStateConflict}
	}
}

func RetryEligible(latest operation.Operation) bool {
	if latest.ID == "" || latest.Type != operation.TypeRuntimeInstall {
		return false
	}
	if latest.Status != operation.StatusFailed && latest.Status != operation.StatusCancelled {
		return false
	}
	return latest.Retryable
}

func ValidateIdempotencyKey(value string) error {
	if len(value) < 1 || len(value) > 128 {
		return ErrInvalidIdempotencyKey
	}
	for _, character := range value {
		if character < 33 || character > 126 {
			return ErrInvalidIdempotencyKey
		}
	}
	return nil
}

func newOperationID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "op_" + base64.RawURLEncoding.EncodeToString(random), nil
}

func newCorrelationID() (string, error) {
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "cor_" + base64.RawURLEncoding.EncodeToString(random), nil
}
