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
	generation, err := store.layout.GenerationPath(got.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(generation, fileGeneration)); err != nil {
		t.Fatal(err)
	}
	if candidate, ok := readCandidateRecord(generation); !ok || candidate.TransactionID != got.ID || candidate.GenerationID != got.GenerationID {
		t.Fatalf("candidate record %#v valid=%v", candidate, ok)
	}
	staging, err := store.layout.StagingPath(got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatal("new install built a staging runtime tree")
	}
	if store.ReadActive().Valid {
		t.Fatal("sealed candidate must not be active")
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
	generation, _ := store.layout.GenerationPath(got.GenerationID)
	if _, err := os.Stat(filepath.Join(generation, fileGeneration)); err == nil {
		t.Fatal("canceled build sealed")
	}
	if store.ReadActive().Valid {
		t.Fatal("canceled candidate activated")
	}
}

func TestManagerFailpointsDoNotSeal(t *testing.T) {
	points := []string{
		FailBeforeGenerationMkdir,
		FailAfterGenerationMkdir,
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
			if store.ReadActive().Valid {
				t.Fatal("failpoint activated a generation")
			}
			if point == FailBeforeGenerationJSON || point == FailAfterSealBeforeSecondWalk {
				generation, _ := store.layout.GenerationPath(got.GenerationID)
				if _, err := os.Stat(filepath.Join(generation, fileGeneration)); err == nil {
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

func TestManagerRequiresCandidateOwnershipRecordAtSeal(t *testing.T) {
	store := mustStore(t)
	mgr := NewManager(store, func(_ context.Context, generationAbs, _ string) error {
		if err := writeMinimalStagingTree(generationAbs); err != nil {
			return err
		}
		return os.Remove(filepath.Join(generationAbs, fileCandidate))
	}, nil)
	got, err := mgr.SealNew(context.Background(), mustCreated(t))
	if err == nil {
		t.Fatal("missing candidate ownership record accepted")
	}
	if got.State != StateFailed || got.ErrorCode != CodeSealInvalid {
		t.Fatalf("got %#v", got)
	}
}

func fakeBuild(_ context.Context, generationAbs, hermesHome string) error {
	if generationAbs == "" || hermesHome == "" {
		return errors.New("missing generation or home")
	}
	return writeMinimalStagingTree(generationAbs)
}
