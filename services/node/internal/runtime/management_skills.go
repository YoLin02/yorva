package runtime

import "context"

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
	if !s.InstallationState.Valid() || !s.EnabledState.Valid() || !s.ScanState.Valid() {
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

// SkillManager owns only typed, approved-source mutations. Implementations must
// independently enforce source qualification, scan verdicts, and authoritative
// Profile read-back; callers cannot request a force bypass through this contract.
type SkillManager interface {
	InstallSkill(context.Context, Installation, string, SkillInstallRequest, ProgressSink) (Skill, error)
	UpdateSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error)
	RemoveSkill(context.Context, Installation, string, string, ProgressSink) (Skill, error)
	ConfigureSkill(context.Context, Installation, string, SkillConfigureRequest, ProgressSink) (Skill, error)
}
