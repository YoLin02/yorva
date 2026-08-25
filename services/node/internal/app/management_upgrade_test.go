package app

import (
	"context"
	"errors"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeRuntimeManagementTargetResolver struct {
	target    RuntimeManagementTarget
	err       error
	runtimeID string
}

func (f *fakeRuntimeManagementTargetResolver) ResolveRuntimeManagementTarget(_ context.Context, runtimeID string) (RuntimeManagementTarget, error) {
	f.runtimeID = runtimeID
	return f.target, f.err
}

type fakeManagedUpgradeAdapter struct {
	plan           yorvaruntime.UpgradePlan
	planErr        error
	plans          int
	upgrades       int
	rollbacks      int
	installation   yorvaruntime.Installation
	upgradeResult  yorvaruntime.UpgradeResult
	upgradeErr     error
	rollbackResult yorvaruntime.UpgradeResult
	rollbackErr    error
}

func (f *fakeManagedUpgradeAdapter) PlanUpgrade(_ context.Context, installation yorvaruntime.Installation) (yorvaruntime.UpgradePlan, error) {
	f.plans++
	f.installation = installation
	return f.plan, f.planErr
}

func (f *fakeManagedUpgradeAdapter) UpgradeRuntime(context.Context, yorvaruntime.Installation, yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	f.upgrades++
	return f.upgradeResult, f.upgradeErr
}

func (f *fakeManagedUpgradeAdapter) RollbackRuntime(context.Context, yorvaruntime.Installation, yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	f.rollbacks++
	return f.rollbackResult, f.rollbackErr
}

func completeUnqualifiedUpgradePlan() yorvaruntime.UpgradePlan {
	return yorvaruntime.UpgradePlan{
		State:                   yorvaruntime.UpgradeAvailable,
		Rollback:                yorvaruntime.RollbackEligible,
		CurrentVersion:          "0.20.2",
		TargetVersion:           "0.20.5",
		Managed:                 true,
		InventoryComplete:       true,
		ProtectionPointRequired: true,
		ProtectionPointReady:    true,
		PlanEvidenceComplete:    true,
		ObservedAt:              time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	}
}

func qualifiedExecutableUpgradePlan() yorvaruntime.UpgradePlan {
	plan := completeUnqualifiedUpgradePlan()
	plan.UpgradeMutationQualified = true
	plan.RollbackMutationQualified = true
	return plan
}

func upgradeTarget(adapter *fakeManagedUpgradeAdapter) RuntimeManagementTarget {
	installation := yorvaruntime.Installation{
		RuntimeKind:  "hermes",
		Path:         "C:/managed/hermes.exe",
		Version:      "0.20.2",
		SupportState: yorvaruntime.DiscoverySupported,
	}
	return RuntimeManagementTarget{
		Installation:   installation,
		InstallationID: "install_1",
		Bundle: yorvaruntime.Bundle{
			Descriptor:  yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
			UpgradePlan: adapter,
			Upgrade:     adapter,
			Rollback:    adapter,
		},
	}
}

func TestManagementUpgradePlanUsesExactRuntimeTarget(t *testing.T) {
	adapter := &fakeManagedUpgradeAdapter{plan: completeUnqualifiedUpgradePlan()}
	resolver := &fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)}
	service := NewManagementUpgrade(resolver)

	plan, err := service.PlanUpgrade(context.Background(), "hermes")
	if err != nil || !plan.PlanEvidenceComplete || plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("PlanUpgrade() = %#v, %v", plan, err)
	}
	if resolver.runtimeID != "hermes" || adapter.installation != resolver.target.Installation {
		t.Fatalf("target = %q %#v", resolver.runtimeID, adapter.installation)
	}
}

func TestManagementUpgradePlanFailsClosedOnInvalidEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*yorvaruntime.UpgradePlan)
	}{
		{"invalid state", func(plan *yorvaruntime.UpgradePlan) { plan.State = "INVALID" }},
		{"current identity mismatch", func(plan *yorvaruntime.UpgradePlan) { plan.CurrentVersion = "0.20.1" }},
		{"protection point missing", func(plan *yorvaruntime.UpgradePlan) { plan.ProtectionPointReady = false }},
		{"inventory incomplete", func(plan *yorvaruntime.UpgradePlan) { plan.InventoryComplete = false }},
		{"rollback ineligible", func(plan *yorvaruntime.UpgradePlan) { plan.Rollback = yorvaruntime.RollbackIneligible }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := completeUnqualifiedUpgradePlan()
			test.mutate(&plan)
			adapter := &fakeManagedUpgradeAdapter{plan: plan}
			service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})
			if _, err := service.PlanUpgrade(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) {
				t.Fatalf("PlanUpgrade() error = %v", err)
			}
		})
	}
}

func TestManagementUpgradePlanCapabilityFalseAndAdapterErrorsAreStable(t *testing.T) {
	target := upgradeTarget(nil)
	target.Bundle.UpgradePlan = nil
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: target})
	if _, err := service.PlanUpgrade(context.Background(), "hermes"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("nil planner error = %v", err)
	}

	adapter := &fakeManagedUpgradeAdapter{planErr: errors.New("unsafe internal path C:/secret")}
	service = NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})
	if _, err := service.PlanUpgrade(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("adapter error = %v", err)
	}
}

func TestManagementUpgradeMutationWorkerBoundariesValidatePlanAndResult(t *testing.T) {
	observedAt := time.Date(2026, 8, 25, 13, 0, 0, 0, time.UTC)
	adapter := &fakeManagedUpgradeAdapter{
		plan:           qualifiedExecutableUpgradePlan(),
		upgradeResult:  yorvaruntime.UpgradeResult{State: yorvaruntime.UpgradeSucceeded, ObservedAt: observedAt},
		rollbackResult: yorvaruntime.UpgradeResult{State: yorvaruntime.UpgradeRolledBack, ObservedAt: observedAt},
	}
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})

	upgraded, err := service.ExecuteUpgradeOperation(context.Background(), "hermes", nil)
	if err != nil || upgraded.State != yorvaruntime.UpgradeSucceeded {
		t.Fatalf("ExecuteUpgradeOperation() = %#v, %v", upgraded, err)
	}
	rolledBack, err := service.ExecuteRollbackOperation(context.Background(), "hermes", nil)
	if err != nil || rolledBack.State != yorvaruntime.UpgradeRolledBack {
		t.Fatalf("ExecuteRollbackOperation() = %#v, %v", rolledBack, err)
	}
	if adapter.plans != 2 || adapter.upgrades != 1 || adapter.rollbacks != 1 {
		t.Fatalf("adapter calls = plans %d upgrades %d rollbacks %d", adapter.plans, adapter.upgrades, adapter.rollbacks)
	}
}

func TestManagementUpgradeMutationWorkerRejectsInvalidAndFailedResults(t *testing.T) {
	adapter := &fakeManagedUpgradeAdapter{plan: qualifiedExecutableUpgradePlan()}
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})
	if _, err := service.ExecuteUpgradeOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("invalid Upgrade result error = %v", err)
	}

	adapter.rollbackErr = errors.New("unsafe internal rollback error")
	if _, err := service.ExecuteRollbackOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("failed Rollback result error = %v", err)
	}
}

func TestManagementUpgradeMutationWorkerRequiresWiredCapabilities(t *testing.T) {
	adapter := &fakeManagedUpgradeAdapter{plan: qualifiedExecutableUpgradePlan()}
	target := upgradeTarget(adapter)
	target.Bundle.Upgrade = nil
	target.Bundle.Rollback = nil
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: target})

	if _, err := service.ExecuteUpgradeOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("missing Upgrade capability error = %v", err)
	}
	if _, err := service.ExecuteRollbackOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("missing Rollback capability error = %v", err)
	}
	if adapter.plans != 0 || adapter.upgrades != 0 || adapter.rollbacks != 0 {
		t.Fatalf("missing capability reached adapter: %#v", adapter)
	}
}

func TestManagementUpgradeMutationWorkerDoesNotRunBlockedPlan(t *testing.T) {
	plan := completeUnqualifiedUpgradePlan()
	plan.State = yorvaruntime.UpgradeBlocked
	plan.PlanEvidenceComplete = false
	plan.Rollback = yorvaruntime.RollbackIneligible
	plan.InventoryComplete = false
	plan.ProtectionPointReady = false
	adapter := &fakeManagedUpgradeAdapter{plan: plan}
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})

	if _, err := service.ExecuteUpgradeOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("blocked Upgrade error = %v", err)
	}
	if _, err := service.ExecuteRollbackOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("blocked Rollback error = %v", err)
	}
	if adapter.plans != 2 || adapter.upgrades != 0 || adapter.rollbacks != 0 {
		t.Fatalf("blocked plan reached mutation adapter: %#v", adapter)
	}
}

func TestManagementUpgradeMutationWorkerRejectsCompleteButUnqualifiedPlan(t *testing.T) {
	adapter := &fakeManagedUpgradeAdapter{plan: completeUnqualifiedUpgradePlan()}
	service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})

	if _, err := service.ExecuteUpgradeOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("unqualified Upgrade error = %v", err)
	}
	if _, err := service.ExecuteRollbackOperation(context.Background(), "hermes", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("unqualified Rollback error = %v", err)
	}
	if adapter.upgrades != 0 || adapter.rollbacks != 0 {
		t.Fatalf("unqualified plan reached mutation adapter: %#v", adapter)
	}
}

func TestManagementUpgradeBlockedAndUnknownPlansCannotCarryQualification(t *testing.T) {
	for _, state := range []yorvaruntime.UpgradeAvailabilityState{yorvaruntime.UpgradeBlocked, yorvaruntime.UpgradeUnknown} {
		t.Run(string(state), func(t *testing.T) {
			plan := completeUnqualifiedUpgradePlan()
			plan.State = state
			plan.PlanEvidenceComplete = false
			plan.Rollback = yorvaruntime.RollbackUnknown
			plan.InventoryComplete = false
			plan.ProtectionPointReady = false
			adapter := &fakeManagedUpgradeAdapter{plan: plan}
			service := NewManagementUpgrade(&fakeRuntimeManagementTargetResolver{target: upgradeTarget(adapter)})

			got, err := service.PlanUpgrade(context.Background(), "hermes")
			if err != nil {
				t.Fatalf("PlanUpgrade() error = %v", err)
			}
			if got.PlanEvidenceComplete || got.UpgradeMutationQualified || got.RollbackMutationQualified || got.UpgradeExecutable() || got.RollbackExecutable() {
				t.Fatalf("blocked/unknown plan implied qualification: %#v", got)
			}
		})
	}
}
