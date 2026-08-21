package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGCKeepsD4RedCases(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())

	prev := mustPublish(t, mgr)
	prev, err := mgr.ReconcileEnvironment(context.Background(), prev)
	if err != nil {
		t.Fatal(err)
	}
	if prev.State != StateCommitted {
		t.Fatalf("prev %#v", prev)
	}
	active := mustPublish(t, mgr)
	active, err = mgr.ReconcileEnvironment(context.Background(), active)
	if err != nil {
		t.Fatal(err)
	}

	failed := mustSeal(t, mgr)
	failedDir, err := store.layout.FailedPath(failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := store.layout.GenerationPath(failed.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(failedDir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(generation, failedDir); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadTransaction(failed.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.State = StateFailed
	loaded.UpdatedAt = time.Now().UTC()
	if err := store.SaveTransaction(loaded); err != nil {
		t.Fatal(err)
	}

	olderFailed := mustSeal(t, mgr)
	olderGeneration, err := store.layout.GenerationPath(olderFailed.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	olderDest, err := store.layout.FailedPath(olderFailed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(olderGeneration, olderDest); err != nil {
		t.Fatal(err)
	}
	older, err := store.LoadTransaction(olderFailed.ID)
	if err != nil {
		t.Fatal(err)
	}
	older.State = StateFailed
	older.UpdatedAt = time.Now().UTC().Add(-time.Hour)
	if err := store.SaveTransaction(older); err != nil {
		t.Fatal(err)
	}

	legacy := filepath.Join(store.layout.Root, "hermes-agent", "bin")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "hermes.exe"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".env", "config.yaml", "SOUL.md", "skills", "sessions", "logs"} {
		path := filepath.Join(store.layout.Root, name)
		if name == "skills" || name == "sessions" || name == "logs" {
			if err := os.MkdirAll(path, 0o700); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte("user"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mystery := filepath.Join(store.layout.Root, "mystery")
	if err := os.Mkdir(mystery, 0o700); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(store.layout.GenerationsRoot(), "gen_"+repeatA(22))
	if err := os.MkdirAll(filepath.Join(orphan, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}

	report, err := Collect(store, GCHooks{})
	if err != nil {
		t.Fatal(err)
	}

	mustExist := []string{
		mustGen(t, store, active.GenerationID),
		mustGen(t, store, prev.GenerationID),
		failedDir,
		filepath.Join(legacy, "hermes.exe"),
		filepath.Join(store.layout.Root, ".env"),
		filepath.Join(store.layout.Root, "config.yaml"),
		filepath.Join(store.layout.Root, "SOUL.md"),
		filepath.Join(store.layout.Root, "skills"),
		filepath.Join(store.layout.Root, "sessions"),
		filepath.Join(store.layout.Root, "logs"),
		mystery,
		orphan,
	}
	for _, path := range mustExist {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("D4 red path missing %s report=%#v", path, report)
		}
	}
	if _, err := os.Stat(olderDest); !os.IsNotExist(err) {
		t.Fatalf("older failed tree was retained: %s", olderDest)
	}
	loadedActive, err := store.LoadTransaction(active.ID)
	if err != nil || loadedActive.State != StateCommitted {
		t.Fatalf("GC changed committed txn %#v %v", loadedActive, err)
	}
}

func TestGCFailpointDoesNotChangeCommitted(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	txn, err := mgr.ReconcileEnvironment(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	newer := mustSeal(t, mgr)
	newerLoaded, err := store.LoadTransaction(newer.ID)
	if err != nil {
		t.Fatal(err)
	}
	newerLoaded.State = StateFailed
	newerLoaded.UpdatedAt = time.Now().UTC()
	if err := store.SaveTransaction(newerLoaded); err != nil {
		t.Fatal(err)
	}
	older := mustSeal(t, mgr)
	olderLoaded, err := store.LoadTransaction(older.ID)
	if err != nil {
		t.Fatal(err)
	}
	olderLoaded.State = StateFailed
	olderLoaded.UpdatedAt = time.Now().UTC().Add(-time.Hour)
	if err := store.SaveTransaction(olderLoaded); err != nil {
		t.Fatal(err)
	}
	_, err = collectWithFail(store, GCHooks{}, FailGCBeforeDelete)
	if err == nil {
		t.Fatal("expected failpoint")
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil || loaded.State != StateCommitted {
		t.Fatalf("committed changed %#v %v", loaded, err)
	}
}
