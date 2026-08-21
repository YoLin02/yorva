package install

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SealInput struct {
	RootAbs                string
	TransactionID          string
	GenerationID           string
	RuntimeKind            string
	SourcePin              string
	ExpectedVersion        string
	GenerationRelativePath string
	CreatedAt              time.Time
}

type SealResult struct {
	ManifestSHA256 string
	SealSHA256     string
	LineageID      string
	SealedAt       time.Time
}

type sealHooks struct {
	AfterWalk        func() error
	AfterManifest    func() error
	BeforeGeneration func() error
	BeforeSecondWalk func() error
}

func SealGeneration(ops atomicOps, in SealInput, hooks sealHooks) (SealResult, error) {
	if err := ValidateGenerationRel(in.GenerationRelativePath, in.GenerationID); err != nil {
		return SealResult{}, err
	}
	if err := ParseTransactionID(in.TransactionID); err != nil {
		return SealResult{}, err
	}
	if err := rejectReparse(in.RootAbs); err != nil {
		return SealResult{}, err
	}
	entries, err := walkSealManifest(in.RootAbs)
	if err != nil {
		return SealResult{}, err
	}
	if hooks.AfterWalk != nil {
		if err := hooks.AfterWalk(); err != nil {
			return SealResult{}, err
		}
	}
	manifest := ManifestFile{Schema: manifestSchema, Entries: entries}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return SealResult{}, err
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestPath := filepath.Join(in.RootAbs, fileManifest)
	if err := writeAtomicRecord(ops, manifestPath, manifestBytes); err != nil {
		return SealResult{}, err
	}
	if hooks.AfterManifest != nil {
		if err := hooks.AfterManifest(); err != nil {
			_ = os.Remove(manifestPath)
			return SealResult{}, err
		}
	}
	if hooks.BeforeGeneration != nil {
		if err := hooks.BeforeGeneration(); err != nil {
			_ = os.Remove(manifestPath)
			return SealResult{}, err
		}
	}
	lineage, err := newLineageID()
	if err != nil {
		_ = os.Remove(manifestPath)
		return SealResult{}, err
	}
	now := in.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rec := GenerationRecord{
		Schema:                 generationSchema,
		LineageID:              lineage,
		TransactionID:          in.TransactionID,
		GenerationID:           in.GenerationID,
		RuntimeKind:            in.RuntimeKind,
		SourcePin:              in.SourcePin,
		ExpectedVersion:        in.ExpectedVersion,
		GenerationRelativePath: in.GenerationRelativePath,
		ManifestSHA256:         sha256Hex(manifestBytes),
		CreatedAt:              now,
		SealedAt:               now,
	}
	if err := ValidateGenerationRel(rec.GenerationRelativePath, rec.GenerationID); err != nil {
		_ = os.Remove(manifestPath)
		return SealResult{}, err
	}
	genBytes, err := json.Marshal(rec)
	if err != nil {
		_ = os.Remove(manifestPath)
		return SealResult{}, err
	}
	genBytes = append(genBytes, '\n')
	genPath := filepath.Join(in.RootAbs, fileGeneration)
	if err := writeAtomicRecord(ops, genPath, genBytes); err != nil {
		_ = os.Remove(manifestPath)
		return SealResult{}, err
	}
	if hooks.BeforeSecondWalk != nil {
		if err := hooks.BeforeSecondWalk(); err != nil {
			removeSealFiles(in.RootAbs)
			return SealResult{}, err
		}
	}
	again, err := walkSealManifest(in.RootAbs)
	if err != nil || !manifestsEqual(entries, again) {
		removeSealFiles(in.RootAbs)
		if err != nil {
			return SealResult{}, err
		}
		return SealResult{}, ErrSealInvalid
	}
	return SealResult{
		ManifestSHA256: rec.ManifestSHA256,
		SealSHA256:     sha256Hex(genBytes),
		LineageID:      lineage,
		SealedAt:       rec.SealedAt,
	}, nil
}

func walkSealManifest(root string) ([]ManifestEntry, error) {
	var entries []ManifestEntry
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := rejectReparse(path); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return ErrReparsePoint
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if slash == fileManifest || slash == fileGeneration {
			return nil
		}
		sum, size, err := hashRegularFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, ManifestEntry{Path: slash, Size: size, SHA256: sum})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func manifestsEqual(a, b []ManifestEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hashRegularFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	sum := sha256.New()
	n, err := io.Copy(sum, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(sum.Sum(nil)), n, nil
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func newLineageID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func removeSealFiles(root string) {
	_ = os.Remove(filepath.Join(root, fileManifest))
	_ = os.Remove(filepath.Join(root, fileGeneration))
}
