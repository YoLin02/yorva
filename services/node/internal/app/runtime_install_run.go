package app

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	operationDeadline   = 60 * time.Minute
	operationCancelWait = 45 * time.Second
)

type installWorker struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type HostApplier interface {
	PlatformSupported() bool
	SetInstallIdentity(operation.Operation)
	ValidateTarget(retry bool, previous operation.Operation) error
	ExpectedPin() string
	Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error
	ManagedInstallDir() string
	ExpectedVersion() string
	ContainsManagedPath(path string) bool
	CanonicalPublicLauncher() string
}

type InstallCompleter interface {
	CompleteInstallSuccess(context.Context, operation.Operation, operation.Operation, sqlite.AcceptedInstallation) error
}

func (s *RuntimeInstall) WithHost(applier HostApplier, completer InstallCompleter, nodeID string) *RuntimeInstall {
	s.applier = applier
	s.completer = completer
	s.nodeID = nodeID
	s.workers = map[string]*installWorker{}
	return s
}

func (s *RuntimeInstall) bindWorker(parent context.Context, id string) context.Context {
	timeout := operationDeadline
	if s.deadline > 0 {
		timeout = s.deadline
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	done := make(chan struct{})
	s.mu.Lock()
	if s.workers == nil {
		s.workers = map[string]*installWorker{}
	}
	s.workers[id] = &installWorker{cancel: cancel, done: done}
	s.mu.Unlock()
	return ctx
}

func (s *RuntimeInstall) requestCancel(id string) <-chan struct{} {
	s.mu.Lock()
	worker := s.workers[id]
	s.mu.Unlock()
	if worker == nil {
		return nil
	}
	worker.cancel()
	return worker.done
}

func (s *RuntimeInstall) stopWorker(id string) {
	s.mu.Lock()
	worker := s.workers[id]
	delete(s.workers, id)
	s.mu.Unlock()
	if worker == nil {
		return
	}
	worker.cancel()
	select {
	case <-worker.done:
	default:
		close(worker.done)
	}
}

func (s *RuntimeInstall) execute(ctx context.Context, started operation.Operation) {
	defer s.stopWorker(started.ID)
	current := started
	fail := func(code yorvaruntime.ErrorCode, retryable bool) {
		latest, err := s.store.GetOperation(context.Background(), started.ID)
		if err != nil || operation.IsTerminal(latest.Status) {
			return
		}
		now := s.now()
		next := latest
		next.Status = operation.StatusFailed
		next.ErrorCode = code
		next.Retryable = retryable
		next.CompletedAt = &now
		next.UpdatedAt = now
		_ = s.persistUpdate(context.Background(), latest, next)
		s.logInstall("failed", next)
	}
	if s.applier == nil {
		return
	}
	if !s.applier.PlatformSupported() || runtime.GOOS != "windows" && !s.allowNonWindows {
		fail(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, false)
		return
	}
	previous, _, _ := s.store.PreviousRuntimeInstall(ctx, started.TargetID, started.ID)
	s.applier.SetInstallIdentity(started)
	if err := s.applier.ValidateTarget(RetryEligibleForPin(previous, s.applier.ExpectedPin()), previous); err != nil {
		fail(installErrorCodeOr(err, yorvaruntime.ErrorRuntimeInstallTargetOccupied), false)
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.persistUpdate(ctx, current, running); err != nil {
		return
	}
	current = running
	report := func(stage operation.Stage, warning string) {
		updated := current
		updated.Stage = stage
		if warning != "" {
			updated.Message = warning
		}
		updated.UpdatedAt = s.now()
		if err := s.persistUpdate(context.Background(), current, updated); err == nil {
			current = updated
			s.logInstall("stage", current)
		}
	}
	if err := s.applier.Apply(ctx, started.ID, report); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fail(yorvaruntime.ErrorRuntimeInstallTimeout, true)
			return
		}
		if errors.Is(err, context.Canceled) {
			latest, getErr := s.store.GetOperation(context.Background(), started.ID)
			if getErr != nil || operation.IsTerminal(latest.Status) {
				return
			}
			now := s.now()
			next := latest
			next.Status = operation.StatusCancelled
			next.ErrorCode = yorvaruntime.ErrorRuntimeInstallCancelled
			next.Retryable = true
			next.CompletedAt = &now
			next.UpdatedAt = now
			_ = s.persistUpdate(context.Background(), latest, next)
			s.logInstall("cancelled", next)
			return
		}
		fail(installErrorCodeOr(err, yorvaruntime.ErrorRuntimeInstallStageFailed), isRetryableInstall(err))
		return
	}
	report(operation.StagePostcheckDiscovery, "")
	discovery, err := s.discovery.Detect(ctx, yorvaruntime.Kind(started.TargetID))
	if err != nil || !postcheckAccepted(discovery, s.applier) {
		fail(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, true)
		return
	}
	report(operation.StageCleanup, "")
	now = s.now()
	succeeded := current
	succeeded.Status = operation.StatusSucceeded
	succeeded.CompletedAt = &now
	succeeded.UpdatedAt = now
	if s.completer == nil {
		_ = s.persistUpdate(context.Background(), current, succeeded)
		s.logInstall("succeeded", succeeded)
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
		return
	}
	s.emitOperation(current, succeeded, false)
	s.logInstall("succeeded", succeeded)
}

func postcheckAccepted(discovery yorvaruntime.Discovery, applier HostApplier) bool {
	if discovery.State != yorvaruntime.DiscoverySupported || discovery.Selected == nil {
		return false
	}
	if discovery.Selected.Version != applier.ExpectedVersion() {
		return false
	}
	launcher := applier.CanonicalPublicLauncher()
	if launcher == "" || discovery.Selected.Path != launcher {
		return false
	}
	return applier.ContainsManagedPath(discovery.Selected.Path)
}

func installErrorCodeOr(err error, fallback yorvaruntime.ErrorCode) yorvaruntime.ErrorCode {
	message := err.Error()
	for _, code := range []yorvaruntime.ErrorCode{
		yorvaruntime.ErrorRuntimeInstallIntegrityFailed,
		yorvaruntime.ErrorRuntimeInstallInsufficientDisk,
		yorvaruntime.ErrorRuntimeInstallSourceUnavailable,
		yorvaruntime.ErrorRuntimeInstallProtocolUnsupported,
		yorvaruntime.ErrorRuntimeInstallManifestMismatch,
		yorvaruntime.ErrorRuntimeInstallTimeout,
		yorvaruntime.ErrorRuntimeInstallOutputLimit,
		yorvaruntime.ErrorRuntimeInstallPrivilegeRequired,
		yorvaruntime.ErrorRuntimeInstallTargetOccupied,
		yorvaruntime.ErrorRuntimeInstallStageFailed,
		yorvaruntime.ErrorHermesNodeMissing,
		yorvaruntime.ErrorHermesNodeUnsupported,
		yorvaruntime.ErrorHermesNPMMissing,
		yorvaruntime.ErrorHermesNPMUnsupported,
		yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed,
		yorvaruntime.ErrorHermesNPMArchiveIntegrityFailed,
		yorvaruntime.ErrorHermesNodeDepsFailed,
		yorvaruntime.ErrorHermesNodeDepsTimeout,
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
	case yorvaruntime.ErrorRuntimeInstallSourceUnavailable, yorvaruntime.ErrorRuntimeInstallTimeout, yorvaruntime.ErrorRuntimeInstallPostcheckFailed, yorvaruntime.ErrorRuntimeInstallInsufficientDisk:
		return true
	default:
		return false
	}
}
