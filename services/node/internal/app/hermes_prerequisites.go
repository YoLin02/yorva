package app

import (
	"context"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type PrerequisiteSnapshot struct {
	NodeState         string
	NodeVersion       string
	NodeCode          yorvaruntime.ErrorCode
	NPMState          string
	NPMVersion        string
	NPMCode           yorvaruntime.ErrorCode
	DepsState         string
	DepsCode          yorvaruntime.ErrorCode
	Retryable         bool
	CheckedAt         time.Time
	ActiveOperationID string
}

type PrerequisiteApplier interface {
	Inspect() PrerequisiteSnapshot
	Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error
}

func (s *RuntimeInstall) WithPrerequisite(applier PrerequisiteApplier) *RuntimeInstall {
	s.prereq = applier
	return s
}

func (s *RuntimeInstall) InspectPrerequisites(ctx context.Context) (PrerequisiteSnapshot, error) {
	if s.prereq == nil {
		return PrerequisiteSnapshot{}, ErrRuntimeKindNotFound
	}
	snap := s.prereq.Inspect()
	if active, ok, err := s.store.ActiveHermesHostMutation(ctx, "hermes"); err == nil && ok && active.Type == operation.TypeHermesPrerequisites {
		snap.ActiveOperationID = active.ID
	}
	if snap.DepsState != "READY" {
		if latest, ok, err := latestHermesPrerequisite(s.store, ctx); err == nil && ok {
			switch latest.ErrorCode {
			case yorvaruntime.ErrorHermesNodeDepsTimeout:
				snap.DepsState = "TIMED_OUT"
				snap.DepsCode = latest.ErrorCode
			case yorvaruntime.ErrorHermesNodeDepsFailed:
				snap.DepsState = "FAILED"
				snap.DepsCode = latest.ErrorCode
			}
		}
	}
	return snap, nil
}

func latestHermesPrerequisite(store installOperationStore, ctx context.Context) (operation.Operation, bool, error) {
	values, err := store.ListOperations(ctx, string(operation.TargetRuntimeKind), "hermes", 20)
	if err != nil {
		return operation.Operation{}, false, err
	}
	var latest operation.Operation
	found := false
	for _, value := range values {
		if value.Type != operation.TypeHermesPrerequisites {
			continue
		}
		if !found || value.CreatedAt.After(latest.CreatedAt) {
			latest = value
			found = true
		}
	}
	return latest, found, nil
}

func (s *RuntimeInstall) StartPrerequisites(ctx context.Context, idempotencyKey string) (InstallStartResult, error) {
	if err := s.rejectInstallGate(); err != nil {
		return InstallStartResult{}, err
	}
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.store.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return s.replayIdempotent(existing, operation.TypeHermesPrerequisites, "hermes")
	}
	if err := s.rejectActiveHermesMutation(ctx, "hermes"); err != nil {
		return s.replayIfCurrentKey(ctx, idempotencyKey, operation.TypeHermesPrerequisites, "hermes", err)
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
		Type:           operation.TypeHermesPrerequisites,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  correlation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.persistCreate(ctx, created); err != nil {
		return s.recoverCreate(ctx, created, err)
	}
	if s.prereq != nil {
		workerCtx := s.bindWorker(context.Background(), created.ID)
		go s.executePrerequisites(workerCtx, created)
	}
	return InstallStartResult{Operation: created, Created: true}, nil
}

func (s *RuntimeInstall) executePrerequisites(ctx context.Context, started operation.Operation) {
	defer s.stopWorker(started.ID)
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
	}
	now := s.now()
	running := started
	running.Status = operation.StatusRunning
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.persistUpdate(ctx, started, running); err != nil {
		return
	}
	current := running
	report := func(stage operation.Stage, _ string) {
		updated := current
		updated.Stage = stage
		updated.UpdatedAt = s.now()
		if err := s.persistUpdate(context.Background(), current, updated); err == nil {
			current = updated
		}
	}
	if s.prereq == nil {
		fail(yorvaruntime.ErrorHermesNodeMissing, false)
		return
	}
	if err := s.prereq.Apply(ctx, started.ID, report); err != nil {
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
			return
		}
		code := installErrorCodeOr(err, yorvaruntime.ErrorHermesNodeDepsFailed)
		fail(code, code == yorvaruntime.ErrorHermesNodeDepsTimeout || code == yorvaruntime.ErrorHermesNodeMissing)
		return
	}
	now = s.now()
	succeeded := current
	succeeded.Status = operation.StatusSucceeded
	succeeded.CompletedAt = &now
	succeeded.UpdatedAt = now
	_ = s.persistUpdate(context.Background(), current, succeeded)
}
