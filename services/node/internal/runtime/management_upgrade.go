package runtime

import (
	"context"
	"time"
)

type UpgradeAvailabilityState string

const (
	UpgradeUpToDate  UpgradeAvailabilityState = "UP_TO_DATE"
	UpgradeAvailable UpgradeAvailabilityState = "AVAILABLE"
	UpgradeBlocked   UpgradeAvailabilityState = "BLOCKED"
	UpgradeUnknown   UpgradeAvailabilityState = "UNKNOWN"
)

func (s UpgradeAvailabilityState) Valid() bool {
	return s == UpgradeUpToDate || s == UpgradeAvailable || s == UpgradeBlocked || s == UpgradeUnknown
}

type RollbackEligibilityState string

const (
	RollbackEligible   RollbackEligibilityState = "ELIGIBLE"
	RollbackIneligible RollbackEligibilityState = "INELIGIBLE"
	RollbackUnknown    RollbackEligibilityState = "UNKNOWN"
)

func (s RollbackEligibilityState) Valid() bool {
	return s == RollbackEligible || s == RollbackIneligible || s == RollbackUnknown
}

type UpgradePlan struct {
	State                     UpgradeAvailabilityState
	Rollback                  RollbackEligibilityState
	CurrentVersion            string
	TargetVersion             string
	Managed                   bool
	InventoryComplete         bool
	ProtectionPointRequired   bool
	ProtectionPointReady      bool
	PlanEvidenceComplete      bool
	UpgradeMutationQualified  bool
	RollbackMutationQualified bool
	ObservedAt                time.Time
}

func (p UpgradePlan) Validate() error {
	if !p.State.Valid() || !p.Rollback.Valid() || p.ObservedAt.IsZero() {
		return ErrInvalidManagementContract
	}
	if err := validateBoundedText("current Runtime version", p.CurrentVersion); err != nil {
		return err
	}
	if err := validateBoundedText("target Runtime version", p.TargetVersion); err != nil {
		return err
	}
	if p.State == UpgradeAvailable && (p.CurrentVersion == "" || p.TargetVersion == "") {
		return ErrInvalidManagementContract
	}
	if p.State == UpgradeAvailable && !p.PlanEvidenceComplete {
		return ErrInvalidManagementContract
	}
	if p.PlanEvidenceComplete {
		if p.State != UpgradeAvailable || !p.Managed || !p.InventoryComplete || p.Rollback != RollbackEligible ||
			(p.ProtectionPointRequired && !p.ProtectionPointReady) {
			return ErrInvalidManagementContract
		}
	}
	if p.UpgradeMutationQualified && !p.PlanEvidenceComplete {
		return ErrInvalidManagementContract
	}
	if p.RollbackMutationQualified && (!p.PlanEvidenceComplete || p.Rollback != RollbackEligible) {
		return ErrInvalidManagementContract
	}
	return nil
}

// UpgradeExecutable is mutation truth, not merely planning completeness.
func (p UpgradePlan) UpgradeExecutable() bool {
	return p.PlanEvidenceComplete &&
		p.UpgradeMutationQualified &&
		p.State == UpgradeAvailable &&
		p.Managed &&
		p.InventoryComplete &&
		p.Rollback == RollbackEligible &&
		(!p.ProtectionPointRequired || p.ProtectionPointReady)
}

func (p UpgradePlan) RollbackExecutable() bool {
	return p.PlanEvidenceComplete &&
		p.RollbackMutationQualified &&
		p.State == UpgradeAvailable &&
		p.Managed &&
		p.InventoryComplete &&
		p.Rollback == RollbackEligible &&
		(!p.ProtectionPointRequired || p.ProtectionPointReady)
}

type UpgradeOutcomeState string

const (
	UpgradeSucceeded        UpgradeOutcomeState = "SUCCEEDED"
	UpgradeRolledBack       UpgradeOutcomeState = "ROLLED_BACK"
	UpgradeRecoveryRequired UpgradeOutcomeState = "RECOVERY_REQUIRED"
	UpgradeOutcomeUnknown   UpgradeOutcomeState = "UNKNOWN"
)

func (s UpgradeOutcomeState) Valid() bool {
	return s == UpgradeSucceeded || s == UpgradeRolledBack || s == UpgradeRecoveryRequired || s == UpgradeOutcomeUnknown
}

type UpgradeResult struct {
	State      UpgradeOutcomeState
	ObservedAt time.Time
}

func (r UpgradeResult) Validate() error {
	if !r.State.Valid() || r.ObservedAt.IsZero() {
		return ErrInvalidManagementContract
	}
	return nil
}

type UpgradePlanner interface {
	PlanUpgrade(context.Context, Installation) (UpgradePlan, error)
}

// RuntimeUpgrader chooses the exact adapter-owned packaged target. The caller
// cannot supply a version, source, branch, path, command, or force option.
type RuntimeUpgrader interface {
	UpgradeRuntime(context.Context, Installation, ProgressSink) (UpgradeResult, error)
}

type RuntimeRollbacker interface {
	RollbackRuntime(context.Context, Installation, ProgressSink) (UpgradeResult, error)
}
