package runtime

import (
	"context"
	"time"
)

type DiscoveryState string

const (
	DiscoveryNotInstalled     DiscoveryState = "NOT_INSTALLED"
	DiscoverySupported        DiscoveryState = "SUPPORTED"
	DiscoveryUnsupported      DiscoveryState = "UNSUPPORTED"
	DiscoveryBrokenExecutable DiscoveryState = "BROKEN_EXECUTABLE"
	DiscoveryMalformedVersion DiscoveryState = "MALFORMED_VERSION"
	DiscoveryTimedOut         DiscoveryState = "TIMED_OUT"
	DiscoveryAmbiguous        DiscoveryState = "AMBIGUOUS"
)

type ErrorCode string

const (
	ErrorRuntimeNotInstalled       ErrorCode = "RUNTIME_NOT_INSTALLED"
	ErrorRuntimeUnsupported        ErrorCode = "RUNTIME_UNSUPPORTED"
	ErrorRuntimeExecutableBroken   ErrorCode = "RUNTIME_EXECUTABLE_BROKEN"
	ErrorRuntimeVersionMalformed   ErrorCode = "RUNTIME_VERSION_MALFORMED"
	ErrorRuntimeDiscoveryTimeout   ErrorCode = "RUNTIME_DISCOVERY_TIMEOUT"
	ErrorRuntimeDiscoveryCancelled ErrorCode = "RUNTIME_DISCOVERY_CANCELLED"
	ErrorRuntimeCommandOutputLimit ErrorCode = "RUNTIME_COMMAND_OUTPUT_LIMIT"
	ErrorRuntimeDiscoveryAmbiguous ErrorCode = "RUNTIME_DISCOVERY_AMBIGUOUS"

	ErrorRuntimeInstallPlatformUnsupported ErrorCode = "RUNTIME_INSTALL_PLATFORM_UNSUPPORTED"
	ErrorRuntimeInstallAlreadyPresent      ErrorCode = "RUNTIME_INSTALL_ALREADY_PRESENT"
	ErrorRuntimeInstallStateConflict       ErrorCode = "RUNTIME_INSTALL_STATE_CONFLICT"
	ErrorRuntimeInstallInProgress          ErrorCode = "RUNTIME_INSTALL_IN_PROGRESS"
	ErrorRuntimeInstallTargetOccupied      ErrorCode = "RUNTIME_INSTALL_TARGET_OCCUPIED"
	ErrorRuntimeInstallSourceUnavailable   ErrorCode = "RUNTIME_INSTALL_SOURCE_UNAVAILABLE"
	ErrorRuntimeInstallInsufficientDisk    ErrorCode = "RUNTIME_INSTALL_INSUFFICIENT_DISK"
	ErrorRuntimeInstallIntegrityFailed     ErrorCode = "RUNTIME_INSTALL_INTEGRITY_FAILED"
	ErrorRuntimeInstallProtocolUnsupported ErrorCode = "RUNTIME_INSTALL_PROTOCOL_UNSUPPORTED"
	ErrorRuntimeInstallManifestMismatch    ErrorCode = "RUNTIME_INSTALL_MANIFEST_MISMATCH"
	ErrorRuntimeInstallStageFailed         ErrorCode = "RUNTIME_INSTALL_STAGE_FAILED"
	ErrorRuntimeInstallPrivilegeRequired   ErrorCode = "RUNTIME_INSTALL_PRIVILEGE_REQUIRED"
	ErrorRuntimeInstallTimeout             ErrorCode = "RUNTIME_INSTALL_TIMEOUT"
	ErrorRuntimeInstallOutputLimit         ErrorCode = "RUNTIME_INSTALL_OUTPUT_LIMIT"
	ErrorRuntimeInstallPostcheckFailed     ErrorCode = "RUNTIME_INSTALL_POSTCHECK_FAILED"
	ErrorRuntimeInstallCancelled           ErrorCode = "RUNTIME_INSTALL_CANCELLED"
	ErrorRuntimeInstallNotReady            ErrorCode = "INSTALL_NOT_READY"
	ErrorRuntimeInstallBlockedUnsafe       ErrorCode = "INSTALL_BLOCKED_UNSAFE"
	ErrorOperationInterrupted              ErrorCode = "OPERATION_INTERRUPTED"
	ErrorOperationNotCancellable           ErrorCode = "OPERATION_NOT_CANCELLABLE"
	ErrorIdempotencyKeyConflict            ErrorCode = "IDEMPOTENCY_KEY_CONFLICT"

	ErrorHermesNodeMissing                ErrorCode = "RUNTIME_HERMES_NODE_MISSING"
	ErrorHermesNodeUnsupported            ErrorCode = "RUNTIME_HERMES_NODE_UNSUPPORTED"
	ErrorHermesNPMMissing                 ErrorCode = "RUNTIME_HERMES_NPM_MISSING"
	ErrorHermesNPMUnsupported             ErrorCode = "RUNTIME_HERMES_NPM_UNSUPPORTED"
	ErrorHermesNodeArchiveIntegrityFailed ErrorCode = "RUNTIME_HERMES_NODE_ARCHIVE_INTEGRITY_FAILED"
	ErrorHermesNPMArchiveIntegrityFailed  ErrorCode = "RUNTIME_HERMES_NPM_ARCHIVE_INTEGRITY_FAILED"
	ErrorHermesNodeDepsFailed             ErrorCode = "RUNTIME_HERMES_NODE_DEPS_FAILED"
	ErrorHermesNodeDepsTimeout            ErrorCode = "RUNTIME_HERMES_NODE_DEPS_TIMEOUT"

	ErrorRuntimeNotSupported          ErrorCode = "RUNTIME_NOT_SUPPORTED"
	ErrorInstanceInvalidName          ErrorCode = "INSTANCE_INVALID_NAME"
	ErrorInstanceAlreadyExists        ErrorCode = "INSTANCE_ALREADY_EXISTS"
	ErrorInstanceNotFound             ErrorCode = "INSTANCE_NOT_FOUND"
	ErrorInstanceNotAvailable         ErrorCode = "INSTANCE_NOT_AVAILABLE"
	ErrorInstanceProtected            ErrorCode = "INSTANCE_PROTECTED"
	ErrorInstanceConfirmationMismatch ErrorCode = "INSTANCE_CONFIRMATION_MISMATCH"
	ErrorInstanceConflict             ErrorCode = "INSTANCE_CONFLICT"
	ErrorInstanceQueryFailed          ErrorCode = "INSTANCE_QUERY_FAILED"
	ErrorInstanceOutputUnrecognized   ErrorCode = "INSTANCE_OUTPUT_UNRECOGNIZED"
	ErrorInstanceOperationTimedOut    ErrorCode = "INSTANCE_OPERATION_TIMED_OUT"
	ErrorCapabilityNotSupported       ErrorCode = "CAPABILITY_NOT_SUPPORTED"
	ErrorModelProviderUnsupported     ErrorCode = "MODEL_PROVIDER_UNSUPPORTED"
	ErrorModelConfigInvalid           ErrorCode = "MODEL_CONFIG_INVALID"
	ErrorModelConfigQueryFailed       ErrorCode = "MODEL_CONFIG_QUERY_FAILED"
	ErrorModelConfigApplyFailed       ErrorCode = "MODEL_CONFIG_APPLY_FAILED"
	ErrorModelConfigIncomplete        ErrorCode = "MODEL_CONFIG_INCOMPLETE"
	ErrorModelCredentialRequired      ErrorCode = "MODEL_CREDENTIAL_REQUIRED"
	ErrorModelCredentialQueryFailed   ErrorCode = "MODEL_CREDENTIAL_QUERY_FAILED"
	ErrorModelCredentialWriteFailed   ErrorCode = "MODEL_CREDENTIAL_WRITE_FAILED"
	ErrorModelCredentialDeleteFailed  ErrorCode = "MODEL_CREDENTIAL_DELETE_FAILED"
	ErrorInstanceConfigConflict       ErrorCode = "INSTANCE_CONFIG_CONFLICT"
)

type Candidate struct {
	Path      string
	Version   string
	State     DiscoveryState
	ErrorCode ErrorCode
}

type Warning struct {
	Code    string
	Message string
}

type Discovery struct {
	RuntimeKind    Kind
	State          DiscoveryState
	ErrorCode      ErrorCode
	Selected       *Candidate
	Candidates     []Candidate
	Warnings       []Warning
	DetectedAt     time.Time
	SupportedRange string
}

type Discoverer interface {
	Detect(ctx context.Context) (Discovery, error)
}
