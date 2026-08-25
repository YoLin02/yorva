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

type UpgradeCompatibilityState string

const (
	UpgradeCompatibilityNotRequired UpgradeCompatibilityState = "NOT_REQUIRED"
	UpgradeCompatibilityProven      UpgradeCompatibilityState = "PROVEN"
	UpgradeCompatibilityUnsafe      UpgradeCompatibilityState = "UNSAFE"
	UpgradeCompatibilityUnknown     UpgradeCompatibilityState = "UNKNOWN"
)

func (s UpgradeCompatibilityState) Valid() bool {
	return s == UpgradeCompatibilityNotRequired || s == UpgradeCompatibilityProven ||
		s == UpgradeCompatibilityUnsafe || s == UpgradeCompatibilityUnknown
}

// UpgradePlanReason is a closed, safe explanation for a non-actionable plan.
// It deliberately omits paths, source digests, commands and internal record IDs.
type UpgradePlanReason string

const (
	UpgradeReasonManagedEvidenceUnknown UpgradePlanReason = "MANAGED_EVIDENCE_UNKNOWN"
	UpgradeReasonCurrentIdentityUnknown UpgradePlanReason = "CURRENT_IDENTITY_UNKNOWN"
	UpgradeReasonInventoryUnknown       UpgradePlanReason = "INVENTORY_UNKNOWN"
	UpgradeReasonProtectionRequired     UpgradePlanReason = "PROTECTION_POINT_REQUIRED"
	UpgradeReasonCompatibilityUnknown   UpgradePlanReason = "COMPATIBILITY_UNKNOWN"
	UpgradeReasonPostchecksUnqualified  UpgradePlanReason = "POSTCHECKS_UNQUALIFIED"
)

func (r UpgradePlanReason) Valid() bool {
	switch r {
	case UpgradeReasonManagedEvidenceUnknown, UpgradeReasonCurrentIdentityUnknown,
		UpgradeReasonInventoryUnknown, UpgradeReasonProtectionRequired,
		UpgradeReasonCompatibilityUnknown, UpgradeReasonPostchecksUnqualified:
		return true
	default:
		return false
	}
}

type UpgradePlan struct {
	State                     UpgradeAvailabilityState
	Rollback                  RollbackEligibilityState
	CurrentVersion            string
	TargetVersion             string
	CandidateLabel            string
	Compatibility             UpgradeCompatibilityState
	Reasons                   []UpgradePlanReason
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
	if err := validateBoundedText("upgrade candidate label", p.CandidateLabel); err != nil {
		return err
	}
	if !p.Compatibility.Valid() || len(p.Reasons) > 16 {
		return ErrInvalidManagementContract
	}
	seenReasons := make(map[UpgradePlanReason]struct{}, len(p.Reasons))
	for _, reason := range p.Reasons {
		if !reason.Valid() {
			return ErrInvalidManagementContract
		}
		if _, duplicate := seenReasons[reason]; duplicate {
			return ErrInvalidManagementContract
		}
		seenReasons[reason] = struct{}{}
	}
	if p.State == UpgradeAvailable && (p.CurrentVersion == "" || p.TargetVersion == "") {
		return ErrInvalidManagementContract
	}
	if p.State == UpgradeAvailable && !p.PlanEvidenceComplete {
		return ErrInvalidManagementContract
	}
	if p.PlanEvidenceComplete {
		if p.State != UpgradeAvailable || !p.Managed || !p.InventoryComplete || p.Rollback != RollbackEligible ||
			(p.ProtectionPointRequired && !p.ProtectionPointReady) || p.Compatibility != UpgradeCompatibilityProven || len(p.Reasons) != 0 {
			return ErrInvalidManagementContract
		}
	}
	if p.State == UpgradeUpToDate && p.Compatibility != UpgradeCompatibilityNotRequired {
		return ErrInvalidManagementContract
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
