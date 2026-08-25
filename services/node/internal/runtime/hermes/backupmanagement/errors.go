package backupmanagement

import "errors"

// ErrorCode is a stable, safe classification for structural verification
// failures. It is suitable for adapter-level normalization; the Error string
// intentionally contains no archive member name or parser detail.
type ErrorCode string

const (
	ErrorInputInvalid               ErrorCode = "BACKUP_VERIFIER_INPUT_INVALID"
	ErrorManifestInvalid            ErrorCode = "BACKUP_MANIFEST_INVALID"
	ErrorManifestNotCanonical       ErrorCode = "BACKUP_MANIFEST_NOT_CANONICAL"
	ErrorFormatUnsupported          ErrorCode = "BACKUP_FORMAT_UNSUPPORTED"
	ErrorScopeUnsupported           ErrorCode = "BACKUP_SCOPE_UNSUPPORTED"
	ErrorRuntimeUnsupported         ErrorCode = "BACKUP_RUNTIME_UNSUPPORTED"
	ErrorArchiveInvalid             ErrorCode = "BACKUP_ARCHIVE_TRUNCATED_OR_MALFORMED"
	ErrorArchiveMemberLimit         ErrorCode = "BACKUP_ARCHIVE_MEMBER_LIMIT"
	ErrorArchiveMemberSizeLimit     ErrorCode = "BACKUP_ARCHIVE_MEMBER_SIZE_LIMIT"
	ErrorArchiveTotalSizeLimit      ErrorCode = "BACKUP_ARCHIVE_TOTAL_SIZE_LIMIT"
	ErrorArchiveCompressionLimit    ErrorCode = "BACKUP_ARCHIVE_COMPRESSION_RATIO_LIMIT"
	ErrorArchivePathUnsafe          ErrorCode = "BACKUP_ARCHIVE_PATH_UNSAFE"
	ErrorArchiveTypeUnsafe          ErrorCode = "BACKUP_ARCHIVE_TYPE_UNSAFE"
	ErrorArchiveDuplicateMember     ErrorCode = "BACKUP_ARCHIVE_DUPLICATE_MEMBER"
	ErrorArchiveCaseCollision       ErrorCode = "BACKUP_ARCHIVE_CASE_COLLISION"
	ErrorArchiveRootUnsupported     ErrorCode = "BACKUP_ARCHIVE_ROOT_UNSUPPORTED"
	ErrorPayloadMissing             ErrorCode = "BACKUP_PAYLOAD_MISSING"
	ErrorPayloadSizeMismatch        ErrorCode = "BACKUP_PAYLOAD_SIZE_MISMATCH"
	ErrorPayloadChecksumMismatch    ErrorCode = "BACKUP_PAYLOAD_CHECKSUM_MISMATCH"
	ErrorPayloadMemberCountMismatch ErrorCode = "BACKUP_PAYLOAD_MEMBER_COUNT_MISMATCH"
	ErrorPayloadExpandedMismatch    ErrorCode = "BACKUP_PAYLOAD_EXPANDED_SIZE_MISMATCH"
)

var ErrVerification = errors.New("backup structural verification failed")

// VerificationError exposes only a stable code and safe fixed message.
type VerificationError struct {
	code ErrorCode
}

func (e *VerificationError) Error() string {
	return string(e.code) + ": " + errorMessage(e.code)
}

func (e *VerificationError) Unwrap() error { return ErrVerification }

func (e *VerificationError) Code() ErrorCode { return e.code }

// ErrorCodeOf obtains the stable code without requiring callers to inspect an
// error string.
func ErrorCodeOf(err error) (ErrorCode, bool) {
	var verificationError *VerificationError
	if !errors.As(err, &verificationError) {
		return "", false
	}
	return verificationError.code, true
}

func verificationError(code ErrorCode) error {
	return &VerificationError{code: code}
}

func errorMessage(code ErrorCode) string {
	switch code {
	case ErrorInputInvalid:
		return "the verifier input or limits are invalid"
	case ErrorManifestInvalid:
		return "the backup manifest is invalid"
	case ErrorManifestNotCanonical:
		return "the backup manifest is not canonical"
	case ErrorFormatUnsupported:
		return "the backup format or policy is unsupported"
	case ErrorScopeUnsupported:
		return "the backup scope is unsupported"
	case ErrorRuntimeUnsupported:
		return "the backup runtime identity or version is unsupported"
	case ErrorArchiveInvalid:
		return "the backup archive is truncated or malformed"
	case ErrorArchiveMemberLimit:
		return "the backup archive member limit was exceeded"
	case ErrorArchiveMemberSizeLimit:
		return "a backup archive member exceeded its size limit"
	case ErrorArchiveTotalSizeLimit:
		return "the backup archive expanded-size limit was exceeded"
	case ErrorArchiveCompressionLimit:
		return "the backup archive compression-ratio limit was exceeded"
	case ErrorArchivePathUnsafe:
		return "the backup archive contains an unsafe member path"
	case ErrorArchiveTypeUnsafe:
		return "the backup archive contains an unsafe member type"
	case ErrorArchiveDuplicateMember:
		return "the backup archive contains a duplicate member"
	case ErrorArchiveCaseCollision:
		return "the backup archive contains a case-folding collision"
	case ErrorArchiveRootUnsupported:
		return "the backup archive contains an unsupported root"
	case ErrorPayloadMissing:
		return "the backup payload is missing"
	case ErrorPayloadSizeMismatch:
		return "the backup payload size does not match its manifest"
	case ErrorPayloadChecksumMismatch:
		return "the backup payload checksum does not match its manifest"
	case ErrorPayloadMemberCountMismatch:
		return "the backup payload member count does not match its manifest"
	case ErrorPayloadExpandedMismatch:
		return "the backup payload expanded size does not match its manifest"
	default:
		return "backup structural verification failed"
	}
}
