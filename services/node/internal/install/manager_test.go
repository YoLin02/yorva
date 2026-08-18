package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var errInjected = errors.New("injected")

func TestManagerSealNew(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	txn := mustCreated(t)
	got, err := mgr.SealNew(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateSealed || got.ManifestSHA256 == "" || got.SealSHA256 == "" || got.SealedAt == nil {
		t.Fatalf("sealed %#v", got)
	}
	staging, err := store.layout.StagingPath(got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staging, fileGeneration)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.layout.Root, dirGenerations)); !os.IsNotExist(err) {
		t.Fatal("batch 3 must not publish generations")
	}
	loaded, err := store.LoadTransaction(got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != StateSealed {
		t.Fatalf("disk state %#v", loaded)
	}
}

func TestManagerRetryAllocatesNewIDs(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, nil)
	a, err := mgr.SealNew(context.Background(), mustCreated(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.SealNew(context.Background(), mustCreated(t))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.GenerationID == b.GenerationID || a.StagingRelativePath == b.StagingRelativePath {
		t.Fatalf("ids reused: %#v %#v", a, b)
	}
}

func TestManagerCancelLeavesUnsealed(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, func(context.Context, string, string) error {
		return context.Canceled
	}, nil)
	txn := mustCreated(t)
	got, err := mgr.SealNew(context.Background(), txn)
	if err == nil {
		t.Fatal("expected cancel")
	}
	if got.State != StateFailed || got.ErrorCode != CodeInterrupted {
		t.Fatalf("got %#v", got)
	}
	if _, err := os.Stat(filepath.Join(store.layout.Root, dirGenerations)); !os.IsNotExist(err) {
		t.Fatal("canceled build published a generation")
	}
	staging, _ := store.layout.StagingPath(got.ID)
	if _, err := os.Stat(filepath.Join(staging, fileGeneration)); err == nil {
		t.Fatal("canceled build sealed")
	}
}

func TestManagerFailpointsDoNotSeal(t *testing.T) {
	points := []string{
		FailBeforeStagingMkdir,
		FailAfterStagingMkdir,
		FailAfterBuild,
		FailBeforeValidate,
		FailDuringManifestWalk,
		FailAfterManifest,
		FailBeforeGenerationJSON,
		FailAfterSealBeforeSecondWalk,
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			store := mustStore(t)
			mgr := NewManager(store, fakeBuild, nil).withFailpoint(point)
			txn := mustCreated(t)
			got, err := mgr.SealNew(context.Background(), txn)
			if err == nil {
				t.Fatal("expected failpoint error")
			}
			if got.State == StateSealed {
				t.Fatal("failpoint reached SEALED")
			}
			if _, err := os.Stat(filepath.Join(store.layout.Root, dirGenerations)); !os.IsNotExist(err) {
				t.Fatal("failpoint published")
			}
			if point == FailBeforeGenerationJSON || point == FailAfterSealBeforeSecondWalk {
				staging, _ := store.layout.StagingPath(got.ID)
				if _, err := os.Stat(filepath.Join(staging, fileGeneration)); err == nil {
					t.Fatal("generation.json remained")
				}
			}
		})
	}
}

func TestManagerValidateFailureDoesNotSeal(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, fakeBuild, func(context.Context, string, string) error {
		return errors.New("missing launcher")
	})
	got, err := mgr.SealNew(context.Background(), mustCreated(t))
	if err == nil {
		t.Fatal("expected validate failure")
	}
	if got.State != StateFailed || got.ErrorCode != CodeSealInvalid {
		t.Fatalf("got %#v", got)
	}
}

func fakeBuild(_ context.Context, stagingAbs, hermesHome string) error {
	if stagingAbs == "" || hermesHome == "" {
		return errors.New("missing staging or home")
	}
	return writeMinimalStagingTree(stagingAbs)
}
