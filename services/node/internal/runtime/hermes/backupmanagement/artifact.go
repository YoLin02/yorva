package backupmanagement

const (
	// EncryptionQualified is deliberately false on structural verification
	// results. Product capability is represented only by qualified Bundle
	// wiring; parsing an artifact must never grant that authority.
	EncryptionQualified = false
	// RestoreMutationQualified follows the same rule: artifact verification is
	// a prerequisite, not authorization to mutate Runtime state.
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
	BackupID           string
	CreatedAt          string
	Scope              string
	RuntimeKind        string
	RuntimeVersion     string
	Installation       InstallationIdentity
	InclusionPolicy    string
	IncludedCategories []DataCategory
	ExclusionPolicy    string
	ExcludedCategories []ExcludedDataCategory
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
