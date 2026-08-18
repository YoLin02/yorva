package install

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestActivePointerRoundTrip(t *testing.T) {
	store := mustStore(t)
	if got := store.ReadActive(); got.Present || got.Valid {
		t.Fatalf("missing pointer: %#v", got)
	}
	rec := validActive(t)
	if err := store.WriteActive(rec); err != nil {
		t.Fatal(err)
	}
	got := store.ReadActive()
	if !got.Valid || got.GenerationID != rec.GenerationID {
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
	if !got.Present || got.Valid || got.GenerationID != "" {
		t.Fatalf("malformed treated as valid: %#v", got)
	}
}

func TestActivePointerPostReplaceReadback(t *testing.T) {
	root := t.TempDir()
	store := mustStoreAt(t, root)
	first := validActive(t)
	if err := store.WriteActive(first); err != nil {
		t.Fatal(err)
	}
	second := validActive(t)
	failing := mustStoreWithHook(t, root, failAt(stepAfterReplace))
	if err := failing.WriteActive(second); err == nil {
		t.Fatal("injected failure succeeded")
	}
	got := store.ReadActive()
	if !got.Valid || got.GenerationID != second.GenerationID {
		t.Fatalf("complete new pointer not recovered: %#v", got)
	}
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
