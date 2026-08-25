package upgrademanagement

type UpgradeAvailability string

const (
	UpgradeAvailable UpgradeAvailability = "AVAILABLE"
	UpgradeUpToDate  UpgradeAvailability = "UP_TO_DATE"
	UpgradeBlocked   UpgradeAvailability = "BLOCKED"
	UpgradeUnknown   UpgradeAvailability = "UNKNOWN"
)

// ReasonCode is stable policy data. UI and API integration must not infer a
// decision by matching human-readable error text.
type ReasonCode string

const (
	ReasonManagedUnknown               ReasonCode = "MANAGED_ELIGIBILITY_UNKNOWN"
	ReasonUnmanaged                    ReasonCode = "INSTALLATION_UNMANAGED"
	ReasonActivePointerUnknown         ReasonCode = "ACTIVE_POINTER_UNKNOWN"
	ReasonActivePointerNotValid        ReasonCode = "ACTIVE_POINTER_NOT_VALID"
	ReasonCurrentSealUnknown           ReasonCode = "CURRENT_SEAL_UNKNOWN"
	ReasonCurrentSealInvalid           ReasonCode = "CURRENT_SEAL_INVALID"
	ReasonCurrentIdentityInvalid       ReasonCode = "CURRENT_IDENTITY_INVALID"
	ReasonActiveGenerationMismatch     ReasonCode = "ACTIVE_GENERATION_MISMATCH"
	ReasonTargetUnknown                ReasonCode = "PACKAGED_TARGET_UNKNOWN"
	ReasonTargetIdentityInvalid        ReasonCode = "PACKAGED_TARGET_IDENTITY_INVALID"
	ReasonTargetIdentityMismatch       ReasonCode = "PACKAGED_TARGET_IDENTITY_MISMATCH"
	ReasonTargetAlreadyActive          ReasonCode = "TARGET_ALREADY_ACTIVE"
	ReasonInventoryUnknown             ReasonCode = "INVENTORY_UNKNOWN"
	ReasonInventoryIncomplete          ReasonCode = "INVENTORY_INCOMPLETE"
	ReasonProtectionPointUnknown       ReasonCode = "PROTECTION_POINT_UNKNOWN"
	ReasonProtectionPointUnavailable   ReasonCode = "PROTECTION_POINT_UNAVAILABLE"
	ReasonCompatibilityUnknown         ReasonCode = "COMPATIBILITY_RECORD_UNKNOWN"
	ReasonCompatibilityMissing         ReasonCode = "COMPATIBILITY_RECORD_MISSING"
	ReasonCompatibilityInvalid         ReasonCode = "COMPATIBILITY_RECORD_INVALID"
	ReasonCompatibilityPairMismatch    ReasonCode = "COMPATIBILITY_PAIR_MISMATCH"
	ReasonUpgradeCompatibilityUnknown  ReasonCode = "UPGRADE_COMPATIBILITY_UNKNOWN"
	ReasonUpgradeCompatibilityUnsafe   ReasonCode = "UPGRADE_COMPATIBILITY_UNSAFE"
	ReasonRollbackCompatibilityUnknown ReasonCode = "ROLLBACK_COMPATIBILITY_UNKNOWN"
	ReasonRollbackCompatibilityUnsafe  ReasonCode = "ROLLBACK_COMPATIBILITY_UNSAFE"
	ReasonPostchecksUnknown            ReasonCode = "POSTCHECK_QUALIFICATION_UNKNOWN"
	ReasonPostchecksUnqualified        ReasonCode = "REQUIRED_POSTCHECK_UNQUALIFIED"
	ReasonPreviousGenerationUnknown    ReasonCode = "PREVIOUS_GENERATION_UNKNOWN"
	ReasonPreviousGenerationMissing    ReasonCode = "PREVIOUS_GENERATION_MISSING"
	ReasonPreviousGenerationInvalid    ReasonCode = "PREVIOUS_GENERATION_INVALID"
	ReasonPreviousGenerationSame       ReasonCode = "PREVIOUS_GENERATION_EQUALS_ACTIVE"
	ReasonProtectionRestoreUnknown     ReasonCode = "PROTECTION_RESTORE_UNKNOWN"
	ReasonProtectionRestoreIncomplete  ReasonCode = "PROTECTION_RESTORE_INCOMPLETE"
)

type UpgradePlanInput struct {
	Managed         ManagedState
	Active          ActivePointerObservation
	Current         SealedGenerationObservation
	Target          TargetObservation
	Inventory       InventoryObservation
	ProtectionPoint ProtectionPointObservation
	Compatibility   CompatibilityRecord
	PostcheckPolicy PostcheckQualification
}

// UpgradePlan is planning evidence, not mutation authority. PlanEvidenceComplete
// means the supplied planning evidence is complete and mutually consistent.
// Mutation qualification remains separate and false until exact destructive-flow
// qualification is accepted and wired into the production adapter.
type UpgradePlan struct {
	Availability              UpgradeAvailability
	Current                   SnapshotIdentity
	Target                    SnapshotIdentity
	PlanEvidenceComplete      bool
	RollbackEligible          bool
	UpgradeMutationQualified  bool
	RollbackMutationQualified bool
	Reasons                   []ReasonCode
	Conflicts                 []Conflict
	RequiredPostchecks        []Postcheck
}

func BuildUpgradePlan(input UpgradePlanInput) UpgradePlan {
	reasons := newReasons()
	unknown := false

	switch input.Managed {
	case ManagedProven:
	case ManagedNo:
		reasons.add(ReasonUnmanaged)
	default:
		reasons.add(ReasonManagedUnknown)
		unknown = true
	}

	switch input.Active.State {
	case ActivePointerValid:
	case ActivePointerUnknown:
		reasons.add(ReasonActivePointerUnknown)
		unknown = true
	default:
		reasons.add(ReasonActivePointerNotValid)
	}

	switch input.Current.State {
	case EvidenceVerified:
		if input.Current.Identity.Validate() != nil {
			reasons.add(ReasonCurrentIdentityInvalid)
		}
		if !input.Current.exact() {
			reasons.add(ReasonCurrentSealInvalid)
		}
	case EvidenceUnknown:
		reasons.add(ReasonCurrentSealUnknown)
		unknown = true
	default:
		reasons.add(ReasonCurrentSealInvalid)
	}
	if input.Active.State == ActivePointerValid && input.Current.State == EvidenceVerified && !input.Active.names(input.Current) {
		reasons.add(ReasonActiveGenerationMismatch)
	}

	targetExact := false
	switch input.Target.State {
	case EvidenceVerified:
		compiledValid := input.Target.Compiled.Validate() == nil
		observedValid := input.Target.Observed.Validate() == nil
		if !compiledValid || !observedValid {
			reasons.add(ReasonTargetIdentityInvalid)
		} else if !input.Target.Compiled.Equal(input.Target.Observed) {
			reasons.add(ReasonTargetIdentityMismatch)
		} else {
			targetExact = true
		}
	case EvidenceUnknown:
		reasons.add(ReasonTargetUnknown)
		unknown = true
	default:
		reasons.add(ReasonTargetIdentityInvalid)
	}
	if input.Managed == ManagedProven && input.Current.exact() && input.Active.names(input.Current) &&
		targetExact && input.Current.Identity.Equal(input.Target.Compiled) {
		return UpgradePlan{
			Availability:              UpgradeUpToDate,
			Current:                   input.Current.Identity,
			Target:                    input.Target.Compiled,
			UpgradeMutationQualified:  false,
			RollbackMutationQualified: false,
			Reasons:                   []ReasonCode{ReasonTargetAlreadyActive},
			Conflicts:                 conflicts(),
			RequiredPostchecks:        requiredPostchecks(input.Inventory.Features),
		}
	}

	features := input.Inventory.Features
	switch input.Inventory.State {
	case EvidenceVerified:
	case EvidenceUnknown:
		reasons.add(ReasonInventoryUnknown)
		unknown = true
	default:
		reasons.add(ReasonInventoryIncomplete)
	}
	required := requiredPostchecks(features)

	switch input.ProtectionPoint.State {
	case EvidenceVerified:
		if !closedIDPattern.MatchString(input.ProtectionPoint.ID) {
			reasons.add(ReasonProtectionPointUnavailable)
		}
	case EvidenceUnknown:
		reasons.add(ReasonProtectionPointUnknown)
		unknown = true
	default:
		reasons.add(ReasonProtectionPointUnavailable)
	}

	compatibilityValid, compatibilityUnknown := assessCompatibility(
		input.Compatibility,
		input.Current.Identity,
		input.Target.Compiled,
		reasons,
		true,
	)
	unknown = unknown || compatibilityUnknown

	switch input.PostcheckPolicy.State {
	case EvidenceUnknown:
		reasons.add(ReasonPostchecksUnknown)
		unknown = true
	case EvidenceVerified:
		if !postchecksQualified(input.PostcheckPolicy, required) {
			reasons.add(ReasonPostchecksUnqualified)
		}
	default:
		reasons.add(ReasonPostchecksUnqualified)
	}

	plan := UpgradePlan{
		Current:                   input.Current.Identity,
		Target:                    input.Target.Compiled,
		UpgradeMutationQualified:  false,
		RollbackMutationQualified: false,
		Reasons:                   reasons.values(),
		Conflicts:                 conflicts(),
		RequiredPostchecks:        required,
	}

	if len(plan.Reasons) == 0 {
		plan.Availability = UpgradeAvailable
		plan.PlanEvidenceComplete = true
		plan.RollbackEligible = compatibilityValid
		return plan
	}
	if unknown {
		plan.Availability = UpgradeUnknown
	} else {
		plan.Availability = UpgradeBlocked
	}
	return plan
}

func (p UpgradePlan) UpgradeExecutable() bool {
	return p.PlanEvidenceComplete && p.UpgradeMutationQualified
}

func (p UpgradePlan) RollbackExecutable() bool {
	return p.PlanEvidenceComplete && p.RollbackEligible && p.RollbackMutationQualified
}

func assessCompatibility(record CompatibilityRecord, from, to SnapshotIdentity, reasons *reasonSet, requireUpgrade bool) (bool, bool) {
	unknown := false
	switch record.State {
	case EvidenceVerified:
		if !closedIDPattern.MatchString(record.ID) ||
			!closedIDPattern.MatchString(record.WindowsEvidenceID) ||
			record.From.Validate() != nil || record.To.Validate() != nil ||
			record.FromConfigSchema <= 0 || record.ToConfigSchema <= 0 ||
			!record.UserDataInventoryComplete || !record.Migration.valid() ||
			!record.Upgrade.valid() || !record.Rollback.valid() {
			reasons.add(ReasonCompatibilityInvalid)
			return false, false
		}
		if !record.From.Equal(from) || !record.To.Equal(to) {
			reasons.add(ReasonCompatibilityPairMismatch)
		}
	case EvidenceMissing:
		reasons.add(ReasonCompatibilityMissing)
		return false, false
	case EvidenceUnknown:
		reasons.add(ReasonCompatibilityUnknown)
		return false, true
	default:
		reasons.add(ReasonCompatibilityInvalid)
		return false, false
	}

	if requireUpgrade {
		switch record.Upgrade {
		case CompatibilityProven:
		case CompatibilityUnknown:
			reasons.add(ReasonUpgradeCompatibilityUnknown)
			unknown = true
		case CompatibilityUnsafe:
			reasons.add(ReasonUpgradeCompatibilityUnsafe)
		default:
			reasons.add(ReasonCompatibilityInvalid)
		}
	}
	switch record.Rollback {
	case CompatibilityProven:
	case CompatibilityUnknown:
		reasons.add(ReasonRollbackCompatibilityUnknown)
		unknown = true
	case CompatibilityUnsafe:
		reasons.add(ReasonRollbackCompatibilityUnsafe)
	default:
		reasons.add(ReasonCompatibilityInvalid)
	}
	return len(reasons.values()) == 0, unknown
}

type reasonSet struct {
	seen        map[ReasonCode]struct{}
	valuesSlice []ReasonCode
}

func newReasons() *reasonSet {
	return &reasonSet{seen: make(map[ReasonCode]struct{})}
}

func (r *reasonSet) add(reason ReasonCode) {
	if _, exists := r.seen[reason]; exists {
		return
	}
	r.seen[reason] = struct{}{}
	r.valuesSlice = append(r.valuesSlice, reason)
}

func (r *reasonSet) values() []ReasonCode {
	return append([]ReasonCode(nil), r.valuesSlice...)
}
