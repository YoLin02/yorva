package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifySealedTreeRejectsPostSealMutation(t *testing.T) {
	store, txn := sealPublishedFixture(t)
	dest, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySealedTree(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPublishedGeneration(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		t.Fatal(err)
	}

	t.Run("modified file", func(t *testing.T) {
		target := filepath.Join(dest, "bin", "hermes.exe")
		if err := os.WriteFile(target, []byte("foreign-launcher"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := VerifySealedTree(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err == nil {
			t.Fatal("modified launcher still verified")
		}
		if err := VerifyPublishedGeneration(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
			t.Fatal("metadata-only verify should still pass")
		}
	})
}

func TestVerifySealedTreeRejectsInsertAndDelete(t *testing.T) {
	store, txn := sealPublishedFixture(t)
	dest, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}

	extra := filepath.Join(dest, "venv", "Scripts", "foreign.exe")
	if err := os.WriteFile(extra, []byte("mz-foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifySealedTree(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err == nil {
		t.Fatal("inserted executable still verified")
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dest, "bin", "hermes-acp.exe")); err != nil {
		t.Fatal(err)
	}
	if err := VerifySealedTree(dest, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err == nil {
		t.Fatal("deleted launcher still verified")
	}
}

func TestPublishRejectsMutatedSealedGeneration(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(store, func(_ context.Context, staging, _ string) error {
		return writeMinimalStagingTree(staging)
	}, nil)
	txn, err := NewCreatedTransaction("hermes", "op_mut", "pin", "0.20.2")
	if err != nil {
		t.Fatal(err)
	}
	txn, err = mgr.SealNew(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generation, "bin", "hermes.exe"), []byte("mutated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.PublishAndActivate(context.Background(), txn); err == nil {
		t.Fatal("publish accepted mutated sealed tree")
	}
	if store.ReadActive().Valid {
		t.Fatal("active.json written from mutated tree")
	}
}

func sealPublishedFixture(t *testing.T) (*Store, InstallTransaction) {
	t.Helper()
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(store, func(_ context.Context, staging, _ string) error {
		return writeMinimalStagingTree(staging)
	}, nil)
	txn, err := NewCreatedTransaction("hermes", "op_seal", "pin", "0.20.2")
	if err != nil {
		t.Fatal(err)
	}
	txn, err = mgr.SealNew(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	txn, err = mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	return store, txn
}
