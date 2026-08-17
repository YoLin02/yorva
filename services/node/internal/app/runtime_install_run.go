package app

import (
	"context"
	"errors"
	"runtime"
	"strings"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type HostApplier interface {
	PlatformSupported() bool
	ValidateTarget(retry bool) error
	Apply(ctx context.Context, operationID string, report func(operation.Stage)) error
	ManagedInstallDir() string
	ExpectedVersion() string
	ContainsManagedPath(path string) bool
}

type InstallCompleter interface {
	CompleteInstallSuccess(context.Context, operation.Operation, operation.Operation, sqlite.AcceptedInstallation) error
}

func (s *RuntimeInstall) WithHost(applier HostApplier, completer InstallCompleter, nodeID string) *RuntimeInstall {
	s.applier = applier
	s.completer = completer
	s.nodeID = nodeID
	s.cancels = map[string]context.CancelFunc{}
	return s
}

func (s *RuntimeInstall) bindWorker(parent context.Context, id string) context.Context {
	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	if s.cancels == nil {
		s.cancels = map[string]context.CancelFunc{}
	}
	s.cancels[id] = cancel
	s.mu.Unlock()
	return ctx
}

func (s *RuntimeInstall) stopWorker(id string) {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	if ok {
		delete(s.cancels, id)
	}
	s.mu.Unlock()
	if ok {
		cancel()
	}
}

func (s *RuntimeInstall) execute(ctx context.Context, started operation.Operation) {
	defer s.stopWorker(started.ID)
	current := started
	fail := func(code yorvaruntime.ErrorCode, retryable bool) {
		now := s.now()
		next := current
		next.Status = operation.StatusFailed
		next.ErrorCode = code
		next.Retryable = retryable
		next.CompletedAt = &now
		next.UpdatedAt = now
		_ = s.store.UpdateOperation(context.Background(), current, next)
	}
	if s.applier == nil {
		return
	}
	if !s.applier.PlatformSupported() || runtime.GOOS != "windows" && !s.allowNonWindows {
		fail(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, false)
		return
	}
	previous, _, _ := s.store.PreviousRuntimeInstall(ctx, started.TargetID, started.ID)
	if err := s.applier.ValidateTarget(RetryEligible(previous)); err != nil {
		fail(installErrorCodeOr(err, yorvaruntime.ErrorRuntimeInstallTargetOccupied), false)
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.store.UpdateOperation(ctx, current, running); err != nil {
		return
	}
	current = running
	report := func(stage operation.Stage) {
		updated := current
		updated.Stage = stage
		updated.UpdatedAt = s.now()
		if err := s.store.UpdateOperation(context.Background(), current, updated); err == nil {
			current = updated
		}
	}
	if err := s.applier.Apply(ctx, started.ID, report); err != nil {
		if errors.Is(err, context.Canceled) {
			now := s.now()
			next := current
			next.Status = operation.StatusCancelled
			next.ErrorCode = yorvaruntime.ErrorRuntimeInstallCancelled
			next.Retryable = true
			next.CompletedAt = &now
			next.UpdatedAt = now
			_ = s.store.UpdateOperation(context.Background(), current, next)
			return
		}
		fail(installErrorCodeOr(err, yorvaruntime.ErrorRuntimeInstallStageFailed), isRetryableInstall(err))
		return
	}
	report(operation.StagePostcheckDiscovery)
	discovery, err := s.discovery.Detect(ctx, yorvaruntime.Kind(started.TargetID))
	if err != nil || !postcheckAccepted(discovery, s.applier) {
		fail(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, true)
		return
	}
	report(operation.StageCleanup)
	now = s.now()
	succeeded := current
	succeeded.Status = operation.StatusSucceeded
	succeeded.CompletedAt = &now
	succeeded.UpdatedAt = now
	if s.completer == nil {
		_ = s.store.UpdateOperation(context.Background(), current, succeeded)
		return
	}
	installationID, err := sqlite.NewInstallationID()
	if err != nil {
		fail(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, true)
		return
	}
	if err := s.completer.CompleteInstallSuccess(context.Background(), current, succeeded, sqlite.AcceptedInstallation{
		ID:             installationID,
		NodeID:         s.nodeID,
		RuntimeKind:    yorvaruntime.Kind(started.TargetID),
		InstallPath:    discovery.Selected.Path,
		Version:        discovery.Selected.Version,
		SupportState:   discovery.State,
		Status:         "accepted",
		LastDetectedAt: discovery.DetectedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		fail(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, true)
	}
}

func postcheckAccepted(discovery yorvaruntime.Discovery, applier HostApplier) bool {
	if discovery.State != yorvaruntime.DiscoverySupported || discovery.Selected == nil {
		return false
	}
	if discovery.Selected.Version != applier.ExpectedVersion() {
		return false
	}
	return applier.ContainsManagedPath(discovery.Selected.Path)
}

func installErrorCodeOr(err error, fallback yorvaruntime.ErrorCode) yorvaruntime.ErrorCode {
	message := err.Error()
	for _, code := range []yorvaruntime.ErrorCode{
		yorvaruntime.ErrorRuntimeInstallIntegrityFailed,
		yorvaruntime.ErrorRuntimeInstallSourceUnavailable,
		yorvaruntime.ErrorRuntimeInstallProtocolUnsupported,
		yorvaruntime.ErrorRuntimeInstallManifestMismatch,
		yorvaruntime.ErrorRuntimeInstallTimeout,
		yorvaruntime.ErrorRuntimeInstallOutputLimit,
		yorvaruntime.ErrorRuntimeInstallPrivilegeRequired,
		yorvaruntime.ErrorRuntimeInstallTargetOccupied,
		yorvaruntime.ErrorRuntimeInstallStageFailed,
	} {
		if strings.Contains(message, string(code)) {
			return code
		}
	}
	var coded interface{ Code() yorvaruntime.ErrorCode }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return fallback
}

func isRetryableInstall(err error) bool {
	code := installErrorCodeOr(err, "")
	switch code {
	case yorvaruntime.ErrorRuntimeInstallSourceUnavailable, yorvaruntime.ErrorRuntimeInstallTimeout, yorvaruntime.ErrorRuntimeInstallPostcheckFailed:
		return true
	default:
		return false
	}
}
