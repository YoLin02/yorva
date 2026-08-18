package app

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/install"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestStartCreatesTransactionBeforeOperation(t *testing.T) {
	root := t.TempDir()
	store := newMemoryOperationStore()
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
	}, &fakeApplier{dir: `C:\h\bin\hermes.exe`, version: "0.20.2"}, &fakeCompleter{store: store}).WithManagedRoot(root)

	started, err := service.Start(context.Background(), "hermes", "install-txn-first")
	if err != nil {
		t.Fatal(err)
	}
	if started.Operation.TransactionID == "" {
		t.Fatal("operation missing transaction projection")
	}
	if started.Operation.OwnershipNonce != "" {
		t.Fatal("retry ownership nonce must not be allocated")
	}
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	txn, err := istore.LoadTransaction(started.Operation.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	if txn.OperationID != started.Operation.ID {
		t.Fatalf("txn op id %s want %s", txn.OperationID, started.Operation.ID)
	}
}

func TestRetryAllocatesNewTransactionAndGeneration(t *testing.T) {
	root := t.TempDir()
	store := newMemoryOperationStore()
	applier := &fakeApplier{dir: `C:\h\bin\hermes.exe`, version: "0.20.2", applyErr: errors.New("boom")}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
	}, applier, &fakeCompleter{store: store}).WithManagedRoot(root)

	first, err := service.Start(context.Background(), "hermes", "retry-a")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, service, first.Operation.ID, operation.StatusFailed)
	second, err := service.Start(context.Background(), "hermes", "retry-b")
	if err != nil {
		t.Fatal(err)
	}
	if first.Operation.TransactionID == "" || first.Operation.TransactionID == second.Operation.TransactionID {
		t.Fatalf("retry reused transaction %q %q", first.Operation.TransactionID, second.Operation.TransactionID)
	}
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a, err := istore.LoadTransaction(first.Operation.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := istore.LoadTransaction(second.Operation.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	if a.GenerationID == b.GenerationID || a.StagingRelativePath == b.StagingRelativePath {
		t.Fatalf("retry reused generation/staging %#v %#v", a, b)
	}
}

func TestFailAfterCreatedBeforeOperationLeavesFailedTxnAndNoStaging(t *testing.T) {
	root := t.TempDir()
	store := newMemoryOperationStore()
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
	}, &fakeApplier{dir: `C:\h\bin\hermes.exe`, version: "0.20.2"}, &fakeCompleter{store: store}).WithManagedRoot(root)
	service.failAfterTxnCreate = true
	_, err := service.Start(context.Background(), "hermes", "fail-before-op")
	if err == nil {
		t.Fatal("expected injected failure")
	}
	if len(store.byID) != 0 {
		t.Fatalf("operation inserted: %#v", store.byID)
	}
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	views, _, err := istore.ListTransactionViews()
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].State != install.StateFailed {
		t.Fatalf("views %#v", views)
	}
	staging, err := istore.Layout().StagingPath(views[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatal("staging created before operation insert")
	}
}

func TestGetProjectsCommittedTransactionAsSucceeded(t *testing.T) {
	root := t.TempDir()
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	txn, err := install.NewCreatedTransaction("hermes", "op_proj", "pin", "0.20.2")
	if err != nil {
		t.Fatal(err)
	}
	if err := istore.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	loaded, err := istore.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.State = install.StateCommitted
	if err := istore.SaveTransaction(loaded); err != nil {
		t.Fatal(err)
	}
	store := newMemoryOperationStore()
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	op := operation.Operation{
		ID:             "op_proj",
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusRunning,
		Stage:          operation.StageInstallPath,
		IdempotencyKey: "proj",
		TransactionID:  txn.ID,
		CreatedAt:      now,
		StartedAt:      &now,
		UpdatedAt:      now,
	}
	if err := store.CreateOperation(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeInstall(nil, store).WithManagedRoot(root)
	got, err := service.Get(context.Background(), op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != operation.StatusSucceeded || got.Stage != operation.StageCleanup {
		t.Fatalf("projected %#v", got)
	}
}

func TestStageProjectionFailureDoesNotStopApply(t *testing.T) {
	store := newMemoryOperationStore()
	applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent\bin\hermes.exe`, version: "0.20.2"}
	completer := &fakeCompleter{store: store}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
		{
			State:    yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: applier.dir, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		},
	}, applier, completer)
	service.failStageProjection = true
	started, err := service.Start(context.Background(), "hermes", "stale-ui")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, service, started.Operation.ID, operation.StatusSucceeded)
}

func TestInterruptStaleProjectsCommittedInsteadOfFailing(t *testing.T) {
	root := t.TempDir()
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	txn, err := install.NewCreatedTransaction("hermes", "op_int", "pin", "0.20.2")
	if err != nil {
		t.Fatal(err)
	}
	if err := istore.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	loaded, err := istore.LoadTransaction(txn.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.State = install.StateCommitted
	if err := istore.SaveTransaction(loaded); err != nil {
		t.Fatal(err)
	}
	store := newMemoryOperationStore()
	now := time.Now().UTC()
	op := operation.Operation{
		ID:             "op_int",
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusRunning,
		IdempotencyKey: "int",
		TransactionID:  txn.ID,
		CreatedAt:      now,
		StartedAt:      &now,
		UpdatedAt:      now,
	}
	if err := store.CreateOperation(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeInstall(nil, store).WithManagedRoot(root)
	interrupted, err := service.InterruptStale(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(interrupted) != 0 {
		t.Fatalf("committed txn was interrupted: %#v", interrupted)
	}
	got, err := store.GetOperation(context.Background(), op.ID)
	if err != nil || got.Status != operation.StatusSucceeded {
		t.Fatalf("projected %#v %v", got, err)
	}
}

func TestFailAfterCommittedLeavesOperationRunningUntilGet(t *testing.T) {
	root := t.TempDir()
	store := newMemoryOperationStore()
	applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent\bin\hermes.exe`, version: "0.20.2"}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
	}, applier, &fakeCompleter{store: store}).WithManagedRoot(root)
	service.failAfterCommittedOp = true
	started, err := service.Start(context.Background(), "hermes", "after-commit")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := store.GetOperation(context.Background(), started.Operation.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == operation.StatusRunning && service.requestCancel(started.Operation.ID) == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("worker did not stop in RUNNING: %#v", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Apply on fake does not COMMIT the txn; simulate COMMITTED on disk.
	istore, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	txn, err := istore.LoadTransaction(started.Operation.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	txn.State = install.StateCommitted
	if err := istore.SaveTransaction(txn); err != nil {
		t.Fatal(err)
	}
	got, err := service.Get(context.Background(), started.Operation.ID)
	if err != nil || got.Status != operation.StatusSucceeded {
		t.Fatalf("GET projection %#v %v", got, err)
	}
}

func TestExecuteDoesNotConsultPreviousInstallOwner(t *testing.T) {
	store := newMemoryOperationStore()
	previous := operation.Operation{
		ID:             "op_old",
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusFailed,
		Retryable:      true,
		OwnershipNonce: "own_should_not_be_used",
		SourcePin:      "df4b65147d7ddd74dd449f9067aabbca5aef0ec7",
		IdempotencyKey: "old",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := store.CreateOperation(context.Background(), previous); err != nil {
		t.Fatal(err)
	}
	seenPrevious := false
	applier := &recordingApplier{fakeApplier: fakeApplier{dir: `C:\h\bin\hermes.exe`, version: "0.20.2"}}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
		{
			State:    yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: `C:\h\bin\hermes.exe`, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		},
	}, applier, &fakeCompleter{store: store})
	started, err := service.Start(context.Background(), "hermes", "no-owner")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, service, started.Operation.ID, operation.StatusSucceeded)
	if applier.retry {
		t.Fatal("ValidateTarget used previous-install retry ownership")
	}
	_ = seenPrevious
}

type recordingApplier struct {
	fakeApplier
	retry bool
}

func (r *recordingApplier) ValidateTarget(retry bool, previous operation.Operation) error {
	r.retry = retry
	if previous.OwnershipNonce != "" {
		r.retry = true
	}
	return nil
}
