package operation

import (
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type Type string

const TypeRuntimeInstall Type = "runtime.install"

const TypeHermesPrerequisites Type = "hermes.prerequisites"

const TypeInstanceCreate Type = "instance.create"

const TypeInstanceDelete Type = "instance.delete"

const TypeModelValidate Type = "model.validate"

const TypeModelProfileApply Type = "model.profile.apply"

const TypeInstanceStart Type = "instance.start"

const TypeInstanceStop Type = "instance.stop"

const TypeInstanceRestart Type = "instance.restart"

const TypeChannelConnect Type = "channel.connect"

const TypeChannelDisconnect Type = "channel.disconnect"

const TypeSkillInstall Type = "skill.install"

const TypeSkillUpdate Type = "skill.update"

const TypeSkillEnable Type = "skill.enable"

const TypeSkillDisable Type = "skill.disable"

const TypeSkillRemove Type = "skill.remove"

const TypeMCPInstall Type = "mcp.install"

const TypeMCPAuthenticate Type = "mcp.authenticate"

const TypeMCPTest Type = "mcp.test"

const TypeMCPConfigure Type = "mcp.configure"

const TypeMCPRemove Type = "mcp.remove"

const TypeBackupCreate Type = "backup.create"

const TypeBackupDelete Type = "backup.delete"

const TypeBackupRestore Type = "backup.restore"

const TypeRuntimeUpgrade Type = "runtime.upgrade"

const TypeRuntimeRollback Type = "runtime.rollback"

type TargetType string

const TargetRuntimeKind TargetType = "runtime-kind"

const TargetRuntimeInstallation TargetType = "runtime-installation"

const TargetInstance TargetType = "instance"

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

type Stage string

const (
	StagePreflight              Stage = "preflight"
	StageSourceDownload         Stage = "source.download"
	StageSourceVerify           Stage = "source.verify"
	StageProtocolVerify         Stage = "protocol.verify"
	StageInstallUV              Stage = "install.uv"
	StageInstallPython          Stage = "install.python"
	StageInstallGit             Stage = "install.git"
	StageInstallNode            Stage = "install.node"
	StageInstallSystemPackages  Stage = "install.system-packages"
	StageInstallRepository      Stage = "install.repository"
	StageInstallVenv            Stage = "install.venv"
	StageInstallDependencies    Stage = "install.dependencies"
	StageInstallNodeDeps        Stage = "install.node-deps"
	StageInstallNPM             Stage = "install.npm"
	StageInstallPath            Stage = "install.path"
	StageInstallConfigTemplates Stage = "install.config-templates"
	StageInstallBootstrapMarker Stage = "install.bootstrap-marker"
	StagePostcheckDiscovery     Stage = "postcheck.discovery"
	StageCleanup                Stage = "cleanup"
	StageInstanceCreate         Stage = "instance.create"
	StageInstanceDelete         Stage = "instance.delete"
	StageInstanceReconcile      Stage = "instance.reconcile"
	StageModelValidate          Stage = "model.validate"
	StageModelProfileApply      Stage = "model.profile.apply"
	StageInstanceStart          Stage = "instance.start"
	StageInstanceStop           Stage = "instance.stop"
	StageInstanceRestart        Stage = "instance.restart"
	StageLifecycleReconcile     Stage = "lifecycle.reconcile"
	StageChannelPreparing       Stage = "channel.preparing"
	StageChannelQRReady         Stage = "channel.qr-ready"
	StageChannelWaiting         Stage = "channel.waiting"
	StageChannelVerifying       Stage = "channel.verifying"
	StageChannelCommitting      Stage = "channel.committing"
	StageChannelDisconnect      Stage = "channel.disconnect"
	StageSkillPreflight         Stage = "skill.preflight"
	StageSkillProject           Stage = "skill.project"
	StageSkillUnproject         Stage = "skill.unproject"
	StageSkillReconcile         Stage = "skill.reconcile"
	StageMCPPreflight           Stage = "mcp.preflight"
	StageMCPConfigure           Stage = "mcp.configure"
	StageMCPTest                Stage = "mcp.test"
	StageMCPReconcile           Stage = "mcp.reconcile"
	StageBackupPreflight        Stage = "backup.preflight"
	StageBackupSnapshot         Stage = "backup.snapshot"
	StageBackupEncrypt          Stage = "backup.encrypt"
	StageBackupPublish          Stage = "backup.publish"
	StageBackupDelete           Stage = "backup.delete"
	StageBackupReconcile        Stage = "backup.reconcile"
	StageRestorePreflight       Stage = "restore.preflight"
	StageRestoreProtection      Stage = "restore.protection"
	StageRestoreApply           Stage = "restore.apply"
	StageRestoreReconcile       Stage = "restore.reconcile"
	StageUpgradePreflight       Stage = "upgrade.preflight"
	StageUpgradeBuild           Stage = "upgrade.build"
	StageUpgradeActivate        Stage = "upgrade.activate"
	StageUpgradeReconcile       Stage = "upgrade.reconcile"
)

type Operation struct {
	ID             string
	Type           Type
	TargetType     TargetType
	TargetID       string
	Status         Status
	Stage          Stage
	Progress       *int
	Message        string
	ErrorCode      yorvaruntime.ErrorCode
	ErrorMessage   string
	Retryable      bool
	IdempotencyKey string
	CorrelationID  string
	SourcePin      string
	OwnershipNonce string
	TransactionID  string
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	UpdatedAt      time.Time
}

func IsTerminal(status Status) bool {
	return status == StatusSucceeded || status == StatusFailed || status == StatusCancelled
}

func ValidTransition(from, to Status) bool {
	if from == to {
		return false
	}
	switch from {
	case StatusPending:
		return to == StatusRunning || to == StatusFailed || to == StatusCancelled
	case StatusRunning:
		return to == StatusSucceeded || to == StatusFailed || to == StatusCancelled
	default:
		return false
	}
}

// ValidProjectionRepair allows InstallTransaction / active.json to correct a
// wrongly terminal Operation. Forward-only worker transitions stay in ValidTransition.
func ValidProjectionRepair(from, to Status) bool {
	if from == to || !IsTerminal(from) {
		return false
	}
	return to == StatusSucceeded || to == StatusRunning
}

func ValidStatusChange(from, to Status) bool {
	return from == to || ValidTransition(from, to) || ValidProjectionRepair(from, to)
}
