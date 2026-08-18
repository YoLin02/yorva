package install

import "time"

const (
	generationSchema = 1
	manifestSchema   = 1
	fileManifest     = "manifest.json"
	fileGeneration   = "generation.json"
)

// ManifestFile is the integrity list for a sealed staging/generation tree.
type ManifestFile struct {
	Schema  int             `json:"schema"`
	Entries []ManifestEntry `json:"entries"`
}

type ManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// GenerationRecord is lineage. It is not anti-malware and not an activation pointer.
type GenerationRecord struct {
	Schema                 int       `json:"schema"`
	LineageID              string    `json:"lineageId"`
	TransactionID          string    `json:"transactionId"`
	GenerationID           string    `json:"generationId"`
	RuntimeKind            string    `json:"runtimeKind"`
	SourcePin              string    `json:"sourcePin"`
	ExpectedVersion        string    `json:"expectedVersion"`
	GenerationRelativePath string    `json:"generationRelativePath"`
	ManifestSHA256         string    `json:"manifestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
	SealedAt               time.Time `json:"sealedAt"`
}
