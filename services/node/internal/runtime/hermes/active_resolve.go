package hermes

import (
	"os"
	"path/filepath"

	"github.com/YoLin02/yorva/services/node/internal/install"
)

type activeGeneration struct {
	officialPaths     []string
	installationRoots []string
}

func resolveActiveGeneration(localAppData string) (activeGeneration, bool) {
	if localAppData == "" {
		return activeGeneration{}, false
	}
	root := filepath.Join(localAppData, "hermes")
	store, err := install.NewStore(root)
	if err != nil {
		return activeGeneration{}, false
	}
	rec, err := store.LoadActive()
	if err != nil || rec.RuntimeKind != string(Kind) {
		return activeGeneration{}, false
	}
	genAbs, err := store.Layout().GenerationPath(rec.GenerationID)
	if err != nil {
		return activeGeneration{}, false
	}
	if err := install.VerifyPublishedGeneration(genAbs, rec.GenerationID, rec.ManifestSHA256, rec.SealSHA256); err != nil {
		return activeGeneration{}, false
	}
	bin := filepath.Join(genAbs, "bin", "hermes.exe")
	venv := filepath.Join(genAbs, "venv", "Scripts", "hermes.exe")
	if _, err := os.Lstat(filepath.Join(genAbs, "bin")); err != nil && !os.IsNotExist(err) {
		return activeGeneration{}, false
	}
	return activeGeneration{
		officialPaths:     []string{bin, venv},
		installationRoots: []string{genAbs},
	}, true
}
