package install

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRecoverEmptyRootIsReady(t *testing.T) {
	root := t.TempDir()
	gate := NewGateHolder()
	decision, err := RecoverWith(context.Background(), root, gate, newMemEnv().store())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionNone && decision.Action != ActionDiagnoseRetain {
		t.Fatalf("%#v", decision)
	}
	if gate.Get() != GateReady {
		t.Fatalf("gate %s", gate.Get())
	}
}

func TestRecoverCreatedMovesStagingAndFails(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil).withEnv(newMemEnv().store())
	txn := mustCreated(t)
	if err := store.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	staging, err := store.layout.StagingPath(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "partial.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = mgr
	gate := NewGateHolder()
	if _, err := RecoverWith(context.Background(), store.layout.Root, gate, newMemEnv().store()); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateFailed {
		t.Fatalf("state %#v", loaded)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatal("staging remained")
	}
	if gate.Get() != GateReady {
		t.Fatalf("gate %s", gate.Get())
	}
}

func TestRecoverSealedForwardsWithoutTouchingUnknown(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustSeal(t, mgr)
	legacy := filepath.Join(store.layout.Root, "hermes-agent")
	if err := os.MkdirAll(filepath.Join(legacy, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "bin", "keep.exe"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate := NewGateHolder()
	if _, err := RecoverWith(context.Background(), store.layout.Root, gate, mem.store()); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateCommitted && loaded.State != StateActivating {
		t.Fatalf("state %#v", loaded)
	}
	if !store.ReadActive().Valid || store.ReadActive().GenerationID != txn.GenerationID {
		t.Fatal("did not activate")
	}
	if _, err := os.Stat(filepath.Join(legacy, "bin", "keep.exe")); err != nil {
		t.Fatal("legacy tree deleted")
	}
	if gate.Get() != GateReady && gate.Get() != GateReconciling {
		t.Fatalf("gate %s", gate.Get())
	}
}

func TestRecoverDuplicateSealedBlocksAndDoesNotPublishTwice(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil).withEnv(newMemEnv().store())
	txn := mustSeal(t, mgr)
	staging, err := store.layout.StagingPath(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(staging, dest); err != nil {
		t.Fatal(err)
	}
	gate := NewGateHolder()
	decision, err := RecoverWith(context.Background(), store.layout.Root, gate, newMemEnv().store())
	if err != nil && gate.Get() != GateBlockedUnsafe {
		t.Fatal(err)
	}
	if gate.Get() != GateBlockedUnsafe && decision.Gate != GateBlockedUnsafe {
		t.Fatalf("decision %#v gate %s", decision, gate.Get())
	}
	if !store.ReadActive().Valid {
		// must not have activated the duplicate world
	}
	if store.ReadActive().Valid {
		t.Fatal("duplicate world activated")
	}
	if _, err := os.Stat(staging); err != nil {
		t.Fatal("staging removed while blocked")
	}
}

func TestRecoverD2IsNoopOrForward(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	_ = mustSeal(t, mgr)
	gate := NewGateHolder()
	if _, err := RecoverWith(context.Background(), store.layout.Root, gate, mem.store()); err != nil {
		t.Fatal(err)
	}
	again, err := RecoverWith(context.Background(), store.layout.Root, gate, mem.store())
	if err != nil {
		t.Fatal(err)
	}
	if again.Action != ActionNone && again.Action != ActionDiagnoseRetain {
		t.Fatalf("second recover not stable: %#v", again)
	}
}

func TestObserveDoesNotDeleteUnknown(t *testing.T) {
	store := mustStore(t)
	mystery := filepath.Join(store.layout.Root, "mystery")
	if err := os.Mkdir(mystery, 0o700); err != nil {
		t.Fatal(err)
	}
	obs, err := Observe(store, newMemEnv().store())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(obs.UnknownDirectories, "mystery") {
		t.Fatalf("unknown %#v", obs.UnknownDirectories)
	}
	d := DecideRecovery(obs)
	if d.Action != ActionDiagnoseRetain {
		t.Fatalf("%#v", d)
	}
	if err := Execute(context.Background(), store, NewManager(store, nil, nil), obs, d); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mystery); err != nil {
		t.Fatal("unknown dir deleted")
	}
}

func copyDir(from, to string) error {
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(to, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, 0o700)
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return err
		}
		return os.WriteFile(dest, payload, 0o600)
	})
}
