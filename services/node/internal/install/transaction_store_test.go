package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveLoadTransactionCAS(t *testing.T) {
	store := mustStore(t)
	txn := mustCreated(t)
	if err := store.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 1 || loaded.State != StateCreated {
		t.Fatalf("loaded %#v", loaded)
	}
	stale := loaded
	stale.Revision = 0
	if err := store.SaveTransaction(stale); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale write: %v", err)
	}
	loaded.State = StateBuilding
	if err := store.SaveTransaction(loaded); err != nil {
		t.Fatal(err)
	}
	again, err := store.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.Revision != 2 || again.State != StateBuilding {
		t.Fatalf("after update %#v", again)
	}
}

func TestAtomicFailpointsPreserveOldOrForwardNew(t *testing.T) {
	root := t.TempDir()
	store := mustStoreAt(t, root)
	txn := mustCreated(t)
	if err := store.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(mustTxnPath(t, store, txn.ID))
	if err != nil {
		t.Fatal(err)
	}
	next := txn
	next.Revision = 1
	next.State = StateBuilding
	next.UpdatedAt = time.Now().UTC()

	preReplace := []atomicStep{
		stepBeforeTempCreate,
		stepAfterTempCreate,
		stepAfterWrite,
		stepAfterFileSync,
		stepAfterClose,
		stepAfterPreReplaceDirSync,
	}
	for _, step := range preReplace {
		t.Run("pre "+stepName(step), func(t *testing.T) {
			failing := mustStoreWithHook(t, root, failAt(step))
			if err := failing.SaveTransaction(next); err == nil {
				t.Fatal("injected failure succeeded")
			}
			got, err := os.ReadFile(mustTxnPath(t, store, txn.ID))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(old) {
				t.Fatal("old record was not preserved")
			}
			assertNoActivatedTemp(t, store.layout.TransactionsDir())
		})
	}

	postReplace := []atomicStep{stepAfterReplace, stepAfterFinalDirSync, stepAfterReadback}
	for _, step := range postReplace {
		t.Run("post "+stepName(step), func(t *testing.T) {
			freshRoot := t.TempDir()
			base := mustStoreAt(t, freshRoot)
			created := mustCreated(t)
			if err := base.SaveTransaction(created); err != nil {
				t.Fatal(err)
			}
			update := created
			update.Revision = 1
			update.State = StateBuilding
			update.UpdatedAt = time.Now().UTC()
			failing := mustStoreWithHook(t, freshRoot, failAt(step))
			if err := failing.SaveTransaction(update); err == nil {
				t.Fatal("injected failure succeeded")
			}
			loaded, err := base.LoadTransaction(created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.State != StateBuilding || loaded.Revision != 2 {
				t.Fatalf("complete new record not on disk after replace: %#v", loaded)
			}
		})
	}
}

func TestListClassifiesMalformedReservedName(t *testing.T) {
	store := mustStore(t)
	if err := store.layout.EnsureControl(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.layout.TransactionsDir(), "txn_bad.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	views, collision, err := store.ListTransactionViews()
	if err != nil {
		t.Fatal(err)
	}
	if collision {
		t.Fatal("unexpected collision")
	}
	if len(views) != 1 || !views[0].OccupiesReservedName || views[0].Valid {
		t.Fatalf("views %#v", views)
	}
}

func TestDirSyncFailure(t *testing.T) {
	root := t.TempDir()
	store := mustStoreAt(t, root)
	txn := mustCreated(t)
	if err := store.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(mustTxnPath(t, store, txn.ID))
	if err != nil {
		t.Fatal(err)
	}
	next := txn
	next.Revision = 1
	next.State = StateBuilding
	next.UpdatedAt = time.Now().UTC()

	t.Run("before replace", func(t *testing.T) {
		ops := defaultAtomicOps()
		ops.SyncDir = func(string) error { return errors.New("syncdir") }
		failing, err := newStoreWithOps(root, ops)
		if err != nil {
			t.Fatal(err)
		}
		if err := failing.SaveTransaction(next); err == nil {
			t.Fatal("dir-sync failure succeeded")
		}
		got, err := os.ReadFile(mustTxnPath(t, store, txn.ID))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(old) {
			t.Fatal("old record lost after pre-replace dir-sync failure")
		}
	})
	t.Run("after replace", func(t *testing.T) {
		fresh := t.TempDir()
		base := mustStoreAt(t, fresh)
		created := mustCreated(t)
		if err := base.SaveTransaction(created); err != nil {
			t.Fatal(err)
		}
		update := created
		update.Revision = 1
		update.State = StateBuilding
		update.UpdatedAt = time.Now().UTC()
		ops := defaultAtomicOps()
		calls := 0
		ops.SyncDir = func(dir string) error {
			calls++
			if calls == 1 {
				return syncDirectory(dir)
			}
			return errors.New("syncdir")
		}
		failing, err := newStoreWithOps(fresh, ops)
		if err != nil {
			t.Fatal(err)
		}
		if err := failing.SaveTransaction(update); err == nil {
			t.Fatal("post-replace dir-sync failure succeeded")
		}
		loaded, err := base.LoadTransaction(created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.State != StateBuilding {
			t.Fatalf("complete new record not recovered after post-replace dir-sync: %#v", loaded)
		}
	})
}

func TestFirstWriteFailpointLeavesNoRecord(t *testing.T) {
	root := t.TempDir()
	txn := mustCreated(t)
	failing := mustStoreWithHook(t, root, failAt(stepAfterWrite))
	if err := failing.SaveTransaction(txn); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := failing.LoadTransaction(txn.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dest appeared: %v", err)
	}
}

func TestSaveTransactionRejectsEscapingPaths(t *testing.T) {
	store := mustStore(t)
	txn := mustCreated(t)
	txn.GenerationRelativePath = `C:\Windows\` + txn.GenerationID
	if err := store.SaveTransaction(txn); err == nil {
		t.Fatal("absolute generation path accepted")
	}
	txn = mustCreated(t)
	txn.GenerationRelativePath = "generations/../staging/" + txn.ID
	if err := store.SaveTransaction(txn); err == nil {
		t.Fatal("escaping generation path accepted")
	}
}

func mustStore(t *testing.T) *Store {
	t.Helper()
	return mustStoreAt(t, t.TempDir())
}

func mustStoreAt(t *testing.T, root string) *Store {
	t.Helper()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func mustStoreWithHook(t *testing.T, root string, hook func(atomicStep) error) *Store {
	t.Helper()
	ops := defaultAtomicOps()
	ops.Hook = hook
	store, err := newStoreWithOps(root, ops)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func mustCreated(t *testing.T) InstallTransaction {
	t.Helper()
	txn, err := NewCreatedTransaction("hermes", "op_test", "df4b65147d7ddd74dd449f9067aabbca5aef0ec7", "0.20.2")
	if err != nil {
		t.Fatal(err)
	}
	return txn
}

func mustTxnPath(t *testing.T, store *Store, id string) string {
	t.Helper()
	path, err := store.layout.TransactionPath(id)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func failAt(step atomicStep) func(atomicStep) error {
	return func(got atomicStep) error {
		if got == step {
			return errors.New("injected")
		}
		return nil
	}
}

func assertNoActivatedTemp(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".yorva-atom-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Size() > 0 {
			payload, _ := os.ReadFile(filepath.Join(dir, entry.Name()))
			var probe map[string]any
			if json.Unmarshal(payload, &probe) == nil && probe["state"] != nil {
				t.Fatalf("complete temp left behind: %s", entry.Name())
			}
		}
	}
}

func stepName(step atomicStep) string {
	switch step {
	case stepBeforeTempCreate:
		return "beforeTempCreate"
	case stepAfterTempCreate:
		return "afterTempCreate"
	case stepAfterWrite:
		return "afterWrite"
	case stepAfterFileSync:
		return "afterFileSync"
	case stepAfterClose:
		return "afterClose"
	case stepAfterPreReplaceDirSync:
		return "afterPreReplaceDirSync"
	case stepAfterReplace:
		return "afterReplace"
	case stepAfterFinalDirSync:
		return "afterFinalDirSync"
	case stepAfterReadback:
		return "afterReadback"
	default:
		return "unknown"
	}
}
