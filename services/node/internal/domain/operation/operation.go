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

type TargetType string

const TargetRuntimeKind TargetType = "runtime-kind"

const TargetRuntimeInstallation TargetType = "runtime-installation"

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
