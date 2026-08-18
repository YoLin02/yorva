package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSealIncludesAllFilesExceptSealArtifacts(t *testing.T) {
	staging := t.TempDir()
	writeMinimalStaging(t, staging)
	atom := filepath.Join(staging, ".yorva-atom-deadbeef")
	if err := os.WriteFile(atom, []byte("temp-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	txn := mustCreated(t)
	got, err := SealStaging(defaultAtomicOps(), sealInput(txn, staging), sealHooks{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ManifestSHA256 == "" || got.SealSHA256 == "" || got.LineageID == "" {
		t.Fatalf("incomplete seal %#v", got)
	}
	payload, err := os.ReadFile(filepath.Join(staging, fileManifest))
	if err != nil {
		t.Fatal(err)
	}
	var manifest ManifestFile
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range manifest.Entries {
		seen[entry.Path] = true
		if entry.Path == fileManifest || entry.Path == fileGeneration {
			t.Fatal("seal artifacts must not be listed")
		}
	}
	if !seen[".yorva-atom-deadbeef"] {
		t.Fatal("prefix ignore leaked into seal walk")
	}
	if !seen["bin/hermes.exe"] || !seen["bin/hermes-acp.exe"] {
		t.Fatal("launchers missing from manifest")
	}
}

func TestSealSecondWalkMismatchRemovesSealFiles(t *testing.T) {
	staging := t.TempDir()
	writeMinimalStaging(t, staging)
	txn := mustCreated(t)
	_, err := SealStaging(defaultAtomicOps(), sealInput(txn, staging), sealHooks{
		BeforeSecondWalk: func() error {
			return os.WriteFile(filepath.Join(staging, "mutated-after-seal.txt"), []byte("x"), 0o600)
		},
	})
	if !errors.Is(err, ErrSealInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(staging, fileManifest)); !os.IsNotExist(err) {
		t.Fatal("manifest left after invalid second walk")
	}
	if _, err := os.Stat(filepath.Join(staging, fileGeneration)); !os.IsNotExist(err) {
		t.Fatal("generation.json left after invalid second walk")
	}
}

func TestSealStopsBeforeGenerationJSON(t *testing.T) {
	staging := t.TempDir()
	writeMinimalStaging(t, staging)
	txn := mustCreated(t)
	_, err := SealStaging(defaultAtomicOps(), sealInput(txn, staging), sealHooks{
		BeforeGeneration: func() error { return errInjected },
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Stat(filepath.Join(staging, fileGeneration)); !os.IsNotExist(err) {
		t.Fatal("generation.json written before hook")
	}
}

func sealInput(txn InstallTransaction, staging string) SealInput {
	return SealInput{
		StagingAbs:             staging,
		TransactionID:          txn.ID,
		GenerationID:           txn.GenerationID,
		RuntimeKind:            txn.RuntimeKind,
		SourcePin:              txn.SourcePin,
		ExpectedVersion:        txn.ExpectedVersion,
		GenerationRelativePath: txn.GenerationRelativePath,
		CreatedAt:              time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
	}
}

func writeMinimalStaging(t testing.TB, root string) {
	t.Helper()
	if err := writeMinimalStagingTree(root); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalStagingTree(root string) error {
	files := map[string]string{
		"LICENSE":                       "license",
		"pyproject.toml":                `version = "0.20.2"`,
		"scripts/install.ps1":           "script",
		"hermes_cli/main.py":            "pass",
		"venv/Scripts/hermes.exe":       "mz-hermes",
		"venv/Scripts/hermes-acp.exe":   "mz-acp",
		"bin/hermes.exe":                "mz-hermes",
		"bin/hermes-acp.exe":            "mz-acp",
		".hermes-bootstrap-complete":    "ok",
		"venv/Lib/site-packages/ok.txt": "dep",
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return err
		}
	}
	return nil
}
