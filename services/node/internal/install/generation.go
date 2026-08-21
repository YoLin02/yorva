package install

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	generationSchema = 1
	manifestSchema   = 1
	fileManifest     = "manifest.json"
	fileGeneration   = "generation.json"
	fileCandidate    = ".yorva-candidate.json"
)

const candidateSchema = 1
const candidateRecordMaxBytes = 4 * 1024

// CandidateRecord proves that YORVA created an inactive generation directory for
// one exact transaction. It is included in the final manifest but is never an
// activation pointer.
type CandidateRecord struct {
	Schema        int       `json:"schema"`
	TransactionID string    `json:"transactionId"`
	GenerationID  string    `json:"generationId"`
	RuntimeKind   string    `json:"runtimeKind"`
	CreatedAt     time.Time `json:"createdAt"`
}

func writeCandidateRecord(ops atomicOps, root string, txn InstallTransaction) error {
	rec := CandidateRecord{
		Schema:        candidateSchema,
		TransactionID: txn.ID,
		GenerationID:  txn.GenerationID,
		RuntimeKind:   txn.RuntimeKind,
		CreatedAt:     txn.CreatedAt,
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return writeAtomicRecord(ops, filepath.Join(root, fileCandidate), payload)
}

func readCandidateRecord(root string) (CandidateRecord, bool) {
	path := filepath.Join(root, fileCandidate)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || isReparsePoint(info) || info.Size() > candidateRecordMaxBytes {
		return CandidateRecord{}, false
	}
	if err := rejectReparse(path); err != nil {
		return CandidateRecord{}, false
	}
	file, err := os.Open(path)
	if err != nil {
		return CandidateRecord{}, false
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, candidateRecordMaxBytes+1))
	if err != nil || len(payload) > candidateRecordMaxBytes {
		return CandidateRecord{}, false
	}
	var rec CandidateRecord
	if json.Unmarshal(payload, &rec) != nil || rec.Schema != candidateSchema ||
		ParseTransactionID(rec.TransactionID) != nil || ParseGenerationID(rec.GenerationID) != nil ||
		rec.RuntimeKind == "" || rec.CreatedAt.IsZero() {
		return CandidateRecord{}, false
	}
	return rec, true
}

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
