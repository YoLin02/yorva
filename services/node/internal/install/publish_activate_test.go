package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishAndActivateWritesPointerAndLeavesPredecessor(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	first := mustSeal(t, mgr)
	first, err := mgr.PublishAndActivate(context.Background(), first)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != StateActivating {
		t.Fatalf("first state %#v", first)
	}
	firstDir, err := store.layout.GenerationPath(first.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(firstDir, "keep.txt")
	if err := os.WriteFile(keep, []byte("pred"), 0o600); err != nil {
		t.Fatal(err)
	}

	second := mustSeal(t, mgr)
	second, err = mgr.PublishAndActivate(context.Background(), second)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != StateActivating || second.ActiveBeforeGeneration != first.GenerationID {
		t.Fatalf("second %#v", second)
	}
	got := store.ReadActive()
	if !got.Valid || got.GenerationID != second.GenerationID {
		t.Fatalf("active %#v", got)
	}
	if body, err := os.ReadFile(keep); err != nil || string(body) != "pred" {
		t.Fatalf("predecessor moved or mutated: %q %v", body, err)
	}
	if _, err := os.Stat(mustStaging(t, store, first.ID)); !os.IsNotExist(err) {
		t.Fatal("first staging still present")
	}
	if _, err := os.Stat(mustStaging(t, store, second.ID)); !os.IsNotExist(err) {
		t.Fatal("second staging still present")
	}
	if _, err := os.Stat(filepath.Join(firstDir, fileGeneration)); err != nil {
		t.Fatal(err)
	}
}

func TestPublishAndActivateRollsForwardIfAlreadyNamed(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustSeal(t, mgr)
	txn, err := mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
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

func TestPublishFailpointsLeavePredecessorPointer(t *testing.T) {
	points := []string{
		FailBeforePublishVerify,
		FailAfterPublishVerify,
		FailAfterPublished,
		FailBeforeActivatingPersist,
		FailAfterActivatingPersist,
		FailDuringActiveWrite,
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			store := mustStore(t)
			mgr := NewManager(store, fakeBuild, nil)
			pred := mustSeal(t, mgr)
			pred, err := mgr.PublishAndActivate(context.Background(), pred)
			if err != nil {
				t.Fatal(err)
			}
			txn := mustSeal(t, mgr)
			got, err := mgr.withFailpoint(point).PublishAndActivate(context.Background(), txn)
			if err == nil {
				t.Fatal("expected failpoint")
			}
			if got.State == StateCommitted {
				t.Fatal("batch 4 must not commit")
			}
			active := store.ReadActive()
			if !active.Valid || active.GenerationID != pred.GenerationID {
				t.Fatalf("predecessor pointer lost: %#v", active)
			}
			predDir, _ := store.layout.GenerationPath(pred.GenerationID)
			if _, err := os.Stat(filepath.Join(predDir, fileGeneration)); err != nil {
				t.Fatal("predecessor generation missing")
			}
		})
	}
}

func TestPublishResumesAfterVerifyBeforePersist(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustSeal(t, mgr)
	_, err := mgr.withFailpoint(FailAfterPublishVerify).PublishAndActivate(context.Background(), txn)
	if err == nil {
		t.Fatal("expected failpoint")
	}
	loaded, err := store.LoadTransaction(txn.ID)
	if err != nil || loaded.State != StateSealed {
		t.Fatalf("disk %#v %v", loaded, err)
	}
	dest, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPublishedGeneration(dest, txn.GenerationID, loaded.ManifestSHA256, loaded.SealSHA256); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mustStaging(t, store, txn.ID)); !os.IsNotExist(err) {
		t.Fatal("new flow unexpectedly created staging")
	}
	got, err := mgr.PublishAndActivate(context.Background(), loaded)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateActivating || store.ReadActive().GenerationID != txn.GenerationID {
		t.Fatalf("resume %#v active %#v", got, store.ReadActive())
	}
}

func TestPublishDestExistsDifferentBytesFailClosed(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustSeal(t, mgr)
	dest, err := store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "foreign.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := mgr.PublishAndActivate(context.Background(), txn)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if got.State != StateFailed || got.ErrorCode != CodePublishConflict {
		t.Fatalf("got %#v", got)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal("candidate evidence must remain")
	}
	if store.ReadActive().Valid {
		t.Fatal("activated on conflict")
	}
}

func TestActivateUnrelatedPointerBlocks(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	a := mustPublish(t, mgr)
	b := mustPublish(t, mgr)
	c := mustSeal(t, mgr)
	_, err := mgr.withFailpoint(FailAfterActivatingPersist).PublishAndActivate(context.Background(), c)
	if err == nil {
		t.Fatal("expected failpoint")
	}
	cDisk, err := store.LoadTransaction(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	cDisk.ActiveBeforeGeneration = a.GenerationID
	if err := store.SaveTransaction(cDisk); err != nil {
		t.Fatal(err)
	}
	cDisk, err = store.LoadTransaction(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if store.ReadActive().GenerationID != b.GenerationID {
		t.Fatal("expected B to remain active")
	}
	got, err := mgr.PublishAndActivate(context.Background(), cDisk)
	if !errors.Is(err, ErrBlockedUnsafe) {
		t.Fatalf("err = %v", err)
	}
	if store.ReadActive().GenerationID != b.GenerationID {
		t.Fatal("unrelated activate overwrote pointer")
	}
	if got.State != StateActivating {
		t.Fatalf("state %#v", got)
	}
}

func TestPublishDoesNotWritePATH(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustPublish(t, mgr)
	if txn.State != StateActivating {
		t.Fatalf("%#v", txn)
	}
	if _, err := os.Stat(filepath.Join(store.layout.Root, "hermes-agent")); !os.IsNotExist(err) {
		t.Fatal("live hermes-agent created")
	}
}

func mustSeal(t *testing.T, mgr *Manager) InstallTransaction {
	t.Helper()
	txn, err := mgr.SealNew(context.Background(), mustCreated(t))
	if err != nil {
		t.Fatal(err)
	}
	return txn
}

func mustPublish(t *testing.T, mgr *Manager) InstallTransaction {
	t.Helper()
	txn := mustSeal(t, mgr)
	got, err := mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func mustStaging(t *testing.T, store *Store, id string) string {
	t.Helper()
	path, err := store.layout.StagingPath(id)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
