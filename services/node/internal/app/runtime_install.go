package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/install"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
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
	ActiveHermesHostMutation(context.Context, string) (operation.Operation, bool, error)
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
	ActiveHermesPrerequisite(context.Context, string) (operation.Operation, bool, error)
	ActiveHermesHostMutation(context.Context, string) (operation.Operation, bool, error)
	LatestRuntimeInstall(context.Context, string) (operation.Operation, bool, error)
	PreviousRuntimeInstall(context.Context, string, string) (operation.Operation, bool, error)
	UpdateOperation(context.Context, operation.Operation, operation.Operation) error
	InterruptActiveInstalls(context.Context, time.Time) ([]operation.Operation, error)
	ListOperations(context.Context, string, string, int) ([]operation.Operation, error)
}

type RuntimeInstall struct {
	discovery            *RuntimeDiscovery
	store                installOperationStore
	now                  func() time.Time
	newID                func() (string, error)
	applier              HostApplier
	prereq               PrerequisiteApplier
	completer            InstallCompleter
	nodeID               string
	workers              map[string]*installWorker
	mu                   sync.Mutex
	allowNonWindows      bool
	logger               *slog.Logger
	events               *events.Broker
	deadline             time.Duration
	gate                 *install.GateHolder
	managedRoot          string
	failAfterTxnCreate   bool
	failStageProjection  bool
	failAfterCommittedOp bool
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
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func (s *RuntimeInstall) WithLogger(logger *slog.Logger) *RuntimeInstall {
	if logger != nil {
		s.logger = logger
	}
	return s
}

func (s *RuntimeInstall) WithEvents(broker *events.Broker) *RuntimeInstall {
	s.events = broker
	return s
}

func (s *RuntimeInstall) WithDeadline(deadline time.Duration) *RuntimeInstall {
	if deadline > 0 {
		s.deadline = deadline
	}
	return s
}

func (s *RuntimeInstall) WithInstallGate(gate *install.GateHolder) *RuntimeInstall {
	s.gate = gate
	return s
}

func (s *RuntimeInstall) WithManagedRoot(root string) *RuntimeInstall {
	s.managedRoot = strings.TrimSpace(root)
	return s
}

func (s *RuntimeInstall) persistCreatedTransaction(op operation.Operation) (install.InstallTransaction, error) {
	if s.managedRoot == "" || op.Type != operation.TypeRuntimeInstall {
		return install.InstallTransaction{}, nil
	}
	lock, err := install.AcquireLock(s.managedRoot)
	if err != nil {
		if errors.Is(err, install.ErrLockBusy) {
			return install.InstallTransaction{}, InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallNotReady, Retryable: true}
		}
		return install.InstallTransaction{}, err
	}
	defer lock.Release()
	store, err := install.NewStore(s.managedRoot)
	if err != nil {
		return install.InstallTransaction{}, err
	}
	if store.ReadActive().Invalid() {
		return install.InstallTransaction{}, InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallBlockedUnsafe, Retryable: false}
	}
	if err := store.FailFailableNonterminals(s.now()); err != nil {
		return install.InstallTransaction{}, err
	}
	if store.HasNonterminal() {
		return install.InstallTransaction{}, InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallNotReady, Retryable: true}
	}
	version := ""
	if s.applier != nil {
		version = s.applier.ExpectedVersion()
	}
	txn, err := install.NewCreatedTransaction(op.TargetID, op.ID, op.SourcePin, version)
	if err != nil {
		return install.InstallTransaction{}, err
	}
	if err := store.SaveTransaction(txn); err != nil {
		return install.InstallTransaction{}, err
	}
	return store.LoadTransaction(txn.ID)
}

func (s *RuntimeInstall) markTransactionFailed(txn install.InstallTransaction) {
	if s.managedRoot == "" || txn.ID == "" {
		return
	}
	store, err := install.NewStore(s.managedRoot)
	if err != nil {
		return
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil {
		return
	}
	if loaded.State.Terminal() {
		return
	}
	loaded.State = install.StateFailed
	loaded.ErrorCode = install.CodeInterrupted
	loaded.UpdatedAt = s.now()
	_ = store.SaveTransaction(loaded)
}

func (s *RuntimeInstall) loadLinkedTransaction(op operation.Operation) (install.InstallTransaction, bool) {
	if s.managedRoot == "" || op.TransactionID == "" {
		return install.InstallTransaction{}, false
	}
	store, err := install.NewStore(s.managedRoot)
	if err != nil {
		return install.InstallTransaction{}, false
	}
	txn, err := store.LoadTransaction(op.TransactionID)
	if err != nil {
		return install.InstallTransaction{}, false
	}
	return txn, true
}

func StageFromTransaction(txn install.InstallTransaction) operation.Stage {
	switch txn.State {
	case install.StateCreated:
		return operation.StagePreflight
	case install.StateBuilding:
		return operation.StageSourceDownload
	case install.StateSealed:
		return operation.StageInstallBootstrapMarker
	case install.StatePublished, install.StateActivating:
		return operation.StageInstallPath
	case install.StateCommitted:
		return operation.StageCleanup
	default:
		return operation.StagePreflight
	}
}

func (s *RuntimeInstall) projectOperation(ctx context.Context, op operation.Operation) (operation.Operation, error) {
	txn, ok := s.loadLinkedTransaction(op)
	if !ok {
		return op, nil
	}
	next := op
	next.Stage = StageFromTransaction(txn)
	now := s.now()
	if txn.State == install.StateCommitted {
		if op.Status == operation.StatusSucceeded && op.Stage == operation.StageCleanup {
			return op, nil
		}
		if op.Status == operation.StatusPending {
			next.Status = operation.StatusRunning
			next.StartedAt = &now
			next.UpdatedAt = now
			if err := s.persistUpdate(ctx, op, next); err != nil {
				next.Status = operation.StatusSucceeded
				next.CompletedAt = &now
				return next, nil
			}
			op = next
		}
		next = op
		next.Status = operation.StatusSucceeded
		next.Stage = operation.StageCleanup
		next.ErrorCode = ""
		next.Retryable = false
		next.CompletedAt = &now
		next.UpdatedAt = now
		if next.StartedAt == nil {
			next.StartedAt = &now
		}
		_ = s.persistUpdate(ctx, op, next)
		return next, nil
	}
	if txn.State == install.StateActivating {
		if op.Status == operation.StatusRunning && next.Stage == op.Stage {
			return op, nil
		}
		next.Status = operation.StatusRunning
		next.ErrorCode = ""
		next.Retryable = false
		next.CompletedAt = nil
		next.UpdatedAt = now
		if next.StartedAt == nil {
			next.StartedAt = &now
		}
		_ = s.persistUpdate(ctx, op, next)
		return next, nil
	}
	if txn.State == install.StateFailed && !operation.IsTerminal(op.Status) {
		next.Status = operation.StatusFailed
		if txn.ErrorCode != "" {
			next.ErrorCode = yorvaruntime.ErrorCode(txn.ErrorCode)
		}
		next.Retryable = true
		next.CompletedAt = &now
		next.UpdatedAt = now
		_ = s.persistUpdate(ctx, op, next)
		return next, nil
	}
	if next.Stage != op.Stage && !operation.IsTerminal(op.Status) {
		next.UpdatedAt = now
		if err := s.persistUpdate(ctx, op, next); err != nil {
			return next, nil
		}
	}
	return next, nil
}

func (s *RuntimeInstall) projectOpenInstalls(ctx context.Context) {
	ops, err := s.store.ListOperations(ctx, string(operation.TargetRuntimeKind), "", 50)
	if err != nil {
		return
	}
	for _, op := range ops {
		if op.Type == operation.TypeRuntimeInstall {
			_, _ = s.projectOperation(ctx, op)
		}
	}
}

func (s *RuntimeInstall) filesystemOwnsOperation(op operation.Operation) bool {
	txn, ok := s.loadLinkedTransaction(op)
	if !ok {
		return false
	}
	return txn.State == install.StateActivating || txn.State == install.StateCommitted
}

func (s *RuntimeInstall) rejectInstallGate() error {
	if s.managedRoot != "" {
		if store, err := install.NewStore(s.managedRoot); err == nil && store.ReadActive().Invalid() {
			if s.gate != nil {
				s.gate.Set(install.GateBlockedUnsafe)
			}
			return InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallBlockedUnsafe, Retryable: false}
		}
	}
	switch s.gate.Get() {
	case install.GateReady, "":
		return nil
	case install.GateReconciling:
		return InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallNotReady, Retryable: true}
	default:
		return InstallRejection{Code: yorvaruntime.ErrorRuntimeInstallBlockedUnsafe, Retryable: false}
	}
}

func (s *RuntimeInstall) Start(ctx context.Context, kind yorvaruntime.Kind, idempotencyKey string) (InstallStartResult, error) {
	if err := s.rejectInstallGate(); err != nil {
		return InstallStartResult{}, err
	}
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.store.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return s.replayIdempotent(existing, operation.TypeRuntimeInstall, string(kind))
	}

	if err := s.rejectActiveHermesMutation(ctx, string(kind)); err != nil {
		return s.replayIfCurrentKey(ctx, idempotencyKey, operation.TypeRuntimeInstall, string(kind), err)
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
		s.logInstall("rejected", operation.Operation{
			Type:       operation.TypeRuntimeInstall,
			TargetType: operation.TargetRuntimeKind,
			TargetID:   string(kind),
			Status:     operation.StatusFailed,
			Stage:      operation.StagePreflight,
			ErrorCode:  decision.Code,
			Retryable:  decision.Retryable,
		})
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
	if s.applier != nil {
		created.SourcePin = s.applier.ExpectedPin()
	}
	txn, txnErr := s.persistCreatedTransaction(created)
	if txnErr != nil {
		return InstallStartResult{}, txnErr
	}
	if txn.ID != "" {
		created.TransactionID = txn.ID
	}
	if s.failAfterTxnCreate {
		s.markTransactionFailed(txn)
		return InstallStartResult{}, errors.New("injected fail after CREATED")
	}
	if err := s.persistCreate(ctx, created); err != nil {
		s.markTransactionFailed(txn)
		return s.recoverCreate(ctx, created, err)
	}
	s.logInstall("created", created)
	if s.applier != nil {
		workerCtx := s.bindWorker(context.Background(), created.ID)
		go s.execute(workerCtx, created)
	}
	return InstallStartResult{Operation: created, Created: true}, nil
}

func (s *RuntimeInstall) Get(ctx context.Context, id string) (operation.Operation, error) {
	value, err := s.store.GetOperation(ctx, id)
	if err != nil {
		return operation.Operation{}, err
	}
	return s.projectOperation(ctx, value)
}

func (s *RuntimeInstall) List(ctx context.Context, targetType, targetID string, limit int) ([]operation.Operation, error) {
	return s.store.ListOperations(ctx, targetType, targetID, limit)
}

func (s *RuntimeInstall) Cancel(ctx context.Context, id string) (operation.Operation, error) {
	current, err := s.store.GetOperation(ctx, id)
	if err != nil {
		return operation.Operation{}, err
	}
	if s.filesystemOwnsOperation(current) {
		return s.projectOperation(ctx, current)
	}
	if operation.IsTerminal(current.Status) {
		return operation.Operation{}, InstallRejection{
			Code:      yorvaruntime.ErrorOperationNotCancellable,
			Retryable: false,
		}
	}
	done := s.requestCancel(id)
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
		case <-time.After(operationCancelWait):
		}
		latest, getErr := s.store.GetOperation(ctx, id)
		if getErr != nil {
			return operation.Operation{}, getErr
		}
		if latest.Status == operation.StatusCancelled {
			s.logInstall("cancelled", latest)
			return latest, nil
		}
		if operation.IsTerminal(latest.Status) {
			return operation.Operation{}, InstallRejection{
				Code:      yorvaruntime.ErrorOperationNotCancellable,
				Retryable: false,
			}
		}
		return latest, nil
	}
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.ErrorCode = yorvaruntime.ErrorRuntimeInstallCancelled
	next.Retryable = true
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.persistUpdate(ctx, current, next); err != nil {
		latest, getErr := s.store.GetOperation(ctx, id)
		if getErr == nil && latest.Status == operation.StatusCancelled {
			s.logInstall("cancelled", latest)
			return latest, nil
		}
		return operation.Operation{}, err
	}
	s.logInstall("cancelled", next)
	return next, nil
}

func (s *RuntimeInstall) rejectActiveHermesMutation(ctx context.Context, runtimeKind string) error {
	active, ok, err := s.store.ActiveHermesHostMutation(ctx, runtimeKind)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return InstallRejection{
		Code:      yorvaruntime.ErrorRuntimeInstallInProgress,
		Retryable: true,
		ActiveID:  active.ID,
	}
}

func (s *RuntimeInstall) normalizeCreateConflict(ctx context.Context, runtimeKind string, err error) error {
	if err == nil {
		return nil
	}
	if active, ok, lookupErr := s.store.ActiveHermesHostMutation(ctx, runtimeKind); lookupErr == nil && ok {
		return InstallRejection{
			Code:      yorvaruntime.ErrorRuntimeInstallInProgress,
			Retryable: true,
			ActiveID:  active.ID,
		}
	}
	return err
}

func (s *RuntimeInstall) replayIdempotent(existing operation.Operation, wantType operation.Type, targetID string) (InstallStartResult, error) {
	if existing.Type != wantType || existing.TargetType != operation.TargetRuntimeKind || existing.TargetID != targetID {
		return InstallStartResult{}, InstallRejection{Code: yorvaruntime.ErrorIdempotencyKeyConflict, Retryable: false}
	}
	return InstallStartResult{Operation: existing}, nil
}

func (s *RuntimeInstall) replayIfCurrentKey(ctx context.Context, key string, wantType operation.Type, targetID string, err error) (InstallStartResult, error) {
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorRuntimeInstallInProgress {
		return InstallStartResult{}, err
	}
	existing, ok, getErr := s.store.GetOperationByIdempotencyKey(ctx, key)
	if getErr != nil {
		return InstallStartResult{}, getErr
	}
	if ok {
		return s.replayIdempotent(existing, wantType, targetID)
	}
	return InstallStartResult{}, err
}

func (s *RuntimeInstall) recoverCreate(ctx context.Context, intended operation.Operation, err error) (InstallStartResult, error) {
	existing, ok, getErr := s.store.GetOperationByIdempotencyKey(ctx, intended.IdempotencyKey)
	if getErr != nil {
		return InstallStartResult{}, getErr
	}
	if ok {
		return s.replayIdempotent(existing, intended.Type, intended.TargetID)
	}
	if errors.Is(err, sqlite.ErrDuplicateIdempotency) {
		return InstallStartResult{}, InstallRejection{Code: yorvaruntime.ErrorIdempotencyKeyConflict, Retryable: false}
	}
	return InstallStartResult{}, s.normalizeCreateConflict(ctx, intended.TargetID, err)
}

func (s *RuntimeInstall) InterruptStale(ctx context.Context) ([]operation.Operation, error) {
	s.projectOpenInstalls(ctx)
	active, err := s.store.ListOperations(ctx, string(operation.TargetRuntimeKind), "", 50)
	if err != nil {
		return nil, err
	}
	var interrupted []operation.Operation
	now := s.now()
	for _, current := range active {
		if current.Type != operation.TypeRuntimeInstall && current.Type != operation.TypeHermesPrerequisites {
			continue
		}
		if operation.IsTerminal(current.Status) {
			continue
		}
		if current.Type == operation.TypeRuntimeInstall && s.filesystemOwnsOperation(current) {
			continue
		}
		next := current
		completed := now.UTC()
		next.Status = operation.StatusFailed
		next.ErrorCode = yorvaruntime.ErrorOperationInterrupted
		next.ErrorMessage = ""
		next.Retryable = true
		next.CompletedAt = &completed
		next.UpdatedAt = completed
		if err := s.persistUpdate(ctx, current, next); err != nil {
			return interrupted, err
		}
		interrupted = append(interrupted, next)
		s.emitOperation(operation.Operation{}, next, false)
	}
	return interrupted, nil
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

func RetryEligibleForPin(latest operation.Operation, expectedPin string) bool {
	if !RetryEligible(latest) {
		return false
	}
	if expectedPin == "" || latest.SourcePin == "" || latest.OwnershipNonce == "" || latest.SourcePin != expectedPin {
		return false
	}
	return true
}

func (s *RuntimeInstall) logInstall(event string, op operation.Operation) {
	if s == nil || s.logger == nil {
		return
	}
	attrs := []any{
		"event", event,
		"operationId", op.ID,
		"correlationId", op.CorrelationID,
		"runtimeKind", op.TargetID,
		"stage", string(op.Stage),
		"status", string(op.Status),
	}
	if op.ErrorCode != "" {
		attrs = append(attrs, "errorCode", string(op.ErrorCode), "retryable", op.Retryable)
	}
	s.logger.Info("runtime install", attrs...)
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

func newOwnershipNonce() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "own_" + base64.RawURLEncoding.EncodeToString(random), nil
}
