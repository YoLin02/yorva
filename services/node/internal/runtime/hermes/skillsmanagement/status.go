package skillsmanagement

import yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"

type InstalledState = yorvaruntime.SkillInstallationState

const (
	InstalledStateInstalled    = yorvaruntime.SkillInstalled
	InstalledStateNotInstalled = yorvaruntime.SkillNotInstalled
	InstalledStateUnknown      = yorvaruntime.SkillInstallationUnknown
)

type EnabledState = yorvaruntime.SkillEnabledState

const (
	EnabledStateEnabled  = yorvaruntime.SkillEnabled
	EnabledStateDisabled = yorvaruntime.SkillDisabled
	EnabledStateUnknown  = yorvaruntime.SkillEnabledUnknown
)

type ScanState = yorvaruntime.SkillScanState

const (
	ScanStateClean      = yorvaruntime.SkillScanClean
	ScanStateWarning    = yorvaruntime.SkillScanWarning
	ScanStateBlocked    = yorvaruntime.SkillScanBlocked
	ScanStateNotScanned = yorvaruntime.SkillScanNotScanned
	ScanStateUnknown    = yorvaruntime.SkillScanUnknown
)
