package app

import (
	"context"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const modelValidationOperationTimeout = 45 * time.Second

func (s *InstanceInventory) StartModelValidation(ctx context.Context, instanceID, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		if existing.Type != operation.TypeModelValidate || existing.TargetType != operation.TargetInstance || existing.TargetID != instanceID {
			return InstallStartResult{}, yorvaruntime.ErrInstanceConfigConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if active, ok, err := s.db.ActiveModelValidation(ctx, instanceID); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{}, instanceConfigConflict(active.ID)
	}
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, true)
	if err != nil {
		return InstallStartResult{}, err
	}
	defer unlock()
	configuration, err := models.ReadModelConfig(ctx, installation, row.NativeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	if configuration.State != yorvaruntime.ModelConfigurationConfigured || configuration.ProviderPresetID == "" || configuration.ModelID == "" {
		return InstallStartResult{}, yorvaruntime.ErrModelCredentialRequired
	}
	if active, ok, err := s.db.ActiveModelValidation(ctx, instanceID); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{}, instanceConfigConflict(active.ID)
	}
	now := s.now()
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, err
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, err
	}
	created := operation.Operation{
		ID: id, Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: instanceID,
		Status: operation.StatusPending, Stage: operation.StagePreflight, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, SourcePin: modelConfigFingerprint(configuration.ProviderPresetID, configuration.ModelID), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, created); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil && ok && existing.Type == operation.TypeModelValidate && existing.TargetID == instanceID {
				return InstallStartResult{Operation: existing}, nil
			}
			return InstallStartResult{}, yorvaruntime.ErrInstanceConfigConflict
		}
		return InstallStartResult{}, err
	}
	s.emitValidationOperation(operation.Operation{}, created, true)
	s.startModelValidationWorker(created)
	return InstallStartResult{Operation: created, Created: true}, nil
}

func (s *InstanceInventory) CancelModelValidation(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, err
	}
	if current.Type != operation.TypeModelValidate {
		return operation.Operation{}, ErrInstanceNotFound
	}
	if operation.IsTerminal(current.Status) {
		return current, nil
	}
	s.mu.Lock()
	cancel := s.workers[operationID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return s.cancelModelValidationOperation(ctx, current)
}

func (s *InstanceInventory) cancelModelValidationOperation(ctx context.Context, current operation.Operation) (operation.Operation, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if operation.IsTerminal(current.Status) {
			return current, nil
		}
		now := s.now()
		next := current
		next.Status = operation.StatusCancelled
		next.ErrorCode = yorvaruntime.ErrorModelValidationCancelled
		next.Retryable = true
		next.CompletedAt = &now
		next.UpdatedAt = now
		if err := s.db.UpdateOperation(ctx, current, next); err == nil {
			s.emitValidationOperation(current, next, false)
			return next, nil
		} else if !errors.Is(err, sqlite.ErrInvalidStatusTransition) {
			return operation.Operation{}, err
		}
		latest, err := s.db.GetOperation(ctx, current.ID)
		if err != nil {
			return operation.Operation{}, err
		}
		current = latest
	}
	return operation.Operation{}, sqlite.ErrInvalidStatusTransition
}

func (s *InstanceInventory) startModelValidationWorker(created operation.Operation) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[created.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, created.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runModelValidation(ctx, created)
	}()
}

func (s *InstanceInventory) runModelValidation(parent context.Context, created operation.Operation) {
	current, err := s.db.GetOperation(parent, created.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.Stage = operation.StageModelValidate
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(parent, current, running); err != nil {
		return
	}
	s.emitValidationOperation(current, running, false)

	ctx, cancel := context.WithTimeout(parent, modelValidationOperationTimeout)
	defer cancel()
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, created.TargetID, true)
	if err != nil {
		s.finishModelValidation(created.ID, validationResultForError(ctx, err))
		return
	}
	configuration, err := models.ReadModelConfig(ctx, installation, row.NativeID)
	if err != nil {
		unlock()
		s.finishModelValidation(created.ID, validationResultForError(ctx, err))
		return
	}
	if configuration.State != yorvaruntime.ModelConfigurationConfigured || configuration.ProviderPresetID == "" || configuration.ModelID == "" {
		unlock()
		s.finishModelValidation(created.ID, yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationFailed, ErrorCode: yorvaruntime.ErrorModelValidationFailed})
		return
	}
	if modelConfigFingerprint(configuration.ProviderPresetID, configuration.ModelID) != created.SourcePin {
		unlock()
		s.finishModelValidation(created.ID, yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorInstanceConfigConflict})
		return
	}
	unlock()
	result := models.ValidateModel(ctx, installation, row.NativeID, configuration.ProviderPresetID, configuration.ModelID)
	s.finishModelValidation(created.ID, result)
}

func (s *InstanceInventory) finishModelValidation(operationID string, result yorvaruntime.ModelValidationResult) {
	ctx := context.Background()
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	next := current
	next.Status = operation.StatusFailed
	next.ErrorCode = result.ErrorCode
	next.Retryable = result.State == yorvaruntime.ModelValidationUnknown
	if result.State == yorvaruntime.ModelValidationPassed {
		next.Status = operation.StatusSucceeded
		next.ErrorCode = ""
		next.Retryable = false
	}
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err == nil {
		s.emitValidationOperation(current, next, false)
	}
}

func validationResultForError(ctx context.Context, err error) yorvaruntime.ModelValidationResult {
	switch {
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationCancelled}
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationTimedOut}
	case errors.Is(err, yorvaruntime.ErrModelCredentialRequired):
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationFailed, ErrorCode: yorvaruntime.ErrorModelValidationFailed}
	default:
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: yorvaruntime.ErrorModelValidationUnknown}
	}
}

func (s *InstanceInventory) RecoverModelValidations(ctx context.Context) ([]operation.Operation, error) {
	active, err := s.db.ListActiveModelValidations(ctx)
	if err != nil {
		return nil, err
	}
	recovered := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		now := s.now()
		next := current
		next.Status = operation.StatusFailed
		next.ErrorCode = yorvaruntime.ErrorOperationInterrupted
		next.Retryable = true
		next.CompletedAt = &now
		next.UpdatedAt = now
		if err := s.db.UpdateOperation(ctx, current, next); err != nil {
			return recovered, err
		}
		s.emitValidationOperation(current, next, false)
		recovered = append(recovered, next)
	}
	return recovered, nil
}

func (s *InstanceInventory) emitValidationOperation(previous, next operation.Operation, created bool) {
	if s == nil || s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType == "" {
		return
	}
	payload := events.OperationPayload{
		OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage), CorrelationID: next.CorrelationID,
	}
	if next.ErrorCode != "" {
		payload.ErrorCode = string(next.ErrorCode)
	}
	s.events.Publish(events.NewOperationEvent(eventType, payload, s.now()))
}

func instanceConfigConflict(string) error {
	return yorvaruntime.ErrInstanceConfigConflict
}
