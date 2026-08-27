package runtime

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeHealthInspector struct{}

type fakeSkillProjector struct{}

type fakeNativeSkillManager struct{}

func (fakeSkillProjector) ListSkillProjections(context.Context, Installation, string) ([]Skill, error) {
	return nil, nil
}

func (fakeSkillProjector) InspectSkillProjection(context.Context, Installation, string, string) (Skill, error) {
	return Skill{}, nil
}

func (fakeSkillProjector) ProjectSkill(context.Context, Installation, string, SkillProjectRequest, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeSkillProjector) UnprojectSkill(context.Context, Installation, string, string, string, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeNativeSkillManager) InstallSkill(context.Context, Installation, string, SkillInstallRequest, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeNativeSkillManager) UpdateSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeNativeSkillManager) RemoveSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeNativeSkillManager) ConfigureSkill(context.Context, Installation, string, SkillConfigureRequest, ProgressSink) (Skill, error) {
	return Skill{}, nil
}

func (fakeHealthInspector) InspectRuntimeHealth(context.Context, Installation) (HealthObservation, error) {
	return HealthObservation{}, nil
}

func (fakeHealthInspector) InspectInstanceHealth(context.Context, Installation, string) (HealthObservation, error) {
	return HealthObservation{}, nil
}

func TestManagementCapabilitiesDefaultClosedAndDeriveWiring(t *testing.T) {
	closed := (Bundle{}).ManagementCapabilities()
	if closed != (ManagementCapabilities{}) {
		t.Fatalf("empty Bundle capabilities = %#v, want all false", closed)
	}

	wired := (Bundle{Health: fakeHealthInspector{}}).ManagementCapabilities()
	if !wired.HealthRead {
		t.Fatal("wired health contract did not enable health capability")
	}
	wired.HealthRead = false
	if wired != (ManagementCapabilities{}) {
		t.Fatalf("unwired management capabilities became true: %#v", wired)
	}
	managed := (Bundle{SkillProjection: fakeSkillProjector{}}).ManagementCapabilities()
	if !managed.SkillMutate {
		t.Fatal("wired YORVA Skill projection did not enable Skill mutation")
	}
	nativeOnly := (Bundle{SkillMutate: fakeNativeSkillManager{}}).ManagementCapabilities()
	if nativeOnly.SkillMutate {
		t.Fatal("absent YORVA Skill projection advertised managed mutation")
	}
}

func TestNormalizedManagementStatesFailClosed(t *testing.T) {
	validHealth := []HealthState{HealthHealthy, HealthDegraded, HealthUnhealthy, HealthUnknown}
	for _, state := range validHealth {
		if !state.Valid() {
			t.Fatalf("health state %q is invalid", state)
		}
	}
	validSecurityAudit := []SecurityAuditState{SecurityAuditClean, SecurityAuditWarning, SecurityAuditBlocked, SecurityAuditUnknown}
	for _, state := range validSecurityAudit {
		if !state.Valid() {
			t.Fatalf("security audit state %q is invalid", state)
		}
	}
	validSecuritySeverity := []SecuritySeverity{SecuritySeverityLow, SecuritySeverityMedium, SecuritySeverityHigh, SecuritySeverityCritical, SecuritySeverityUnknown}
	for _, severity := range validSecuritySeverity {
		if !severity.Valid() {
			t.Fatalf("security severity %q is invalid", severity)
		}
	}
	validSkillInstall := []SkillInstallationState{SkillInstalled, SkillNotInstalled, SkillInstallationUnknown}
	for _, state := range validSkillInstall {
		if !state.Valid() {
			t.Fatalf("skill installation state %q is invalid", state)
		}
	}
	validSkillEnabled := []SkillEnabledState{SkillEnabled, SkillDisabled, SkillEnabledUnknown}
	for _, state := range validSkillEnabled {
		if !state.Valid() {
			t.Fatalf("skill enabled state %q is invalid", state)
		}
	}
	validSkillScan := []SkillScanState{SkillScanClean, SkillScanWarning, SkillScanBlocked, SkillScanNotScanned, SkillScanUnknown}
	for _, state := range validSkillScan {
		if !state.Valid() {
			t.Fatalf("skill scan state %q is invalid", state)
		}
	}
	validOwnership := []SkillOwnership{SkillOwnershipYORVAManaged, SkillOwnershipExternal, SkillOwnershipRuntimeBundled, SkillOwnershipUnknown}
	for _, ownership := range validOwnership {
		if !ownership.Valid() {
			t.Fatalf("Skill ownership %q is invalid", ownership)
		}
	}
	validProjection := []SkillProjectionState{
		SkillProjectionProjected, SkillProjectionNotProjected, SkillProjectionDriftMissing,
		SkillProjectionDriftModified, SkillProjectionConflict, SkillProjectionUnknown,
	}
	for _, state := range validProjection {
		if !state.Valid() {
			t.Fatalf("Skill projection state %q is invalid", state)
		}
	}
	validMCP := []MCPState{MCPNotConfigured, MCPConfigured, MCPAuthRequired, MCPReady, MCPFailed, MCPUnknown}
	for _, state := range validMCP {
		if !state.Valid() {
			t.Fatalf("MCP state %q is invalid", state)
		}
	}
	validBackup := []BackupState{BackupCreating, BackupAvailable, BackupRestoring, BackupFailed, BackupDeleting, BackupMissing, BackupChanged, BackupUndecryptable, BackupMalformed, BackupUnknown}
	for _, state := range validBackup {
		if !state.Valid() {
			t.Fatalf("backup state %q is invalid", state)
		}
	}
	validRestore := []RestoreOutcomeState{RestoreSucceeded, RestoreRolledBack, RestoreRecoveryRequired, RestoreUnknown}
	for _, state := range validRestore {
		if !state.Valid() {
			t.Fatalf("Restore outcome %q is invalid", state)
		}
	}
	validUpgrade := []UpgradeAvailabilityState{UpgradeUpToDate, UpgradeAvailable, UpgradeBlocked, UpgradeUnknown}
	for _, state := range validUpgrade {
		if !state.Valid() {
			t.Fatalf("upgrade state %q is invalid", state)
		}
	}
	validRollback := []RollbackEligibilityState{RollbackEligible, RollbackIneligible, RollbackUnknown}
	for _, state := range validRollback {
		if !state.Valid() {
			t.Fatalf("rollback eligibility %q is invalid", state)
		}
	}
	validUpgradeOutcome := []UpgradeOutcomeState{UpgradeSucceeded, UpgradeRolledBack, UpgradeRecoveryRequired, UpgradeOutcomeUnknown}
	for _, state := range validUpgradeOutcome {
		if !state.Valid() {
			t.Fatalf("upgrade outcome %q is invalid", state)
		}
	}
	if HealthState("healthy").Valid() || SecuritySeverity("SEVERE").Valid() || SkillScanState("PASSED").Valid() ||
		SkillOwnership("MANAGED").Valid() || SkillProjectionState("MISSING").Valid() || MCPState("RUNNING").Valid() ||
		BackupState("UNRECOGNIZED").Valid() || UpgradeAvailabilityState("READY").Valid() {
		t.Fatal("unknown normalized state was accepted")
	}
}

func TestSkillProjectRequestIsClosedAndRejectsInvalidDigest(t *testing.T) {
	typeOfRequest := reflect.TypeOf(SkillProjectRequest{})
	wantFields := []string{"SkillID", "SourceDir", "SourceID", "Version", "ContentSHA256", "DeploymentID"}
	if typeOfRequest.NumField() != len(wantFields) {
		t.Fatalf("SkillProjectRequest has %d fields, want %d", typeOfRequest.NumField(), len(wantFields))
	}
	for index, want := range wantFields {
		if got := typeOfRequest.Field(index).Name; got != want {
			t.Fatalf("SkillProjectRequest field %d = %q, want %q", index, got, want)
		}
	}

	valid := SkillProjectRequest{
		SkillID:       "writer",
		SourceDir:     `C:\trusted\yorva\skills\writer`,
		SourceID:      "reviewed-writer",
		Version:       "1.0.0",
		ContentSHA256: strings.Repeat("a", 64),
		DeploymentID:  "deployment-writer",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Skill projection request = %v", err)
	}
	valid.ContentSHA256 = strings.Repeat("A", 64)
	if err := valid.Validate(); err == nil {
		t.Fatal("uppercase content SHA-256 was accepted")
	}
	valid.ContentSHA256 = "not-a-digest"
	if err := valid.Validate(); err == nil {
		t.Fatal("invalid content SHA-256 was accepted")
	}
}

func TestNativeSkillCapabilitiesRemainIndependentFromManagedProjection(t *testing.T) {
	reason := strings.Repeat("x", managementTextMaxBytes+1)
	capabilities := NativeSkillCapabilities{
		Inventory:           NativeSkillCapability{Supported: true},
		NativeInstall:       NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		NativeUpdate:        NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		NativeRemove:        NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		NativeEnableDisable: NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		NativeProfileBinding: NativeSkillCapability{
			Supported: false,
			Reason:    "deferred_upstream",
		},
	}
	if err := capabilities.Validate(); err != nil {
		t.Fatalf("valid native Skill capabilities = %v", err)
	}
	capabilities.NativeInstall.Reason = reason
	if err := capabilities.Validate(); err == nil {
		t.Fatal("oversized native Skill capability reason was accepted")
	}
}

func TestClosedManagementRequestsRejectUnsafeIdentifiers(t *testing.T) {
	invalid := []string{
		"",
		" https://example.invalid",
		"https://example.invalid",
		"../../escape",
		"server/name",
		"TOKEN=value",
		strings.Repeat("a", managementIDMaxBytes+1),
	}
	for _, value := range invalid {
		if err := (SkillInstallRequest{SourceID: value}).Validate(); err == nil {
			t.Fatalf("unsafe Skill source id %q was accepted", value)
		}
		if err := (MCPInstallRequest{PresetID: value}).Validate(); err == nil {
			t.Fatalf("unsafe MCP preset id %q was accepted", value)
		}
		if err := (BackupCreateRequest{DestinationRef: value}).Validate(); err == nil {
			t.Fatalf("unsafe backup destination reference %q was accepted", value)
		}
	}

	if err := (SkillInstallRequest{SourceID: "approved-skill"}).Validate(); err != nil {
		t.Fatalf("valid Skill source id rejected: %v", err)
	}
	if err := (MCPInstallRequest{PresetID: "approved-mcp"}).Validate(); err != nil {
		t.Fatalf("valid MCP preset id rejected: %v", err)
	}
	if err := (BackupCreateRequest{DestinationRef: strings.Repeat("a", 43), OperationID: "op_backup", RuntimeInstallationID: "rtinst_1"}).Validate(); err != nil {
		t.Fatalf("valid backup destination reference rejected: %v", err)
	}
	if err := (BackupCreateRequest{OperationID: "op_system_backup", RuntimeInstallationID: "rtinst_1"}).Validate(); err != nil {
		t.Fatalf("system backup destination rejected: %v", err)
	}
}

func TestMCPConfiguredAndReadyRemainDistinct(t *testing.T) {
	now := time.Now().UTC()
	configured := MCPServer{ID: "server-1", PresetID: "preset-1", State: MCPConfigured, ObservedAt: now}
	if err := configured.Validate(); err != nil {
		t.Fatalf("configured MCP server rejected: %v", err)
	}
	invalidReady := configured
	invalidReady.State = MCPReady
	if err := invalidReady.Validate(); err == nil {
		t.Fatal("READY without a timestamp was accepted")
	}
	invalidConfigured := configured
	invalidConfigured.ReadyAt = &now
	if err := invalidConfigured.Validate(); err == nil {
		t.Fatal("CONFIGURED retained READY evidence")
	}
	ready := invalidReady
	ready.ReadyAt = &now
	if err := ready.Validate(); err != nil {
		t.Fatalf("fresh READY observation rejected: %v", err)
	}
}

func TestMCPAuthenticationAndToolSelectionAreBounded(t *testing.T) {
	if err := (MCPAuthenticateRequest{ServerID: "server-1", Credential: []byte("secret")}).Validate(); err != nil {
		t.Fatalf("bounded credential rejected: %v", err)
	}
	if err := (MCPAuthenticateRequest{ServerID: "server-1"}).Validate(); err == nil {
		t.Fatal("empty MCP credential accepted")
	}
	if err := (MCPAuthenticateRequest{ServerID: "server-1", Credential: make([]byte, managementSecretMaxBytes+1)}).Validate(); err == nil {
		t.Fatal("oversized MCP credential accepted")
	}
	if err := (MCPConfigureRequest{ServerID: "server-1", EnabledToolIDs: []string{"tool-1", "tool-1"}}).Validate(); err == nil {
		t.Fatal("duplicate MCP tool selection accepted")
	}
}

func TestAvailableBackupRequiresVerifiedMetadata(t *testing.T) {
	now := time.Now().UTC()
	backup := Backup{
		ID:             "backup-1",
		State:          BackupAvailable,
		FormatVersion:  "1",
		RuntimeVersion: "0.20.5",
		SizeBytes:      1024,
		ChecksumSHA256: strings.Repeat("a", 64),
		CreatedAt:      now,
		VerifiedAt:     now,
		KeyMode:        BackupKeyDevice,
	}
	if err := backup.Validate(); err != nil {
		t.Fatalf("verified available backup rejected: %v", err)
	}
	backup.ChecksumSHA256 = "not-a-checksum"
	if err := backup.Validate(); err == nil {
		t.Fatal("AVAILABLE backup without a SHA-256 was accepted")
	}
}

func TestUpgradePlanRequiresManagedCompleteAvailability(t *testing.T) {
	now := time.Now().UTC()
	plan := UpgradePlan{
		State:                UpgradeAvailable,
		Rollback:             RollbackEligible,
		CurrentVersion:       "0.20.2",
		TargetVersion:        "0.20.5",
		CandidateLabel:       "Hermes 0.20.5 packaged snapshot",
		Compatibility:        UpgradeCompatibilityProven,
		Managed:              true,
		InventoryComplete:    true,
		PlanEvidenceComplete: true,
		ObservedAt:           now,
	}
	if err := plan.Validate(); err != nil || !plan.PlanEvidenceComplete || plan.UpgradeExecutable() {
		t.Fatalf("planning truth = (%v, %#v), want complete but mutation-unqualified", err, plan)
	}
	plan.UpgradeMutationQualified = true
	if !plan.UpgradeExecutable() {
		t.Fatal("qualified complete upgrade plan was not executable")
	}
	plan.InventoryComplete = false
	if plan.UpgradeExecutable() {
		t.Fatal("partial inventory produced an executable upgrade plan")
	}
	plan.State = UpgradeBlocked
	if plan.UpgradeExecutable() {
		t.Fatal("blocked upgrade plan was executable")
	}
	plan.State = UpgradeAvailable
	plan.InventoryComplete = true
	plan.Rollback = RollbackUnknown
	if plan.UpgradeExecutable() {
		t.Fatal("unknown rollback compatibility produced an executable upgrade plan")
	}
	plan.Rollback = RollbackEligible
	plan.ProtectionPointRequired = true
	if plan.UpgradeExecutable() {
		t.Fatal("missing required protection point produced an executable upgrade plan")
	}
	plan.ProtectionPointReady = true
	if !plan.UpgradeExecutable() {
		t.Fatal("verified protection point did not restore upgrade eligibility")
	}
}

func TestHealthAndLogsRequireBoundedObservedResults(t *testing.T) {
	now := time.Now().UTC()
	health := HealthObservation{State: HealthUnknown, ObservedAt: now}
	if err := health.Validate(); err != nil {
		t.Fatalf("UNKNOWN health observation rejected: %v", err)
	}
	health.State = "BROKEN"
	if err := health.Validate(); err == nil {
		t.Fatal("unknown health value accepted")
	}

	snapshot := LogSnapshot{
		Category:   LogCategoryErrors,
		ObservedAt: now,
		Entries:    []LogEntry{{Timestamp: now, Message: "redacted"}},
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("bounded log snapshot rejected: %v", err)
	}
	snapshot.Entries[0].Message = strings.Repeat("x", managementTextMaxBytes+1)
	if err := snapshot.Validate(); err == nil {
		t.Fatal("oversized log entry accepted")
	}
}
