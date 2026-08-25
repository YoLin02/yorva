package upgrademanagement

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildUpgradePlanAcceptsCompleteExactPlanningEvidence(t *testing.T) {
	input := validUpgradeInput()
	plan := BuildUpgradePlan(input)

	if plan.Availability != UpgradeAvailable || !plan.PlanEvidenceComplete || !plan.RollbackEligible {
		t.Fatalf("plan = %#v, want AVAILABLE executable with rollback eligibility", plan)
	}
	if plan.UpgradeMutationQualified || plan.RollbackMutationQualified || plan.UpgradeExecutable() || plan.RollbackExecutable() {
		t.Fatalf("unapproved mutation became qualified: %#v", plan)
	}
	if len(plan.Reasons) != 0 {
		t.Fatalf("Reasons = %v, want none", plan.Reasons)
	}
	if !slices.Equal(plan.Conflicts, closedConflictSet) {
		t.Fatalf("Conflicts = %v, want closed set %v", plan.Conflicts, closedConflictSet)
	}
	wantPostchecks := requiredPostchecks(input.Inventory.Features)
	if !slices.Equal(plan.RequiredPostchecks, wantPostchecks) {
		t.Fatalf("RequiredPostchecks = %v, want %v", plan.RequiredPostchecks, wantPostchecks)
	}
}

func TestBuildUpgradePlanRejectsInvalidMismatchAndUnmanagedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*UpgradePlanInput)
		reason ReasonCode
	}{
		{
			name: "invalid active pointer",
			mutate: func(input *UpgradePlanInput) {
				input.Active.State = ActivePointerInvalid
			},
			reason: ReasonActivePointerNotValid,
		},
		{
			name: "active seal mismatch",
			mutate: func(input *UpgradePlanInput) {
				input.Current.SealSHA256 = strings.Repeat("0", 64)
			},
			reason: ReasonActiveGenerationMismatch,
		},
		{
			name: "invalid active seal",
			mutate: func(input *UpgradePlanInput) {
				input.Current.State = EvidenceInvalid
			},
			reason: ReasonCurrentSealInvalid,
		},
		{
			name: "invalid current identity",
			mutate: func(input *UpgradePlanInput) {
				input.Current.Identity.Commit = "unknown"
			},
			reason: ReasonCurrentIdentityInvalid,
		},
		{
			name: "target archive mismatch",
			mutate: func(input *UpgradePlanInput) {
				input.Target.Observed.Archive.SHA256 = strings.Repeat("f", 64)
			},
			reason: ReasonTargetIdentityMismatch,
		},
		{
			name: "target source mismatch",
			mutate: func(input *UpgradePlanInput) {
				input.Target.Observed.Source.Repository = "Other/hermes-agent"
			},
			reason: ReasonTargetIdentityMismatch,
		},
		{
			name: "unmanaged",
			mutate: func(input *UpgradePlanInput) {
				input.Managed = ManagedNo
			},
			reason: ReasonUnmanaged,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validUpgradeInput()
			test.mutate(&input)
			plan := BuildUpgradePlan(input)
			if plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified || plan.Availability != UpgradeBlocked {
				t.Fatalf("plan = %#v, want non-executable BLOCKED", plan)
			}
			assertReason(t, plan.Reasons, test.reason)
		})
	}
}

func TestBuildUpgradePlanTreatsExactSameTargetAsUpToDate(t *testing.T) {
	input := validUpgradeInput()
	input.Target.Compiled = input.Current.Identity
	input.Target.Observed = input.Current.Identity
	input.Compatibility.From = input.Current.Identity
	input.Compatibility.To = input.Current.Identity

	plan := BuildUpgradePlan(input)
	if plan.Availability != UpgradeUpToDate || plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified {
		t.Fatalf("plan = %#v, want non-executable UP_TO_DATE", plan)
	}
	assertReason(t, plan.Reasons, ReasonTargetAlreadyActive)
}

func TestSameIdentityDoesNotMaskInvalidActivePointer(t *testing.T) {
	input := validUpgradeInput()
	input.Target.Compiled = input.Current.Identity
	input.Target.Observed = input.Current.Identity
	input.Active.State = ActivePointerInvalid

	plan := BuildUpgradePlan(input)
	if plan.Availability == UpgradeUpToDate || plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified {
		t.Fatalf("plan = %#v, invalid active pointer cannot prove UP_TO_DATE", plan)
	}
	assertReason(t, plan.Reasons, ReasonActivePointerNotValid)
}

func TestBuildUpgradePlanBlocksMissingProtectionCompatibilityAndUnsafeDowngrade(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*UpgradePlanInput)
		reason ReasonCode
	}{
		{
			name: "missing backup",
			mutate: func(input *UpgradePlanInput) {
				input.ProtectionPoint = ProtectionPointObservation{State: EvidenceMissing}
			},
			reason: ReasonProtectionPointUnavailable,
		},
		{
			name: "missing compatibility",
			mutate: func(input *UpgradePlanInput) {
				input.Compatibility = CompatibilityRecord{State: EvidenceMissing}
			},
			reason: ReasonCompatibilityMissing,
		},
		{
			name: "unsafe downgrade",
			mutate: func(input *UpgradePlanInput) {
				input.Compatibility.Rollback = CompatibilityUnsafe
			},
			reason: ReasonRollbackCompatibilityUnsafe,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validUpgradeInput()
			test.mutate(&input)
			plan := BuildUpgradePlan(input)
			if plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified || plan.RollbackEligible || plan.Availability != UpgradeBlocked {
				t.Fatalf("plan = %#v, want blocked with rollback disabled", plan)
			}
			assertReason(t, plan.Reasons, test.reason)
		})
	}
}

func TestBuildUpgradePlanFailsClosedOnUnknownInventoryAndPostcheck(t *testing.T) {
	input := validUpgradeInput()
	input.Inventory.State = EvidenceUnknown
	input.PostcheckPolicy.State = EvidenceUnknown

	plan := BuildUpgradePlan(input)
	if plan.Availability != UpgradeUnknown || plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified {
		t.Fatalf("plan = %#v, want non-executable UNKNOWN", plan)
	}
	assertReason(t, plan.Reasons, ReasonInventoryUnknown)
	assertReason(t, plan.Reasons, ReasonPostchecksUnknown)
}

func TestBuildUpgradePlanNeverQualifiesMissingOrUnknownPlanningEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*UpgradePlanInput)
		reason ReasonCode
	}{
		{
			name: "unknown protection point",
			mutate: func(input *UpgradePlanInput) {
				input.ProtectionPoint = ProtectionPointObservation{State: EvidenceUnknown}
			},
			reason: ReasonProtectionPointUnknown,
		},
		{
			name: "unknown compatibility",
			mutate: func(input *UpgradePlanInput) {
				input.Compatibility = CompatibilityRecord{State: EvidenceUnknown}
			},
			reason: ReasonCompatibilityUnknown,
		},
		{
			name: "missing postcheck qualification",
			mutate: func(input *UpgradePlanInput) {
				input.PostcheckPolicy = PostcheckQualification{State: EvidenceMissing}
			},
			reason: ReasonPostchecksUnqualified,
		},
		{
			name: "unknown postcheck qualification",
			mutate: func(input *UpgradePlanInput) {
				input.PostcheckPolicy = PostcheckQualification{State: EvidenceUnknown}
			},
			reason: ReasonPostchecksUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validUpgradeInput()
			test.mutate(&input)
			plan := BuildUpgradePlan(input)
			if plan.PlanEvidenceComplete || plan.UpgradeMutationQualified || plan.RollbackMutationQualified ||
				plan.UpgradeExecutable() || plan.RollbackExecutable() {
				t.Fatalf("incomplete evidence implied qualification: %#v", plan)
			}
			assertReason(t, plan.Reasons, test.reason)
		})
	}
}

func validUpgradeInput() UpgradePlanInput {
	current := testIdentity("0.20.2", 'a')
	target := testIdentity("0.20.5", 'b')
	features := FeatureInventory{Models: true, Gateways: true, Channels: true, Skills: true, MCP: true}
	generation := testGeneration("gen_aaaaaaaaaaaaaaaaaaaaaa", current, 'c', 'd')
	return UpgradePlanInput{
		Managed: ManagedProven,
		Active: ActivePointerObservation{
			State:          ActivePointerValid,
			GenerationID:   generation.GenerationID,
			SealSHA256:     generation.SealSHA256,
			ManifestSHA256: generation.ManifestSHA256,
		},
		Current: generation,
		Target: TargetObservation{
			State:    EvidenceVerified,
			Compiled: target,
			Observed: target,
		},
		Inventory:       InventoryObservation{State: EvidenceVerified, Features: features},
		ProtectionPoint: ProtectionPointObservation{State: EvidenceVerified, ID: "backup_verified_1"},
		Compatibility: CompatibilityRecord{
			State:                     EvidenceVerified,
			ID:                        "compat_0202_0205",
			WindowsEvidenceID:         "windows_0202_0205",
			From:                      current,
			To:                        target,
			FromConfigSchema:          38,
			ToConfigSchema:            38,
			UserDataInventoryComplete: true,
			Migration:                 MigrationNone,
			Upgrade:                   CompatibilityProven,
			Rollback:                  CompatibilityProven,
		},
		PostcheckPolicy: PostcheckQualification{
			State:     EvidenceVerified,
			Qualified: requiredPostchecks(features),
		},
	}
}

func testIdentity(version string, fill byte) SnapshotIdentity {
	hex := strings.Repeat(string(fill), 64)
	commit := strings.Repeat(string(fill), 40)
	return SnapshotIdentity{
		Version:     version,
		Commit:      commit,
		Archive:     ArtifactIdentity{SizeBytes: 73_798_347, SHA256: hex},
		LicensePath: "LICENSE",
		License:     ArtifactIdentity{SizeBytes: 1_070, SHA256: hex},
		Source: SourceIdentity{
			Repository:    "NousResearch/hermes-agent",
			ArchiveRoot:   "hermes-agent-" + commit,
			InstallerPath: "scripts/install.ps1",
			Installer:     ArtifactIdentity{SizeBytes: 243_941, SHA256: hex},
		},
	}
}

func testGeneration(id string, identity SnapshotIdentity, sealFill, manifestFill byte) SealedGenerationObservation {
	return SealedGenerationObservation{
		State:          EvidenceVerified,
		GenerationID:   id,
		SealSHA256:     strings.Repeat(string(sealFill), 64),
		ManifestSHA256: strings.Repeat(string(manifestFill), 64),
		LineageProven:  true,
		Identity:       identity,
	}
}

func assertReason(t *testing.T, reasons []ReasonCode, want ReasonCode) {
	t.Helper()
	if !slices.Contains(reasons, want) {
		t.Fatalf("Reasons = %v, want %s", reasons, want)
	}
}
