package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestActivateMissingPointerFirstInstall(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustSeal(t, mgr)
	txn, err := mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if txn.ActiveBeforeKind != ActiveBeforeAbsent {
		t.Fatalf("first install snapshot %#v", txn)
	}
	got := store.ReadActive()
	if !got.IsValid() || got.GenerationID != txn.GenerationID {
		t.Fatalf("active %#v", got)
	}
}

func TestActivateInvalidPointerBlocksAndPreservesBytes(t *testing.T) {
	store := mustStore(t)
	if err := store.layout.EnsureControl(); err != nil {
		t.Fatal(err)
	}
	path := store.layout.ActivePath()
	original := []byte(`{"not":"valid"`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustSeal(t, mgr)
	if _, err := mgr.PublishAndActivate(context.Background(), txn); !errors.Is(err, ErrBlockedUnsafe) {
		t.Fatalf("err = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("invalid pointer rewritten: %q", got)
	}
	if !store.ReadActive().Invalid() {
		t.Fatal("invalid pointer not classified")
	}
}

func TestActivateExactPredecessorSucceeds(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	first := mustPublish(t, mgr)
	second := mustSeal(t, mgr)
	second, err := mgr.PublishAndActivate(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if second.ActiveBeforeKind != ActiveBeforeValid || second.ActiveBeforeGeneration != first.GenerationID {
		t.Fatalf("predecessor snapshot %#v", second)
	}
	if second.ActiveBeforeDigest == "" {
		t.Fatal("predecessor digest missing")
	}
	if store.ReadActive().GenerationID != second.GenerationID {
		t.Fatal("new generation not active")
	}
	predDir, err := store.layout.GenerationPath(first.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(predDir, fileGeneration)); err != nil {
		t.Fatal("old generation destroyed")
	}
}

func TestActivatePredecessorIDMismatchBlocks(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	first := mustPublish(t, mgr)
	second := mustSeal(t, mgr)
	second, err := mgr.publishReady(second)
	if err != nil {
		t.Fatal(err)
	}
	second.State = StateActivating
	second.ActiveBeforeKind = ActiveBeforeValid
	second.ActiveBeforeGeneration = "gen_cccccccccccccccccccccc"
	second.ActiveBeforeDigest = first.SealSHA256
	if err := mgr.persist(&second); err != nil {
		t.Fatal(err)
	}
	if err := mgr.activate(&second); !errors.Is(err, ErrBlockedUnsafe) {
		t.Fatalf("err = %v", err)
	}
	if store.ReadActive().GenerationID != first.GenerationID {
		t.Fatal("predecessor pointer replaced")
	}
}

func TestActivatePredecessorDigestMismatchBlocks(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	first := mustPublish(t, mgr)
	second := mustSeal(t, mgr)
	second, err := mgr.publishReady(second)
	if err != nil {
		t.Fatal(err)
	}
	second.State = StateActivating
	second.ActiveBeforeKind = ActiveBeforeValid
	second.ActiveBeforeGeneration = first.GenerationID
	second.ActiveBeforeDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := mgr.persist(&second); err != nil {
		t.Fatal(err)
	}
	if err := mgr.activate(&second); !errors.Is(err, ErrBlockedUnsafe) {
		t.Fatalf("err = %v", err)
	}
	if store.ReadActive().GenerationID != first.GenerationID {
		t.Fatal("predecessor pointer replaced")
	}
}

func TestActivateAlreadyNewGenerationRollsForward(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustPublish(t, mgr)
	again, err := mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if again.State != StateActivating {
		t.Fatalf("state %#v", again)
	}
	if store.ReadActive().GenerationID != txn.GenerationID {
		t.Fatal("pointer changed on roll-forward")
	}
}

func TestActivateCrashWindowsAndIdempotentRecover(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustSeal(t, mgr)
	if _, err := mgr.withFailpoint(FailDuringActiveWrite).PublishAndActivate(context.Background(), txn); err == nil {
		t.Fatal("expected crash before write")
	}
	if store.ReadActive().IsValid() {
		t.Fatal("pointer written before failpoint")
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.PublishAndActivate(context.Background(), loaded); err != nil {
		t.Fatal(err)
	}
	if !store.ReadActive().IsValid() {
		t.Fatal("resume did not write pointer")
	}

	gate := NewGateHolder()
	first, err := RecoverWith(context.Background(), store.layout.Root, gate, mem.store())
	if err != nil {
		t.Fatal(err)
	}
	second, err := RecoverWith(context.Background(), store.layout.Root, gate, mem.store())
	if err != nil {
		t.Fatal(err)
	}
	if first.Action != second.Action || first.Gate != second.Gate || first.NextState != second.NextState {
		t.Fatalf("recover not idempotent %#v %#v", first, second)
	}
}

func TestRecoverInvalidPointerBlocksWithoutRewrite(t *testing.T) {
	store := mustStore(t)
	if err := store.layout.EnsureControl(); err != nil {
		t.Fatal(err)
	}
	path := store.layout.ActivePath()
	original := []byte("{")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	gate := NewGateHolder()
	first, err := RecoverWith(context.Background(), store.layout.Root, gate, newMemEnv().store())
	if err != nil && gate.Get() != GateBlockedUnsafe {
		t.Fatal(err)
	}
	if gate.Get() != GateBlockedUnsafe || first.Action != ActionBlockUnsafe {
		t.Fatalf("decision %#v gate %s", first, gate.Get())
	}
	second, err := RecoverWith(context.Background(), store.layout.Root, gate, newMemEnv().store())
	if err != nil && gate.Get() != GateBlockedUnsafe {
		t.Fatal(err)
	}
	if second.Gate != first.Gate || second.Action != first.Action {
		t.Fatalf("second recover %#v", second)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("recovery rewrote invalid pointer: %q", got)
	}
}

func (m *Manager) publishReady(txn InstallTransaction) (InstallTransaction, error) {
	loaded, err := m.store.LoadTransaction(txn.ID)
	if err != nil {
		return txn, err
	}
	if err := m.publish(&loaded); err != nil {
		return loaded, err
	}
	return loaded, nil
}
