package upgrademanagement

import "testing"

func TestAssessRollbackRequiresSealedLineageCompatibilityProtectionAndPostchecks(t *testing.T) {
	input := validRollbackInput()
	plan := AssessRollback(input)
	if plan.Eligibility != RollbackEligible || !plan.PlanEvidenceComplete {
		t.Fatalf("plan = %#v, want ELIGIBLE executable", plan)
	}
	if plan.RollbackMutationQualified || plan.RollbackExecutable() {
		t.Fatal("unapproved rollback mutation became qualified")
	}
}

func TestPreviousGenerationPresenceAloneNeverMakesRollbackEligible(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RollbackPlanInput)
		reason ReasonCode
	}{
		{
			name: "lineage unproven",
			mutate: func(input *RollbackPlanInput) {
				input.Previous.LineageProven = false
			},
			reason: ReasonPreviousGenerationInvalid,
		},
		{
			name: "compatibility missing",
			mutate: func(input *RollbackPlanInput) {
				input.Compatibility = CompatibilityRecord{State: EvidenceMissing}
			},
			reason: ReasonCompatibilityMissing,
		},
		{
			name: "downgrade unsafe",
			mutate: func(input *RollbackPlanInput) {
				input.Compatibility.Rollback = CompatibilityUnsafe
			},
			reason: ReasonRollbackCompatibilityUnsafe,
		},
		{
			name: "protection restore missing",
			mutate: func(input *RollbackPlanInput) {
				input.ProtectionRestore = ProtectionPointObservation{State: EvidenceMissing}
			},
			reason: ReasonProtectionRestoreIncomplete,
		},
		{
			name: "postcheck unqualified",
			mutate: func(input *RollbackPlanInput) {
				input.PostcheckPolicy.Qualified = input.PostcheckPolicy.Qualified[:1]
			},
			reason: ReasonPostchecksUnqualified,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validRollbackInput()
			test.mutate(&input)
			plan := AssessRollback(input)
			if plan.Eligibility != RollbackIneligible || plan.PlanEvidenceComplete || plan.RollbackMutationQualified {
				t.Fatalf("plan = %#v, want INELIGIBLE", plan)
			}
			assertReason(t, plan.Reasons, test.reason)
		})
	}
}

func TestAssessRollbackFailsClosedOnUnknownCompatibility(t *testing.T) {
	input := validRollbackInput()
	input.Compatibility.State = EvidenceUnknown

	plan := AssessRollback(input)
	if plan.Eligibility != RollbackUnknown || plan.PlanEvidenceComplete || plan.RollbackMutationQualified {
		t.Fatalf("plan = %#v, want non-executable UNKNOWN", plan)
	}
	assertReason(t, plan.Reasons, ReasonCompatibilityUnknown)
}

func validRollbackInput() RollbackPlanInput {
	previousIdentity := testIdentity("0.20.2", 'a')
	currentIdentity := testIdentity("0.20.5", 'b')
	previous := testGeneration("gen_aaaaaaaaaaaaaaaaaaaaaa", previousIdentity, 'c', 'd')
	current := testGeneration("gen_bbbbbbbbbbbbbbbbbbbbbb", currentIdentity, 'e', 'f')
	features := FeatureInventory{Models: true, Gateways: true, Channels: true, Skills: true, MCP: true}
	return RollbackPlanInput{
		Managed: ManagedProven,
		Active: ActivePointerObservation{
			State:          ActivePointerValid,
			GenerationID:   current.GenerationID,
			SealSHA256:     current.SealSHA256,
			ManifestSHA256: current.ManifestSHA256,
		},
		Current:   current,
		Previous:  previous,
		Inventory: InventoryObservation{State: EvidenceVerified, Features: features},
		Compatibility: CompatibilityRecord{
			State:                     EvidenceVerified,
			ID:                        "compat_0202_0205",
			WindowsEvidenceID:         "windows_0202_0205",
			From:                      previousIdentity,
			To:                        currentIdentity,
			FromConfigSchema:          38,
			ToConfigSchema:            38,
			UserDataInventoryComplete: true,
			Migration:                 MigrationNone,
			Upgrade:                   CompatibilityProven,
			Rollback:                  CompatibilityProven,
			ProtectionRestoreRequired: true,
		},
		ProtectionRestore: ProtectionPointObservation{State: EvidenceVerified, ID: "restore_verified_1"},
		PostcheckPolicy: PostcheckQualification{
			State:     EvidenceVerified,
			Qualified: requiredPostchecks(features),
		},
	}
}
