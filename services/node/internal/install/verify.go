package install

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxActiveBytes = 64 << 10

func VerifyPublishedGeneration(genAbs, generationID, manifestSHA, sealSHA string) error {
	_, _, err := loadPublishedSeal(genAbs, generationID, manifestSHA, sealSHA)
	return err
}

// VerifySealedTree is the publish/activate integrity check: metadata hashes
// plus a second full walk that must match the stored manifest. Discovery may
// use VerifyPublishedGeneration (002A3 metadata-only pointer validity).
func VerifySealedTree(genAbs, generationID, manifestSHA, sealSHA string) error {
	_, manifest, err := loadPublishedSeal(genAbs, generationID, manifestSHA, sealSHA)
	if err != nil {
		return err
	}
	if manifest.Schema != manifestSchema {
		return ErrSealInvalid
	}
	entries, err := walkSealManifest(genAbs)
	if err != nil || !manifestsEqual(entries, manifest.Entries) {
		return ErrSealInvalid
	}
	return nil
}

func loadPublishedSeal(genAbs, generationID, manifestSHA, sealSHA string) (GenerationRecord, ManifestFile, error) {
	var rec GenerationRecord
	var manifest ManifestFile
	if err := ParseGenerationID(generationID); err != nil {
		return rec, manifest, err
	}
	if err := rejectReparse(genAbs); err != nil {
		return rec, manifest, err
	}
	genPath := filepath.Join(genAbs, fileGeneration)
	manPath := filepath.Join(genAbs, fileManifest)
	if !isRegularFile(genPath) || !isRegularFile(manPath) {
		return rec, manifest, ErrSealInvalid
	}
	genBytes, err := os.ReadFile(genPath)
	if err != nil {
		return rec, manifest, err
	}
	manBytes, err := os.ReadFile(manPath)
	if err != nil {
		return rec, manifest, err
	}
	if sha256Hex(manBytes) != manifestSHA || sha256Hex(genBytes) != sealSHA {
		return rec, manifest, ErrSealInvalid
	}
	if err := json.Unmarshal(genBytes, &rec); err != nil {
		return rec, manifest, ErrSealInvalid
	}
	if rec.GenerationID != generationID || rec.ManifestSHA256 != manifestSHA {
		return rec, manifest, ErrSealInvalid
	}
	if err := ValidateGenerationRel(rec.GenerationRelativePath, rec.GenerationID); err != nil {
		return rec, manifest, err
	}
	if err := json.Unmarshal(manBytes, &manifest); err != nil {
		return rec, manifest, ErrSealInvalid
	}
	return rec, manifest, nil
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
