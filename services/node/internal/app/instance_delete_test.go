package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestStartDeleteProtectsDefaultAndConfirmation(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var defaultID, coderID string
	for _, item := range listed.Instances {
		if item.Name == "default" {
			defaultID = item.InstanceID
		}
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	if _, err := inventory.StartDelete(context.Background(), defaultID, "default", "del-1"); !errors.Is(err, ErrInstanceProtected) {
		t.Fatalf("default delete = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("protected delete started a process")
	}
	if _, err := inventory.StartDelete(context.Background(), coderID, "wrong", "del-2"); !errors.Is(err, ErrInstanceConfirmationMismatch) {
		t.Fatalf("mismatch = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("mismatch started a process")
	}
}

func TestStartDeleteConvergesToMissingTombstone(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var coderID string
	for _, item := range listed.Instances {
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-ok")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), started.Operation.ID)
		if err == nil && got.Status == operation.StatusSucceeded {
			row, err := inventory.GetInstance(context.Background(), coderID)
			if err != nil || row.Availability != instance.Missing || row.InstanceID != coderID {
				t.Fatalf("tombstone = %#v %v", row, err)
			}
			return
		}
		if err == nil && got.Status == operation.StatusFailed {
			t.Fatalf("delete failed: %#v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("delete operation did not succeed")
}

func waitInstanceOperation(t *testing.T, inventory *InstanceInventory, id string, want operation.Status) operation.Operation {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var last operation.Operation
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), id)
		if err == nil {
			last = got
			if got.Status == want {
				return got
			}
			if operation.IsTerminal(got.Status) && got.Status != want {
				t.Fatalf("operation terminal %#v, want %s", got, want)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("operation did not reach %s: %#v", want, last)
	return last
}

func instanceIDByName(t *testing.T, listed InstanceList, name string) string {
	t.Helper()
	for _, item := range listed.Instances {
		if item.Name == name {
			return item.InstanceID
		}
	}
	t.Fatalf("instance %q not found", name)
	return ""
}

func TestStartDeleteQueryFailureBeforeMutationIsRejected(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	source.setErr(ErrInstanceQueryFailed)
	if _, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-query"); !errors.Is(err, ErrInstanceQueryFailed) {
		t.Fatalf("query failure start = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("failed query authorized Hermes delete")
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("uncertain row = %#v %v", row, err)
	}
	if _, ok, err := inventory.db.ActiveInstanceMutation(context.Background(), listed.RuntimeInstallationID); err != nil || ok {
		t.Fatalf("active mutation after rejected delete = ok=%v err=%v", ok, err)
	}
}

func TestStartDeleteWorkerQueryFailureBeforeMutationDoesNotSucceed(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	source.setErr(ErrInstanceOutputUnrecognized)
	source.failFromCall = 3
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-worker-query")
	if err != nil {
		t.Fatal(err)
	}
	got := waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusFailed)
	if got.ErrorCode != yorvaruntime.ErrorInstanceOutputUnrecognized {
		t.Fatalf("error code = %s", got.ErrorCode)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("worker deleted after query failure")
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("uncertain row = %#v %v", row, err)
	}
}

func TestStartDeleteQueryFailureAfterMutationDoesNotSucceed(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	source.failFromCall = 4
	source.setErr(ErrInstanceQueryFailed)
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-after-query")
	if err != nil {
		t.Fatal(err)
	}
	got := waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusFailed)
	if got.ErrorCode != yorvaruntime.ErrorInstanceQueryFailed {
		t.Fatalf("error code = %s", got.ErrorCode)
	}
	if calls, _ := mutator.snapshot(); calls != 1 {
		t.Fatalf("delete calls = %d", calls)
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("must not infer MISSING after post-delete query failure: %#v %v", row, err)
	}
}

func TestStartDeleteTimeoutIsStableErrorNotAbsence(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	source.setErr(context.DeadlineExceeded)
	if _, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-timeout"); !errors.Is(err, ErrInstanceOperationTimedOut) {
		t.Fatalf("timeout start = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("timeout authorized Hermes delete")
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("timeout row = %#v %v", row, err)
	}
}

func TestStartDeleteWorkerTimeoutAfterMutationDoesNotSucceed(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	coderID := instanceIDByName(t, listed, "coder")
	source.setErr(context.DeadlineExceeded)
	source.failFromCall = 4
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-timeout-after")
	if err != nil {
		t.Fatal(err)
	}
	got := waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusFailed)
	if got.ErrorCode != yorvaruntime.ErrorInstanceOperationTimedOut {
		t.Fatalf("error code = %s", got.ErrorCode)
	}
	if calls, _ := mutator.snapshot(); calls != 1 {
		t.Fatalf("delete calls = %d", calls)
	}
	row, err := inventory.GetInstance(context.Background(), coderID)
	if err != nil || row.Availability != instance.Unknown {
		t.Fatalf("timeout must not become MISSING: %#v %v", row, err)
	}
}

func TestStartDeleteDisappearanceRaceIsSuccess(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	var coderID string
	for _, item := range listed.Instances {
		if item.Name == "coder" {
			coderID = item.InstanceID
		}
	}
	source.removeProfile("coder")
	started, err := inventory.StartDelete(context.Background(), coderID, "coder", "del-race")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), started.Operation.ID)
		if err == nil && got.Status == operation.StatusSucceeded {
			_, _ = mutator.snapshot()
			row, err := inventory.GetInstance(context.Background(), coderID)
			if err != nil || row.Availability != instance.Missing {
				t.Fatalf("race tombstone = %#v %v", row, err)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("disappearance race did not converge")
}
