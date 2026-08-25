package runtime

import (
	"context"
	"encoding/hex"
)

type SkillOwnership string

const (
	SkillOwnershipYORVAManaged   SkillOwnership = "YORVA_MANAGED"
	SkillOwnershipExternal       SkillOwnership = "EXTERNAL"
	SkillOwnershipRuntimeBundled SkillOwnership = "RUNTIME_BUNDLED"
	SkillOwnershipUnknown        SkillOwnership = "UNKNOWN"
)

func (s SkillOwnership) Valid() bool {
	return s == SkillOwnershipYORVAManaged || s == SkillOwnershipExternal ||
		s == SkillOwnershipRuntimeBundled || s == SkillOwnershipUnknown
}

type SkillProjectionState string

const (
	SkillProjectionProjected     SkillProjectionState = "PROJECTED"
	SkillProjectionNotProjected  SkillProjectionState = "NOT_PROJECTED"
	SkillProjectionDriftMissing  SkillProjectionState = "DRIFT_MISSING"
	SkillProjectionDriftModified SkillProjectionState = "DRIFT_MODIFIED"
	SkillProjectionConflict      SkillProjectionState = "CONFLICT"
	SkillProjectionUnknown       SkillProjectionState = "UNKNOWN"
)

func (s SkillProjectionState) Valid() bool {
	return s == SkillProjectionProjected || s == SkillProjectionNotProjected ||
		s == SkillProjectionDriftMissing || s == SkillProjectionDriftModified ||
		s == SkillProjectionConflict || s == SkillProjectionUnknown
}

type SkillInstallationState string

const (
	SkillInstalled           SkillInstallationState = "INSTALLED"
	SkillNotInstalled        SkillInstallationState = "NOT_INSTALLED"
	SkillInstallationUnknown SkillInstallationState = "UNKNOWN"
)

func (s SkillInstallationState) Valid() bool {
	return s == SkillInstalled || s == SkillNotInstalled || s == SkillInstallationUnknown
}

type SkillEnabledState string

const (
	SkillEnabled        SkillEnabledState = "ENABLED"
	SkillDisabled       SkillEnabledState = "DISABLED"
	SkillEnabledUnknown SkillEnabledState = "UNKNOWN"
)

func (s SkillEnabledState) Valid() bool {
	return s == SkillEnabled || s == SkillDisabled || s == SkillEnabledUnknown
}

type SkillScanState string

const (
	SkillScanClean      SkillScanState = "CLEAN"
	SkillScanWarning    SkillScanState = "WARNING"
	SkillScanBlocked    SkillScanState = "BLOCKED"
	SkillScanNotScanned SkillScanState = "NOT_SCANNED"
	SkillScanUnknown    SkillScanState = "UNKNOWN"
)

func (s SkillScanState) Valid() bool {
	return s == SkillScanClean || s == SkillScanWarning || s == SkillScanBlocked || s == SkillScanNotScanned || s == SkillScanUnknown
}

type Skill struct {
	ID                string
	SourceID          string
	Version           string
	Ownership         SkillOwnership
	ProjectionState   SkillProjectionState
	InstallationState SkillInstallationState
	EnabledState      SkillEnabledState
	ScanState         SkillScanState
	UpdateAvailable   bool
}

func (s Skill) Validate() error {
	if err := validateManagementID("skill id", s.ID); err != nil {
		return err
	}
	if s.SourceID != "" {
		if err := validateManagementID("skill source id", s.SourceID); err != nil {
			return err
		}
	}
	if err := validateBoundedText("skill version", s.Version); err != nil {
		return err
	}
	// Empty ownership/projection values remain accepted for native Runtime
	// inventory implementations that have not classified YORVA ownership. A
	// classified value is always closed to the normalized enums above.
	if (s.Ownership != "" && !s.Ownership.Valid()) ||
		(s.ProjectionState != "" && !s.ProjectionState.Valid()) ||
		!s.InstallationState.Valid() || !s.EnabledState.Valid() || !s.ScanState.Valid() {
		return ErrInvalidManagementContract
	}
	return nil
}

type SkillInstallRequest struct {
	SourceID string
}

func (r SkillInstallRequest) Validate() error {
	return validateManagementID("skill source id", r.SourceID)
}

type SkillConfigureRequest struct {
	SkillID string
	Enabled bool
}

func (r SkillConfigureRequest) Validate() error {
	return validateManagementID("skill id", r.SkillID)
}

type SkillReader interface {
	ListSkills(context.Context, Installation, string) ([]Skill, error)
	InspectSkill(context.Context, Installation, string, string) (Skill, error)
}

type NativeSkillCapability struct {
	Supported bool
	Reason    string
}

func (c NativeSkillCapability) Validate() error {
	return validateBoundedText("native Skill capability reason", c.Reason)
}

// NativeSkillCapabilities reports Runtime-native support independently from
// YORVA's managed projection lifecycle. Unsupported native mutation must not
// disable an available SkillProjector.
type NativeSkillCapabilities struct {
	Inventory            NativeSkillCapability
	NativeInstall        NativeSkillCapability
	NativeUpdate         NativeSkillCapability
	NativeRemove         NativeSkillCapability
	NativeEnableDisable  NativeSkillCapability
	NativeProfileBinding NativeSkillCapability
}

func (c NativeSkillCapabilities) Validate() error {
	values := [...]NativeSkillCapability{
		c.Inventory,
		c.NativeInstall,
		c.NativeUpdate,
		c.NativeRemove,
		c.NativeEnableDisable,
		c.NativeProfileBinding,
	}
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// SkillProjectRequest is constructed only from a YORVA-owned, approved source
// artifact. SourceDir is an internal trusted path and must never be populated
// from a public caller-provided path.
type SkillProjectRequest struct {
	SkillID       string
	SourceDir     string
	SourceID      string
	Version       string
	ContentSHA256 string
	DeploymentID  string
}

func (r SkillProjectRequest) Validate() error {
	if err := validateManagementID("skill id", r.SkillID); err != nil {
		return err
	}
	if err := validateBoundedText("skill source directory", r.SourceDir); err != nil || r.SourceDir == "" {
		return ErrInvalidManagementContract
	}
	if err := validateManagementID("skill source id", r.SourceID); err != nil {
		return err
	}
	if err := validateBoundedText("skill version", r.Version); err != nil || r.Version == "" {
		return ErrInvalidManagementContract
	}
	if err := validateManagementID("skill deployment id", r.DeploymentID); err != nil {
		return err
	}
	if len(r.ContentSHA256) != 64 {
		return ErrInvalidManagementContract
	}
	decoded, err := hex.DecodeString(r.ContentSHA256)
	if err != nil || len(decoded) != 32 {
		return ErrInvalidManagementContract
	}
	for _, value := range r.ContentSHA256 {
		if value >= 'A' && value <= 'F' {
			return ErrInvalidManagementContract
		}
	}
	return nil
}

// SkillProjector owns only YORVA-managed copies and verified projections. It
// must not adopt, overwrite, or remove a Runtime-owned/external Skill.
type SkillProjector interface {
	ListSkillProjections(context.Context, Installation, string) ([]Skill, error)
	InspectSkillProjection(context.Context, Installation, string, string) (Skill, error)
	ProjectSkill(context.Context, Installation, string, SkillProjectRequest, ProgressSink) (Skill, error)
	UnprojectSkill(context.Context, Installation, string, string, string, ProgressSink) (Skill, error)
}

// SkillManager owns only typed, approved-source mutations. Implementations must
// independently enforce source qualification, scan verdicts, and authoritative
// Profile read-back; callers cannot request a force bypass through this contract.
type SkillManager interface {
	InstallSkill(context.Context, Installation, string, SkillInstallRequest, ProgressSink) (Skill, error)
	UpdateSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error)
	RemoveSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error)
	ConfigureSkill(context.Context, Installation, string, SkillConfigureRequest, ProgressSink) (Skill, error)
}
