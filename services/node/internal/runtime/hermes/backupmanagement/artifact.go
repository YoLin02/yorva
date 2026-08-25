package backupmanagement

const (
	// EncryptionQualified remains false until ADR-0013 and a reviewed age
	// implementation receive Owner approval and qualification evidence.
	EncryptionQualified = false
	// RestoreMutationQualified remains false because this package performs no
	// extraction, protection-point creation, Runtime mutation or rollback.
	RestoreMutationQualified = false
)

type ArtifactState string

const (
	ArtifactUnverified        ArtifactState = "UNVERIFIED"
	ArtifactStructureVerified ArtifactState = "STRUCTURE_VERIFIED_ONLY"
	ArtifactInvalid           ArtifactState = "INVALID"
)

type ArtifactMetadata struct {
	ArtifactSizeBytes  int64
	FormatVersion      string
	SchemaVersion      int
	PolicyVersion      int
	Scope              string
	RuntimeKind        string
	RuntimeVersion     string
	IncludedCategories []DataCategory
	PayloadSizeBytes   int64
	PayloadSHA256      string
	PayloadMemberCount int
	TotalExpandedBytes int64
}

type VerificationResult struct {
	State                    ArtifactState
	Metadata                 ArtifactMetadata
	ErrorCode                ErrorCode
	EncryptionQualified      bool
	RestoreMutationQualified bool
}

func unverifiedResult() VerificationResult {
	return VerificationResult{
		State:                    ArtifactUnverified,
		EncryptionQualified:      EncryptionQualified,
		RestoreMutationQualified: RestoreMutationQualified,
	}
}

func failedResult(code ErrorCode) VerificationResult {
	result := unverifiedResult()
	result.State = ArtifactInvalid
	result.ErrorCode = code
	return result
}
