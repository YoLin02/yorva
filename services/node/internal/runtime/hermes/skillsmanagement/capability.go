package skillsmanagement

import "errors"

const QualifiedHermesVersion = "0.20.5"

type MutationOperation string

const (
	MutationInstall   MutationOperation = "install"
	MutationUpdate    MutationOperation = "update"
	MutationRemove    MutationOperation = "remove"
	MutationConfigure MutationOperation = "configure"
)

type MutationErrorCode string

const (
	MutationInstallUnavailable     MutationErrorCode = "SKILL_INSTALL_UNAVAILABLE"
	MutationUpdateScanBypassUnsafe MutationErrorCode = "SKILL_UPDATE_SCAN_BYPASS_UNSAFE"
	MutationRemoveUnavailable      MutationErrorCode = "SKILL_REMOVE_UNAVAILABLE"
	MutationConfigureUnavailable   MutationErrorCode = "SKILL_CONFIGURE_UNAVAILABLE"
	MutationVersionUnqualified     MutationErrorCode = "SKILL_MUTATION_VERSION_UNQUALIFIED"
	MutationOperationUnknown       MutationErrorCode = "SKILL_MUTATION_OPERATION_UNKNOWN"
)

const (
	installUnavailableReason   = "Hermes 0.20.5 has no closed approved-source install result with a structured completed postcondition and authoritative read-back"
	updateUnavailableReason    = "Hermes 0.20.5 Skill update internally selects force=True and can bypass a blocked security scan"
	removeUnavailableReason    = "Hermes 0.20.5 Skill removal has no structured completed result or authoritative read-back"
	configureUnavailableReason = "Hermes 0.20.5 Skill configuration requires an interactive terminal and has no qualified closed schema"
	versionUnavailableReason   = "this Hermes version has no qualified Skill mutation surface"
	unknownOperationReason     = "the requested Skill mutation is not a closed supported operation"
)

var ErrMutationUnavailable = errors.New("Hermes Skill mutation capability is unavailable")

type MutationCapability struct {
	operation MutationOperation
	available bool
	code      MutationErrorCode
	reason    string
}

func (c MutationCapability) Operation() MutationOperation { return c.operation }
func (c MutationCapability) Available() bool              { return c.available }
func (c MutationCapability) ErrorCode() MutationErrorCode { return c.code }
func (c MutationCapability) Reason() string               { return c.reason }

type MutationUnavailableError struct {
	capability MutationCapability
}

func (e *MutationUnavailableError) Error() string {
	return string(e.capability.code) + ": " + e.capability.reason
}

func (e *MutationUnavailableError) Unwrap() error {
	return ErrMutationUnavailable
}

func (e *MutationUnavailableError) Operation() MutationOperation {
	return e.capability.operation
}

func (e *MutationUnavailableError) ErrorCode() MutationErrorCode {
	return e.capability.code
}

func (e *MutationUnavailableError) Reason() string {
	return e.capability.reason
}

// MutationCapabilityFor reports capability truth only. There is deliberately
// no executor, command descriptor, force flag, or hidden override in this
// package.
func MutationCapabilityFor(runtimeVersion string, operation MutationOperation) MutationCapability {
	if runtimeVersion != QualifiedHermesVersion {
		return unavailableCapability(operation, MutationVersionUnqualified, versionUnavailableReason)
	}
	switch operation {
	case MutationInstall:
		return unavailableCapability(operation, MutationInstallUnavailable, installUnavailableReason)
	case MutationUpdate:
		return unavailableCapability(operation, MutationUpdateScanBypassUnsafe, updateUnavailableReason)
	case MutationRemove:
		return unavailableCapability(operation, MutationRemoveUnavailable, removeUnavailableReason)
	case MutationConfigure:
		return unavailableCapability(operation, MutationConfigureUnavailable, configureUnavailableReason)
	default:
		return unavailableCapability(operation, MutationOperationUnknown, unknownOperationReason)
	}
}

func RequireMutation(runtimeVersion string, operation MutationOperation) error {
	capability := MutationCapabilityFor(runtimeVersion, operation)
	if capability.available {
		return nil
	}
	return &MutationUnavailableError{capability: capability}
}

func unavailableCapability(operation MutationOperation, code MutationErrorCode, reason string) MutationCapability {
	return MutationCapability{
		operation: operation,
		available: false,
		code:      code,
		reason:    reason,
	}
}
