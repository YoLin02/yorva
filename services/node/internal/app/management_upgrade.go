package app

import (
	"context"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// ManagementUpgrade owns the Runtime-neutral, read-only managed Upgrade plan
// boundary. Runtime-specific identity, seal, compatibility and protection-point
// inspection remain behind the selected Runtime adapter.
type ManagementUpgrade struct {
	targets RuntimeManagementTargetResolver
}

func NewManagementUpgrade(targets RuntimeManagementTargetResolver) *ManagementUpgrade {
	return &ManagementUpgrade{targets: targets}
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
	if !plan.Executable() {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Upgrade.UpgradeRuntime(ctx, target.Installation, progress)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if result.Validate() != nil {
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
	if !rollbackPlanExecutable(plan) {
		return yorvaruntime.UpgradeResult{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Rollback.RollbackRuntime(ctx, target.Installation, progress)
	if err != nil {
		return yorvaruntime.UpgradeResult{}, managementQueryError(ctx, err)
	}
	if result.Validate() != nil {
		return yorvaruntime.UpgradeResult{}, ErrManagementQueryFailed
	}
	return result, nil
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
		// AVAILABLE is reserved for a fully gated exact plan. A missing
		// protection point, incomplete inventory, or unproven rollback must be
		// BLOCKED/UNKNOWN rather than an enabled-looking partial plan.
		if !plan.Executable() {
			return false
		}
	case yorvaruntime.UpgradeUpToDate:
		if !plan.Managed || plan.CurrentVersion == "" || plan.TargetVersion == "" || plan.CurrentVersion != plan.TargetVersion {
			return false
		}
	case yorvaruntime.UpgradeBlocked, yorvaruntime.UpgradeUnknown:
		if plan.Executable() {
			return false
		}
	default:
		return false
	}

	if plan.Rollback == yorvaruntime.RollbackEligible && !rollbackPlanExecutable(plan) {
		return false
	}
	return true
}

func rollbackPlanExecutable(plan yorvaruntime.UpgradePlan) bool {
	if plan.State != yorvaruntime.UpgradeAvailable && plan.State != yorvaruntime.UpgradeUpToDate {
		return false
	}
	return plan.Managed &&
		plan.InventoryComplete &&
		plan.Rollback == yorvaruntime.RollbackEligible &&
		(!plan.ProtectionPointRequired || plan.ProtectionPointReady)
}
