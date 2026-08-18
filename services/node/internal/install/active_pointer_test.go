package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActivePointerRoundTrip(t *testing.T) {
	store := mustStore(t)
	if got := store.ReadActive(); !got.Missing() || got.Present || got.Valid {
		t.Fatalf("missing pointer: %#v", got)
	}
	rec := validActive(t)
	mustMaterializePointerGeneration(t, store, &rec)
	if err := store.WriteActive(rec); err != nil {
		t.Fatal(err)
	}
	got := store.ReadActive()
	if !got.IsValid() || got.GenerationID != rec.GenerationID {
		t.Fatalf("read %#v", got)
	}
	loaded, err := store.LoadActive()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TransactionID != rec.TransactionID || loaded.GenerationRelativePath != GenerationRel(rec.GenerationID) {
		t.Fatalf("loaded %#v", loaded)
	}
}

func TestActivePointerRejectsUnsafePaths(t *testing.T) {
	store := mustStore(t)
	rec := validActive(t)
	rec.GenerationRelativePath = `C:\Windows\generations\` + rec.GenerationID
	if err := store.WriteActive(rec); err == nil {
		t.Fatal("absolute path accepted")
	}
	rec = validActive(t)
	rec.GenerationRelativePath = "generations/../control/" + rec.GenerationID
	if err := store.WriteActive(rec); err == nil {
		t.Fatal("escaping path accepted")
	}
	rec = validActive(t)
	rec.GenerationRelativePath = "generations/" + rec.GenerationID + "/extra"
	if err := store.WriteActive(rec); err == nil {
		t.Fatal("nested generation path accepted")
	}
}

func TestActivePointerMalformedIsNotNewest(t *testing.T) {
	store := mustStore(t)
	if err := store.layout.EnsureControl(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.layout.ActivePath(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := store.ReadActive()
	if !got.Invalid() || got.Valid || got.GenerationID != "" {
		t.Fatalf("malformed treated as valid: %#v", got)
	}
}

func TestActivePointerPostReplaceReadback(t *testing.T) {
	root := t.TempDir()
	store := mustStoreAt(t, root)
	first := validActive(t)
	mustMaterializePointerGeneration(t, store, &first)
	if err := store.WriteActive(first); err != nil {
		t.Fatal(err)
	}
	second := validActive(t)
	mustMaterializePointerGeneration(t, store, &second)
	failing := mustStoreWithHook(t, root, failAt(stepAfterReplace))
	if err := failing.WriteActive(second); err == nil {
		t.Fatal("injected failure succeeded")
	}
	got := store.ReadActive()
	if !got.Valid || got.GenerationID != second.GenerationID {
		t.Fatalf("complete new pointer not recovered: %#v", got)
	}
}

func mustMaterializePointerGeneration(t *testing.T, store *Store, rec *ActiveRecord) {
	t.Helper()
	genAbs, err := store.layout.GenerationPath(rec.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(genAbs, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := json.Marshal(ManifestFile{Schema: manifestSchema})
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	genRec := GenerationRecord{
		Schema:                 generationSchema,
		LineageID:              strings.Repeat("11", 16),
		TransactionID:          rec.TransactionID,
		GenerationID:           rec.GenerationID,
		RuntimeKind:            rec.RuntimeKind,
		SourcePin:              rec.SourcePin,
		ExpectedVersion:        rec.Version,
		GenerationRelativePath: rec.GenerationRelativePath,
		ManifestSHA256:         sha256Hex(manifestBytes),
		CreatedAt:              rec.ActivatedAt,
		SealedAt:               rec.ActivatedAt,
	}
	genBytes, err := json.Marshal(genRec)
	if err != nil {
		t.Fatal(err)
	}
	genBytes = append(genBytes, '\n')
	if err := os.WriteFile(filepath.Join(genAbs, fileManifest), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(genAbs, fileGeneration), genBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	rec.ManifestSHA256 = sha256Hex(manifestBytes)
	rec.SealSHA256 = sha256Hex(genBytes)
}

func validActive(t *testing.T) ActiveRecord {
	t.Helper()
	txnID, err := NewTransactionID()
	if err != nil {
		t.Fatal(err)
	}
	genID, err := NewGenerationID()
	if err != nil {
		t.Fatal(err)
	}
	return ActiveRecord{
		Schema:                 1,
		RuntimeKind:            "hermes",
		GenerationID:           genID,
		GenerationRelativePath: GenerationRel(genID),
		ManifestSHA256:         strings.Repeat("ab", 32),
		SealSHA256:             strings.Repeat("cd", 32),
		SourcePin:              "df4b65147d7ddd74dd449f9067aabbca5aef0ec7",
		Version:                "0.20.2",
		TransactionID:          txnID,
		ActivatedAt:            time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC),
	}
}
