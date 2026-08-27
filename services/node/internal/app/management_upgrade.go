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

var ErrRuntimeMutationConflict = errors.New("another Runtime mutation is active")

// ManagementUpgrade owns the Runtime-neutral, read-only managed Upgrade plan
// boundary. Runtime-specific identity, seal, compatibility and protection-point
// inspection remain behind the selected Runtime adapter.
type ManagementUpgrade struct {
	targets RuntimeManagementTargetResolver
	db      *sqlite.Database
	events  *events.Broker
	now     func() time.Time
	newID   func() (string, error)
}

func NewManagementUpgrade(targets RuntimeManagementTargetResolver) *ManagementUpgrade {
	return &ManagementUpgrade{targets: targets, now: time.Now, newID: newOperationID}
}

func (s *InstanceInventory) NewManagementUpgrade() (*ManagementUpgrade, error) {
	if s == nil || s.db == nil {
		return nil, ErrManagementQueryFailed
	}
	service := NewManagementUpgrade(s)
	service.db, service.events = s.db, s.events
	if _, err := service.RecoverInterrupted(context.Background()); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *ManagementUpgrade) PlanUpgrade(ctx context.Context, runtimeID string) (yorvaruntime.UpgradePlan, error) {
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return yorvaruntime.UpgradePlan{}, err
	}
	if target.Bundle.UpgradePlan == nil {
		return yorvaruntime.UpgradePlan{}, ErrManagementCapabilityUnsupported
	}

	plan, err := target.Bundle.UpgradePlan.PlanUpgrade(ctx, target.Installation)
	if err != nil {
		return yorvaruntime.UpgradePlan{}, managementQueryError(ctx, err)
	}
	if !validManagedUpgradePlan(plan, target.Installation) {
		return yorvaruntime.UpgradePlan{}, ErrManagementQueryFailed
	}
	return plan, nil
}

// ExecuteUpgradeOperation is reserved for a durable Operation worker and is
// intentionally not exposed by HTTP. Compile-time Bundle wiring represents
// accepted capability qualification; the worker still re-plans immediately
// before invoking the adapter.
func (s *ManagementUpgrade) ExecuteUpgradeOperation(ctx context.Context, runtimeID string, progress yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, err
	}
	if target.Bundle.UpgradePlan == nil || target.Bundle.Upgrade == nil {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}
	plan, err := target.Bundle.UpgradePlan.PlanUpgrade(ctx, target.Installation)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if !validManagedUpgradePlan(plan, target.Installation) {
		return yorvaruntime.UpgradeResult{}, ErrManagementQueryFailed
	}
	if !plan.UpgradeExecutable() {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Upgrade.UpgradeRuntime(ctx, target.Installation, progress)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if result.Validate() != nil || result.State != yorvaruntime.UpgradeSucceeded {
		return yorvaruntime.UpgradeResult{}, ErrManagementQueryFailed
	}
	return result, nil
}

// ExecuteRollbackOperation is reserved for a durable Operation worker and is
// intentionally not exposed by HTTP.
func (s *ManagementUpgrade) ExecuteRollbackOperation(ctx context.Context, runtimeID string, progress yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, err
	}
	if target.Bundle.UpgradePlan == nil || target.Bundle.Rollback == nil {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}
	plan, err := target.Bundle.UpgradePlan.PlanUpgrade(ctx, target.Installation)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if !validManagedUpgradePlan(plan, target.Installation) {
		return yorvaruntime.UpgradeResult{}, ErrManagementQueryFailed
	}
	if !plan.RollbackExecutable() {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Rollback.RollbackRuntime(ctx, target.Installation, progress)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if result.Validate() != nil || result.State != yorvaruntime.UpgradeRolledBack {
		return yorvaruntime.UpgradeResult{}, ErrManagementQueryFailed
	}
	return result, nil
}

func (s *ManagementUpgrade) StartUpgrade(ctx context.Context, runtimeID, key string) (InstallStartResult, error) {
	return s.startMutation(ctx, runtimeID, key, false)
}

func (s *ManagementUpgrade) StartRollback(ctx context.Context, runtimeID, key string) (InstallStartResult, error) {
	return s.startMutation(ctx, runtimeID, key, true)
}

func (s *ManagementUpgrade) startMutation(ctx context.Context, runtimeID, key string, rollback bool) (InstallStartResult, error) {
	if s == nil || s.db == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if err := ValidateIdempotencyKey(key); err != nil {
		return InstallStartResult{}, err
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	unlock := lockRuntimeManagementTarget(s.targets, target.InstallationID)
	defer unlock()
	if target.Bundle.UpgradePlan == nil || (!rollback && target.Bundle.Upgrade == nil) || (rollback && target.Bundle.Rollback == nil) {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	plan, err := target.Bundle.UpgradePlan.PlanUpgrade(ctx, target.Installation)
	if err != nil || !validManagedUpgradePlan(plan, target.Installation) || (!rollback && !plan.UpgradeExecutable()) || (rollback && !plan.RollbackExecutable()) {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	opType := operation.TypeRuntimeUpgrade
	if rollback {
		opType = operation.TypeRuntimeRollback
	}
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, key); queryErr != nil {
		return InstallStartResult{}, managementQueryError(ctx, queryErr)
	} else if ok {
		if existing.Type != opType || existing.TargetID != target.InstallationID {
			return InstallStartResult{}, ErrRuntimeMutationConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now().UTC()
	op := operation.Operation{
		ID: id, Type: opType, TargetType: operation.TargetRuntimeInstallation, TargetID: target.InstallationID,
		Status: operation.StatusPending, Stage: operation.StageUpgradePreflight,
		IdempotencyKey: key, CorrelationID: correlationID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrRuntimeMutationConflict
		}
		return InstallStartResult{}, managementQueryError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	go s.runMutation(op, target, rollback)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *ManagementUpgrade) runMutation(op operation.Operation, target RuntimeManagementTarget, rollback bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now().UTC()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageUpgradeBuild, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	var result yorvaruntime.UpgradeResult
	plan, planErr := target.Bundle.UpgradePlan.PlanUpgrade(ctx, target.Installation)
	if planErr != nil || !validManagedUpgradePlan(plan, target.Installation) ||
		(!rollback && !plan.UpgradeExecutable()) || (rollback && !plan.RollbackExecutable()) {
		err = ErrManagementCapabilityUnsupported
	} else if rollback {
		result, err = target.Bundle.Rollback.RollbackRuntime(ctx, target.Installation, nil)
	} else {
		result, err = target.Bundle.Upgrade.UpgradeRuntime(ctx, target.Installation, nil)
	}
	completed := s.now().UTC()
	final := running
	final.Stage, final.CompletedAt, final.UpdatedAt = operation.StageUpgradeReconcile, &completed, completed
	if result.State.Valid() {
		final.Message = string(result.State)
	}
	expectedState := yorvaruntime.UpgradeSucceeded
	if rollback {
		expectedState = yorvaruntime.UpgradeRolledBack
	}
	if err != nil || result.Validate() != nil || result.State != expectedState {
		final.Status, final.Retryable = operation.StatusFailed, result.State != yorvaruntime.UpgradeRecoveryRequired
		if rollback {
			final.ErrorCode = yorvaruntime.ErrorRuntimeRollbackFailed
		} else {
			final.ErrorCode = yorvaruntime.ErrorRuntimeUpgradeFailed
		}
	} else {
		final.Status = operation.StatusSucceeded
	}
	_ = s.persistOperation(context.Background(), running, final)
}

func (s *ManagementUpgrade) RecoverInterrupted(ctx context.Context) ([]operation.Operation, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	active, err := s.db.ListActiveRuntimeManagementOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		now := s.now().UTC()
		next := current
		next.Status, next.Stage = operation.StatusFailed, operation.StageUpgradeReconcile
		next.ErrorCode, next.Retryable, next.CompletedAt, next.UpdatedAt = yorvaruntime.ErrorOperationInterrupted, false, &now, now
		if err := s.persistOperation(ctx, current, next); err != nil {
			return result, err
		}
		result = append(result, next)
	}
	return result, nil
}

func (s *ManagementUpgrade) persistOperation(ctx context.Context, current, next operation.Operation) error {
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return err
	}
	s.emitOperation(current, next, false)
	return nil
}

func (s *ManagementUpgrade) emitOperation(previous, next operation.Operation, created bool) {
	if s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType != "" {
		s.events.Publish(events.NewOperationEvent(eventType, events.OperationPayload{
			OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage),
			ErrorCode: string(next.ErrorCode), CorrelationID: next.CorrelationID,
		}, s.now().UTC()))
	}
}

func (s *ManagementUpgrade) resolve(ctx context.Context, runtimeID string) (RuntimeManagementTarget, error) {
	if s == nil || s.targets == nil {
		return RuntimeManagementTarget{}, ErrManagementQueryFailed
	}
	if runtimeID == "" {
		return RuntimeManagementTarget{}, ErrRuntimeNotSupported
	}
	target, err := s.targets.ResolveRuntimeManagementTarget(ctx, runtimeID)
	if err != nil {
		return RuntimeManagementTarget{}, managementQueryError(ctx, err)
	}
	if target.Installation.RuntimeKind != yorvaruntime.Kind(runtimeID) ||
		target.Installation.Path == "" || target.Installation.Version == "" ||
		target.Installation.SupportState != yorvaruntime.DiscoverySupported ||
		target.Bundle.Descriptor.Kind != target.Installation.RuntimeKind {
		return RuntimeManagementTarget{}, ErrManagementQueryFailed
	}
	return target, nil
}

func validManagedUpgradePlan(plan yorvaruntime.UpgradePlan, installation yorvaruntime.Installation) bool {
	if plan.Validate() != nil || plan.CurrentVersion != installation.Version {
		return false
	}

	switch plan.State {
	case yorvaruntime.UpgradeAvailable:
		// AVAILABLE may describe complete planning evidence without granting
		// mutation authority. Qualification remains explicit and separate.
		if !plan.PlanEvidenceComplete || !managedUpgradePlanningEvidenceComplete(plan) {
			return false
		}
	case yorvaruntime.UpgradeUpToDate:
		if plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified ||
			plan.CurrentVersion == "" || plan.TargetVersion == "" || plan.CurrentVersion != plan.TargetVersion ||
			plan.Compatibility != yorvaruntime.UpgradeCompatibilityNotRequired || len(plan.Reasons) != 0 {
			return false
		}
	case yorvaruntime.UpgradeBlocked, yorvaruntime.UpgradeUnknown:
		if plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified {
			return false
		}
	default:
		return false
	}

	return true
}

func managedUpgradePlanningEvidenceComplete(plan yorvaruntime.UpgradePlan) bool {
	return plan.State == yorvaruntime.UpgradeAvailable &&
		plan.Managed &&
		plan.InventoryComplete &&
		plan.Rollback == yorvaruntime.RollbackEligible &&
		(!plan.ProtectionPointRequired || plan.ProtectionPointReady)
}
