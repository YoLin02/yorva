package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type LifecycleAction string

const (
	LifecycleStart   LifecycleAction = "start"
	LifecycleStop    LifecycleAction = "stop"
	LifecycleRestart LifecycleAction = "restart"
)

type LifecycleView struct {
	State             yorvaruntime.LifecycleState
	ActiveOperationID *string
	ObservedAt        time.Time
	ErrorCode         yorvaruntime.ErrorCode
}

func (s *InstanceInventory) GetLifecycle(ctx context.Context, instanceID string) (LifecycleView, error) {
	row, manager, installation, err := s.resolveLifecycleTarget(ctx, instanceID)
	if err != nil {
		return LifecycleView{}, err
	}
	view := LifecycleView{State: yorvaruntime.LifecycleUnknown, ObservedAt: s.now()}
	if active, ok, activeErr := s.db.ActiveInstanceLifecycle(ctx, row.ID); activeErr != nil {
		return LifecycleView{}, activeErr
	} else if ok {
		view.ActiveOperationID = &active.ID
	}
	status, statusErr := manager.Status(ctx, installation, row.NativeID)
	view.State = status.State
	if statusErr != nil {
		view.State = yorvaruntime.LifecycleUnknown
		view.ErrorCode = lifecycleErrorCode(statusErr, "")
	}
	return view, nil
}

func (s *InstanceInventory) StartLifecycle(ctx context.Context, instanceID string, action LifecycleAction, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if !validLifecycleAction(action) {
		return InstallStartResult{}, ErrRuntimeNotSupported
	}
	row, _, _, err := s.resolveLifecycleTarget(ctx, instanceID)
	if err != nil {
		return InstallStartResult{}, err
	}
	opType, stage := lifecycleOperationShape(action)
	if existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); getErr != nil {
		return InstallStartResult{}, getErr
	} else if ok {
		if existing.Type != opType || existing.TargetID != row.ID {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if _, active, activeErr := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrInstanceConflict
	}
	if _, active, activeErr := s.db.ActiveInstanceLifecycle(ctx, row.ID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrInstanceConflict
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
	op := operation.Operation{
		ID: id, Type: opType, TargetType: operation.TargetInstance, TargetID: row.ID,
		Status: operation.StatusPending, Stage: operation.StagePreflight, Message: string(action),
		IdempotencyKey: idempotencyKey, CorrelationID: correlation, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{}, err
	}
	s.startLifecycleWorker(op, row.RuntimeInstallationID, row.NativeID, action, stage)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *InstanceInventory) CancelLifecycle(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, err
	}
	if !isLifecycleOperation(current.Type) {
		return operation.Operation{}, ErrInstanceNotFound
	}
	s.mu.Lock()
	started := s.started[operationID]
	cancel := s.workers[operationID]
	s.mu.Unlock()
	if started || current.Status == operation.StatusRunning {
		return current, ErrInstanceNotCancellable
	}
	if cancel != nil {
		cancel()
	}
	if operation.IsTerminal(current.Status) {
		return current, nil
	}
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return operation.Operation{}, err
	}
	return next, nil
}

func (s *InstanceInventory) startLifecycleWorker(op operation.Operation, installationID, nativeID string, action LifecycleAction, stage operation.Stage) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[op.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, op.ID)
			delete(s.started, op.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runLifecycle(ctx, op, installationID, nativeID, action, stage)
	}()
}

func (s *InstanceInventory) runLifecycle(ctx context.Context, op operation.Operation, installationID, nativeID string, action LifecycleAction, stage operation.Stage) {
	unlock := s.lockInstallation(installationID)
	defer unlock()
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.Stage = stage
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, running); err != nil {
		return
	}
	s.mu.Lock()
	s.started[op.ID] = true
	s.mu.Unlock()

	commandCtx, cancel := context.WithTimeout(ctx, 80*time.Second)
	defer cancel()
	row, manager, installation, resolveErr := s.resolveLifecycleTarget(commandCtx, op.TargetID)
	if resolveErr != nil || row.RuntimeInstallationID != installationID || row.NativeID != nativeID {
		s.failLifecycle(running, lifecycleErrorCode(resolveErr, action), true)
		return
	}
	var mutationErr error
	switch action {
	case LifecycleStart:
		mutationErr = manager.Start(commandCtx, installation, nativeID)
	case LifecycleStop:
		mutationErr = manager.Stop(commandCtx, installation, nativeID)
	case LifecycleRestart:
		mutationErr = manager.Restart(commandCtx, installation, nativeID)
	}
	if mutationErr != nil {
		s.failLifecycle(running, lifecycleErrorCode(mutationErr, action), lifecycleRetryable(mutationErr))
		return
	}
	s.succeedLifecycle(running)
}

func (s *InstanceInventory) resolveLifecycleTarget(ctx context.Context, instanceID string) (instance.Instance, yorvaruntime.LifecycleManager, yorvaruntime.LifecycleInstallation, error) {
	if s == nil || s.db == nil || s.discovery == nil || instanceID == "" {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrInstanceNotFound
	}
	if err != nil {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, err
	}
	if row.Availability != instance.Available {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrInstanceNotAvailable
	}
	detected, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil || detected.State != yorvaruntime.DiscoverySupported || detected.Selected == nil {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrRuntimeNotSupported
	}
	accepted, err := s.ensureInstallation(ctx, detected)
	if err != nil || accepted.ID != row.RuntimeInstallationID {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrInstanceNotAvailable
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(hermesRuntimeID))
	if !ok || bundle.Lifecycle == nil {
		return instance.Instance{}, nil, yorvaruntime.LifecycleInstallation{}, ErrRuntimeNotSupported
	}
	installation := yorvaruntime.LifecycleInstallation{Executable: detected.Selected.Path, Version: detected.Selected.Version}
	return row, bundle.Lifecycle, installation, nil
}

func (s *InstanceInventory) RecoverLifecycle(ctx context.Context) ([]operation.Operation, error) {
	ops, err := s.db.ListActiveLifecycleOperations(ctx)
	if err != nil {
		return nil, err
	}
	recovered := make([]operation.Operation, 0, len(ops))
	for _, op := range ops {
		current, getErr := s.db.GetOperation(ctx, op.ID)
		if getErr != nil || operation.IsTerminal(current.Status) {
			if getErr != nil {
				return recovered, getErr
			}
			continue
		}
		if current.Type == operation.TypeInstanceRestart {
			next, persistErr := s.persistRecoveredFail(ctx, current, yorvaruntime.ErrorLifecycleResultUnknown, true)
			if persistErr != nil {
				return recovered, persistErr
			}
			recovered = append(recovered, next)
			continue
		}
		view, viewErr := s.GetLifecycle(ctx, current.TargetID)
		if viewErr != nil || view.State == yorvaruntime.LifecycleUnknown {
			next, persistErr := s.persistRecoveredFail(ctx, current, yorvaruntime.ErrorLifecycleResultUnknown, true)
			if persistErr != nil {
				return recovered, persistErr
			}
			recovered = append(recovered, next)
			continue
		}
		expected := yorvaruntime.LifecycleRunning
		if current.Type == operation.TypeInstanceStop {
			expected = yorvaruntime.LifecycleStopped
		}
		var next operation.Operation
		if view.State == expected {
			next, err = s.persistRecoveredSucceed(ctx, current)
		} else {
			next, err = s.persistRecoveredFail(ctx, current, yorvaruntime.ErrorOperationInterrupted, true)
		}
		if err != nil {
			return recovered, err
		}
		recovered = append(recovered, next)
	}
	return recovered, nil
}

func (s *InstanceInventory) succeedLifecycle(current operation.Operation) {
	now := s.now()
	next := current
	next.Status = operation.StatusSucceeded
	next.Stage = operation.StageLifecycleReconcile
	next.CompletedAt = &now
	next.UpdatedAt = now
	_ = s.db.UpdateOperation(context.Background(), current, next)
}

func (s *InstanceInventory) failLifecycle(current operation.Operation, code yorvaruntime.ErrorCode, retryable bool) {
	now := s.now()
	next := current
	next.Status = operation.StatusFailed
	next.Stage = operation.StageLifecycleReconcile
	next.ErrorCode = code
	next.Retryable = retryable
	next.CompletedAt = &now
	next.UpdatedAt = now
	_ = s.db.UpdateOperation(context.Background(), current, next)
}

func validLifecycleAction(action LifecycleAction) bool {
	return action == LifecycleStart || action == LifecycleStop || action == LifecycleRestart
}

func lifecycleOperationShape(action LifecycleAction) (operation.Type, operation.Stage) {
	switch action {
	case LifecycleStop:
		return operation.TypeInstanceStop, operation.StageInstanceStop
	case LifecycleRestart:
		return operation.TypeInstanceRestart, operation.StageInstanceRestart
	default:
		return operation.TypeInstanceStart, operation.StageInstanceStart
	}
}

func isLifecycleOperation(value operation.Type) bool {
	return value == operation.TypeInstanceStart || value == operation.TypeInstanceStop || value == operation.TypeInstanceRestart
}

func lifecycleErrorCode(err error, action LifecycleAction) yorvaruntime.ErrorCode {
	switch {
	case errors.Is(err, yorvaruntime.ErrInstanceNotRunning):
		return yorvaruntime.ErrorInstanceNotRunning
	case errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized):
		return yorvaruntime.ErrorLifecycleOutputUnrecognized
	case errors.Is(err, yorvaruntime.ErrLifecyclePostcondition):
		return yorvaruntime.ErrorLifecyclePostconditionFailed
	case errors.Is(err, yorvaruntime.ErrLifecycleQueryFailed), errors.Is(err, ErrInstanceNotAvailable), errors.Is(err, ErrRuntimeNotSupported):
		return yorvaruntime.ErrorLifecycleQueryFailed
	case action == LifecycleStop:
		return yorvaruntime.ErrorLifecycleStopFailed
	case action == LifecycleRestart:
		return yorvaruntime.ErrorLifecycleRestartFailed
	default:
		return yorvaruntime.ErrorLifecycleStartFailed
	}
}

func lifecycleRetryable(err error) bool {
	return !errors.Is(err, yorvaruntime.ErrInstanceNotRunning) && !errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized)
}
