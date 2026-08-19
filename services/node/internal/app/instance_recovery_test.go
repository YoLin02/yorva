package app

import (
	"context"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func seedOrphanedInstanceOperation(t *testing.T, inventory *InstanceInventory, typ operation.Type, installationID, name, key string) operation.Operation {
	t.Helper()
	now := inventory.now()
	op := operation.Operation{
		ID:             "op_" + key,
		Type:           typ,
		TargetType:     operation.TargetRuntimeInstallation,
		TargetID:       installationID,
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		Message:        name,
		IdempotencyKey: key,
		CorrelationID:  "cor_" + key,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := inventory.db.CreateOperation(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	return op
}

func TestRecoverStaleCreateUsesHermesTruthNotOperationStatus(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	inventory.WithMutator(&fakeMutator{source: source})
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceCreate, listed.RuntimeInstallationID, "coder", "orphan-create-absent")
	first, err := inventory.RecoverStale(context.Background())
	if err != nil || len(first) != 1 || first[0].Status != operation.StatusFailed || first[0].ErrorCode != yorvaruntime.ErrorOperationInterrupted {
		t.Fatalf("absent create recover = %#v %v", first, err)
	}
	source.addProfile(ProfileSnapshot{NativeID: "coder"})
	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceCreate, listed.RuntimeInstallationID, "coder", "orphan-create-present")
	second, err := inventory.RecoverStale(context.Background())
	if err != nil || len(second) != 1 || second[0].Status != operation.StatusSucceeded {
		t.Fatalf("present create recover = %#v %v", second, err)
	}
	again, err := inventory.RecoverStale(context.Background())
	if err != nil || len(again) != 0 {
		t.Fatalf("repeated recover = %#v %v", again, err)
	}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "notes", "post-restart-create"); err != nil {
		t.Fatalf("post-restart create = %v", err)
	}
}

func TestRecoverStaleDeleteUsesHermesTruthNotOperationStatus(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	inventory.WithMutator(&fakeMutator{source: source})
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceDelete, listed.RuntimeInstallationID, "coder", "orphan-delete-present")
	first, err := inventory.RecoverStale(context.Background())
	if err != nil || len(first) != 1 || first[0].Status != operation.StatusFailed || first[0].ErrorCode != yorvaruntime.ErrorOperationInterrupted {
		t.Fatalf("present delete recover = %#v %v", first, err)
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Available {
		t.Fatalf("present delete must not invent absence: %#v %v", row, err)
	}
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "post-restart-delete")
	if err != nil {
		t.Fatalf("post-restart delete = %v", err)
	}
	waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusSucceeded)

	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceDelete, listed.RuntimeInstallationID, "coder", "orphan-delete-absent")
	second, err := inventory.RecoverStale(context.Background())
	if err != nil || len(second) != 1 || second[0].Status != operation.StatusSucceeded {
		t.Fatalf("absent delete recover = %#v %v", second, err)
	}
	tombstone, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || tombstone.Availability != instance.Missing {
		t.Fatalf("recovered absence tombstone = %#v %v", tombstone, err)
	}
}

func TestRecoverStaleQueryFailureDoesNotInferAbsenceAndUnblocksMutation(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	inventory.WithMutator(&fakeMutator{source: source})
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceDelete, listed.RuntimeInstallationID, "coder", "orphan-delete-unknown")
	source.setErr(ErrInstanceQueryFailed)
	first, err := inventory.RecoverStale(context.Background())
	if err != nil || len(first) != 1 || first[0].Status != operation.StatusFailed || first[0].ErrorCode != yorvaruntime.ErrorInstanceQueryFailed {
		t.Fatalf("unknown delete recover = %#v %v", first, err)
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("query failure must stay UNKNOWN: %#v %v", row, err)
	}
	again, err := inventory.RecoverStale(context.Background())
	if err != nil || len(again) != 0 {
		t.Fatalf("repeated recover after query failure = %#v %v", again, err)
	}
	source.setErr(nil)
	if _, err := inventory.StartDelete(context.Background(), coderID, "coder", "after-unknown-recover"); err != nil {
		t.Fatalf("mutation after recovered query failure = %v", err)
	}
}

func TestRecoverStaleTimeoutPreservesUnknown(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceCreate, listed.RuntimeInstallationID, "notes", "orphan-create-timeout")
	source.setErr(context.DeadlineExceeded)
	got, err := inventory.RecoverStale(context.Background())
	if err != nil || len(got) != 1 || got[0].Status != operation.StatusFailed || got[0].ErrorCode != yorvaruntime.ErrorInstanceOperationTimedOut {
		t.Fatalf("timeout recover = %#v %v", got, err)
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("timeout recover row = %#v %v", row, err)
	}
}

func TestRecoverStaleRunningDeleteIsIdempotent(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	inventory.WithMutator(&fakeMutator{source: source})
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	op := seedOrphanedInstanceOperation(t, inventory, operation.TypeInstanceDelete, listed.RuntimeInstallationID, "coder", "orphan-running")
	now := inventory.now()
	running := op
	running.Status = operation.StatusRunning
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := inventory.db.UpdateOperation(context.Background(), op, running); err != nil {
		t.Fatal(err)
	}
	first, err := inventory.RecoverStale(context.Background())
	if err != nil || len(first) != 1 || first[0].Status != operation.StatusFailed {
		t.Fatalf("running recover = %#v %v", first, err)
	}
	second, err := inventory.RecoverStale(context.Background())
	if err != nil || len(second) != 0 {
		t.Fatalf("second recover = %#v %v", second, err)
	}
	if _, ok, err := inventory.db.ActiveInstanceMutation(context.Background(), listed.RuntimeInstallationID); err != nil || ok {
		t.Fatalf("active after recover = ok=%v err=%v", ok, err)
	}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "notes", "after-running-recover"); err != nil {
		t.Fatalf("post-restart create = %v", err)
	}
}
