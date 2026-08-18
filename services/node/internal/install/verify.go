package install

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxActiveBytes = 64 << 10

func VerifyPublishedGeneration(genAbs, generationID, manifestSHA, sealSHA string) error {
	if err := ParseGenerationID(generationID); err != nil {
		return err
	}
	if err := rejectReparse(genAbs); err != nil {
		return err
	}
	genPath := filepath.Join(genAbs, fileGeneration)
	manPath := filepath.Join(genAbs, fileManifest)
	if !isRegularFile(genPath) || !isRegularFile(manPath) {
		return ErrSealInvalid
	}
	genBytes, err := os.ReadFile(genPath)
	if err != nil {
		return err
	}
	manBytes, err := os.ReadFile(manPath)
	if err != nil {
		return err
	}
	if sha256Hex(manBytes) != manifestSHA || sha256Hex(genBytes) != sealSHA {
		return ErrSealInvalid
	}
	var rec GenerationRecord
	if err := json.Unmarshal(genBytes, &rec); err != nil {
		return ErrSealInvalid
	}
	if rec.GenerationID != generationID || rec.ManifestSHA256 != manifestSHA {
		return ErrSealInvalid
	}
	return ValidateGenerationRel(rec.GenerationRelativePath, rec.GenerationID)
}

func isRegularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && !isReparsePoint(info)
}

func dirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
