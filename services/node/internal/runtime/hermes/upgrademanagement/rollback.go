package upgrademanagement

type RollbackEligibility string

const (
	RollbackEligible   RollbackEligibility = "ELIGIBLE"
	RollbackIneligible RollbackEligibility = "INELIGIBLE"
	RollbackUnknown    RollbackEligibility = "UNKNOWN"
)

type RollbackPlanInput struct {
	Managed           ManagedState
	Active            ActivePointerObservation
	Current           SealedGenerationObservation
	Previous          SealedGenerationObservation
	Inventory         InventoryObservation
	Compatibility     CompatibilityRecord
	ProtectionRestore ProtectionPointObservation
	PostcheckPolicy   PostcheckQualification
}

type RollbackPlan struct {
	Eligibility               RollbackEligibility
	Executable                bool
	RollbackMutationQualified bool
	Reasons                   []ReasonCode
	Conflicts                 []Conflict
	RequiredPostchecks        []Postcheck
}

func AssessRollback(input RollbackPlanInput) RollbackPlan {
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

	unknown = assessRollbackGeneration(input.Current, false, reasons) || unknown
	unknown = assessRollbackGeneration(input.Previous, true, reasons) || unknown
	if input.Active.State == ActivePointerValid && input.Current.State == EvidenceVerified && !input.Active.names(input.Current) {
		reasons.add(ReasonActiveGenerationMismatch)
	}
	if input.Current.GenerationID != "" && input.Current.GenerationID == input.Previous.GenerationID {
		reasons.add(ReasonPreviousGenerationSame)
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

	_, compatibilityUnknown := assessCompatibility(
		input.Compatibility,
		input.Previous.Identity,
		input.Current.Identity,
		reasons,
		false,
	)
	unknown = unknown || compatibilityUnknown

	if input.Compatibility.State == EvidenceVerified && input.Compatibility.ProtectionRestoreRequired {
		switch input.ProtectionRestore.State {
		case EvidenceVerified:
			if !closedIDPattern.MatchString(input.ProtectionRestore.ID) {
				reasons.add(ReasonProtectionRestoreIncomplete)
			}
		case EvidenceUnknown:
			reasons.add(ReasonProtectionRestoreUnknown)
			unknown = true
		default:
			reasons.add(ReasonProtectionRestoreIncomplete)
		}
	}

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

	plan := RollbackPlan{
		RollbackMutationQualified: false,
		Reasons:                   reasons.values(),
		Conflicts:                 conflicts(),
		RequiredPostchecks:        required,
	}
	if len(plan.Reasons) == 0 {
		plan.Eligibility = RollbackEligible
		plan.Executable = true
		return plan
	}
	if unknown {
		plan.Eligibility = RollbackUnknown
	} else {
		plan.Eligibility = RollbackIneligible
	}
	return plan
}

func assessRollbackGeneration(generation SealedGenerationObservation, previous bool, reasons *reasonSet) bool {
	switch generation.State {
	case EvidenceVerified:
		if !generation.exact() {
			if previous {
				reasons.add(ReasonPreviousGenerationInvalid)
			} else {
				if generation.Identity.Validate() != nil {
					reasons.add(ReasonCurrentIdentityInvalid)
				}
				reasons.add(ReasonCurrentSealInvalid)
			}
		}
	case EvidenceUnknown:
		if previous {
			reasons.add(ReasonPreviousGenerationUnknown)
		} else {
			reasons.add(ReasonCurrentSealUnknown)
		}
		return true
	case EvidenceMissing:
		if previous {
			reasons.add(ReasonPreviousGenerationMissing)
		} else {
			reasons.add(ReasonCurrentSealInvalid)
		}
	default:
		if previous {
			reasons.add(ReasonPreviousGenerationInvalid)
		} else {
			reasons.add(ReasonCurrentSealInvalid)
		}
	}
	return false
}
