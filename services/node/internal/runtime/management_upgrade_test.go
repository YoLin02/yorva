package runtime

import (
	"testing"
	"time"
)

func TestUpgradePlanSeparatesPlanningEvidenceFromMutationQualification(t *testing.T) {
	plan := UpgradePlan{
		State: UpgradeAvailable, Rollback: RollbackEligible,
		CurrentVersion: "0.20.2", TargetVersion: "0.20.5",
		Managed: true, InventoryComplete: true,
		ProtectionPointRequired: true, ProtectionPointReady: true,
		PlanEvidenceComplete: true,
		ObservedAt:           time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	if plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("unqualified plan became executable: %#v", plan)
	}

	plan.UpgradeMutationQualified = true
	if !plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("qualification truth was conflated: %#v", plan)
	}
	plan.RollbackMutationQualified = true
	if !plan.UpgradeExecutable() || !plan.RollbackExecutable() {
		t.Fatalf("qualified plan did not become executable: %#v", plan)
	}
}

func TestUpgradePlanRejectsQualificationWithoutCompleteEvidence(t *testing.T) {
	base := UpgradePlan{
		State: UpgradeBlocked, Rollback: RollbackIneligible,
		CurrentVersion: "0.20.2", TargetVersion: "0.20.5",
		ObservedAt: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	}
	for name, mutate := range map[string]func(*UpgradePlan){
		"upgrade":  func(plan *UpgradePlan) { plan.UpgradeMutationQualified = true },
		"rollback": func(plan *UpgradePlan) { plan.RollbackMutationQualified = true },
	} {
		t.Run(name, func(t *testing.T) {
			plan := base
			mutate(&plan)
			if err := plan.Validate(); err == nil {
				t.Fatalf("invalid qualification accepted: %#v", plan)
			}
		})
	}
}

func TestUpgradePlanRejectsAvailableStateWithoutCompleteEvidence(t *testing.T) {
	plan := UpgradePlan{
		State: UpgradeAvailable, Rollback: RollbackEligible,
		CurrentVersion: "0.20.2", TargetVersion: "0.20.5",
		Managed: true, InventoryComplete: true,
		ObservedAt: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	}
	if err := plan.Validate(); err == nil {
		t.Fatalf("AVAILABLE without complete planning evidence accepted: %#v", plan)
	}
}
