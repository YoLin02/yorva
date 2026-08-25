package upgrademanagement

import "regexp"

type EvidenceState string

const (
	EvidenceVerified EvidenceState = "VERIFIED"
	EvidenceMissing  EvidenceState = "MISSING"
	EvidenceInvalid  EvidenceState = "INVALID"
	EvidenceUnknown  EvidenceState = "UNKNOWN"
)

func (s EvidenceState) valid() bool {
	return s == EvidenceVerified || s == EvidenceMissing || s == EvidenceInvalid || s == EvidenceUnknown
}

type ManagedState string

const (
	ManagedProven  ManagedState = "MANAGED"
	ManagedNo      ManagedState = "UNMANAGED"
	ManagedUnknown ManagedState = "UNKNOWN"
)

type ActivePointerState string

const (
	ActivePointerValid   ActivePointerState = "VALID"
	ActivePointerMissing ActivePointerState = "MISSING"
	ActivePointerInvalid ActivePointerState = "INVALID"
	ActivePointerUnknown ActivePointerState = "UNKNOWN"
)

type ActivePointerObservation struct {
	State          ActivePointerState
	GenerationID   string
	SealSHA256     string
	ManifestSHA256 string
}

type SealedGenerationObservation struct {
	State          EvidenceState
	GenerationID   string
	SealSHA256     string
	ManifestSHA256 string
	LineageProven  bool
	Identity       SnapshotIdentity
}

func (g SealedGenerationObservation) exact() bool {
	return g.State == EvidenceVerified &&
		generationIDPattern.MatchString(g.GenerationID) &&
		sha256Pattern.MatchString(g.SealSHA256) &&
		sha256Pattern.MatchString(g.ManifestSHA256) &&
		g.LineageProven &&
		g.Identity.Validate() == nil
}

func (a ActivePointerObservation) names(g SealedGenerationObservation) bool {
	return a.State == ActivePointerValid &&
		a.GenerationID == g.GenerationID &&
		a.SealSHA256 == g.SealSHA256 &&
		a.ManifestSHA256 == g.ManifestSHA256
}

type TargetObservation struct {
	State    EvidenceState
	Compiled SnapshotIdentity
	Observed SnapshotIdentity
}

type FeatureInventory struct {
	Models   bool
	Gateways bool
	Channels bool
	Skills   bool
	MCP      bool
}

type InventoryObservation struct {
	State    EvidenceState
	Features FeatureInventory
}

type ProtectionPointObservation struct {
	State EvidenceState
	ID    string
}

type CompatibilitySafety string

const (
	CompatibilityProven  CompatibilitySafety = "PROVEN"
	CompatibilityUnsafe  CompatibilitySafety = "UNSAFE"
	CompatibilityUnknown CompatibilitySafety = "UNKNOWN"
)

func (s CompatibilitySafety) valid() bool {
	return s == CompatibilityProven || s == CompatibilityUnsafe || s == CompatibilityUnknown
}

type MigrationBehavior string

const (
	MigrationNone      MigrationBehavior = "NONE"
	MigrationAutomatic MigrationBehavior = "AUTOMATIC"
	MigrationDeferred  MigrationBehavior = "DEFERRED"
	MigrationLazy      MigrationBehavior = "LAZY"
)

func (m MigrationBehavior) valid() bool {
	return m == MigrationNone || m == MigrationAutomatic || m == MigrationDeferred || m == MigrationLazy
}

// CompatibilityRecord binds one reviewed exact source/target pair. The
// rollback verdict is also an upgrade gate for Phase 7: an unsafe downgrade
// cannot be hidden by merely retaining the previous generation bytes.
type CompatibilityRecord struct {
	State                     EvidenceState
	ID                        string
	WindowsEvidenceID         string
	From                      SnapshotIdentity
	To                        SnapshotIdentity
	FromConfigSchema          int
	ToConfigSchema            int
	UserDataInventoryComplete bool
	Migration                 MigrationBehavior
	Upgrade                   CompatibilitySafety
	Rollback                  CompatibilitySafety
	ProtectionRestoreRequired bool
}

type Postcheck string

const (
	PostcheckActiveSeal     Postcheck = "ACTIVE_POINTER_AND_SEAL"
	PostcheckDualLaunchers  Postcheck = "DUAL_LAUNCHER_IDENTITY"
	PostcheckDiscovery      Postcheck = "ACTIVE_GENERATION_DISCOVERY"
	PostcheckProfiles       Postcheck = "PROFILE_INVENTORY"
	PostcheckModels         Postcheck = "MODEL_READBACK"
	PostcheckLifecycle      Postcheck = "GATEWAY_LIFECYCLE"
	PostcheckChannels       Postcheck = "CHANNEL_RECONCILIATION"
	PostcheckSkills         Postcheck = "SKILL_RECONCILIATION"
	PostcheckMCP            Postcheck = "MCP_RECONCILIATION"
	PostcheckHealthSecurity Postcheck = "HEALTH_AND_SECURITY"
)

func (p Postcheck) valid() bool {
	switch p {
	case PostcheckActiveSeal, PostcheckDualLaunchers, PostcheckDiscovery, PostcheckProfiles,
		PostcheckModels, PostcheckLifecycle, PostcheckChannels, PostcheckSkills,
		PostcheckMCP, PostcheckHealthSecurity:
		return true
	default:
		return false
	}
}

type PostcheckQualification struct {
	State     EvidenceState
	Qualified []Postcheck
}

type Conflict string

const (
	ConflictInstall           Conflict = "RUNTIME_INSTALL"
	ConflictPrerequisites     Conflict = "RUNTIME_PREREQUISITES"
	ConflictRestore           Conflict = "RESTORE"
	ConflictBackupMutation    Conflict = "BACKUP_MUTATION"
	ConflictInstanceMutation  Conflict = "AFFECTED_INSTANCE_MUTATION"
	ConflictLifecycleMutation Conflict = "AFFECTED_LIFECYCLE_MUTATION"
	ConflictChannelMutation   Conflict = "AFFECTED_CHANNEL_MUTATION"
	ConflictSkillMutation     Conflict = "AFFECTED_SKILL_MUTATION"
	ConflictMCPMutation       Conflict = "AFFECTED_MCP_MUTATION"
)

var closedConflictSet = []Conflict{
	ConflictInstall,
	ConflictPrerequisites,
	ConflictRestore,
	ConflictBackupMutation,
	ConflictInstanceMutation,
	ConflictLifecycleMutation,
	ConflictChannelMutation,
	ConflictSkillMutation,
	ConflictMCPMutation,
}

var (
	generationIDPattern = regexp.MustCompile(`^gen_[a-z0-9]{22}$`)
	closedIDPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,127}$`)
)

func conflicts() []Conflict {
	return append([]Conflict(nil), closedConflictSet...)
}

func requiredPostchecks(features FeatureInventory) []Postcheck {
	checks := []Postcheck{
		PostcheckActiveSeal,
		PostcheckDualLaunchers,
		PostcheckDiscovery,
		PostcheckProfiles,
	}
	if features.Models {
		checks = append(checks, PostcheckModels)
	}
	if features.Gateways {
		checks = append(checks, PostcheckLifecycle)
	}
	if features.Channels {
		checks = append(checks, PostcheckChannels)
	}
	if features.Skills {
		checks = append(checks, PostcheckSkills)
	}
	if features.MCP {
		checks = append(checks, PostcheckMCP)
	}
	return append(checks, PostcheckHealthSecurity)
}

func postchecksQualified(policy PostcheckQualification, required []Postcheck) bool {
	if policy.State != EvidenceVerified {
		return false
	}
	seen := make(map[Postcheck]struct{}, len(policy.Qualified))
	for _, check := range policy.Qualified {
		if !check.valid() {
			return false
		}
		if _, duplicate := seen[check]; duplicate {
			return false
		}
		seen[check] = struct{}{}
	}
	for _, check := range required {
		if _, ok := seen[check]; !ok {
			return false
		}
	}
	return true
}
