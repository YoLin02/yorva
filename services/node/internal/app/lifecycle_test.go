package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeLifecycleManager struct {
	mu    sync.Mutex
	state yorvaruntime.LifecycleState
	err   error
}

func (f *fakeLifecycleManager) Status(context.Context, yorvaruntime.LifecycleInstallation, string) (yorvaruntime.LifecycleStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return yorvaruntime.LifecycleStatus{State: f.state}, f.err
}

func (f *fakeLifecycleManager) Start(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.state = yorvaruntime.LifecycleRunning
	return nil
}

func (f *fakeLifecycleManager) Stop(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.state = yorvaruntime.LifecycleStopped
	return nil
}

func (f *fakeLifecycleManager) Restart(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state != yorvaruntime.LifecycleRunning {
		return yorvaruntime.ErrInstanceNotRunning
	}
	return f.err
}

func TestLifecycleCapabilityStatusAndStartOperation(t *testing.T) {
	manager := &fakeLifecycleManager{state: yorvaruntime.LifecycleStopped}
	inventory, _ := newTestInventoryWithLifecycle(t, []ProfileSnapshot{{NativeID: "default", Default: true}, {NativeID: "coder"}}, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil || !listed.Capabilities.Lifecycle {
		t.Fatalf("list = %#v %v", listed, err)
	}
	var coderID string
	for _, item := range listed.Instances {
		if item.Name == "coder" {
			coderID = item.InstanceID
			if !item.Capabilities.Lifecycle {
				t.Fatal("row lifecycle capability false")
			}
		}
	}
	view, err := inventory.GetLifecycle(context.Background(), coderID)
	if err != nil || view.State != yorvaruntime.LifecycleStopped || view.ActiveOperationID != nil {
		t.Fatalf("status = %#v %v", view, err)
	}
	started, err := inventory.StartLifecycle(context.Background(), coderID, LifecycleStart, "life-start-1")
	if err != nil || started.Operation.Type != operation.TypeInstanceStart || started.Operation.TargetID != coderID {
		t.Fatalf("start = %#v %v", started, err)
	}
	done := waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusSucceeded)
	if done.ErrorCode != "" {
		t.Fatalf("done = %#v", done)
	}
	view, err = inventory.GetLifecycle(context.Background(), coderID)
	if err != nil || view.State != yorvaruntime.LifecycleRunning {
		t.Fatalf("running = %#v %v", view, err)
	}
}

func TestLifecycleRestartStoppedFailsWithStableCode(t *testing.T) {
	manager := &fakeLifecycleManager{state: yorvaruntime.LifecycleStopped}
	inventory, _ := newTestInventoryWithLifecycle(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	started, err := inventory.StartLifecycle(context.Background(), listed.Instances[0].InstanceID, LifecycleRestart, "life-restart-stopped")
	if err != nil {
		t.Fatal(err)
	}
	done := waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusFailed)
	if done.ErrorCode != yorvaruntime.ErrorInstanceNotRunning || done.Retryable {
		t.Fatalf("done = %#v", done)
	}
}

func TestRecoverRestartNeverInfersSuccessFromRunningState(t *testing.T) {
	manager := &fakeLifecycleManager{state: yorvaruntime.LifecycleRunning}
	inventory, _ := newTestInventoryWithLifecycle(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 20, 0, 0, 0, time.UTC)
	op := operation.Operation{ID: "op_lifecycle_restart_stale", Type: operation.TypeInstanceRestart, TargetType: operation.TargetInstance, TargetID: listed.Instances[0].InstanceID, Status: operation.StatusRunning, Stage: operation.StageInstanceRestart, IdempotencyKey: "stale-restart", CreatedAt: now, StartedAt: &now, UpdatedAt: now}
	if err := inventory.db.CreateOperation(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	recovered, err := inventory.RecoverLifecycle(context.Background())
	if err != nil || len(recovered) != 1 || recovered[0].Status != operation.StatusFailed || recovered[0].ErrorCode != yorvaruntime.ErrorLifecycleResultUnknown {
		t.Fatalf("recovered = %#v %v", recovered, err)
	}
}

func TestLifecycleStatusErrorProjectsUnknownWithoutRawError(t *testing.T) {
	manager := &fakeLifecycleManager{state: yorvaruntime.LifecycleUnknown, err: yorvaruntime.ErrLifecycleOutputUnrecognized}
	inventory, _ := newTestInventoryWithLifecycle(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	view, err := inventory.GetLifecycle(context.Background(), listed.Instances[0].InstanceID)
	if err != nil || view.State != yorvaruntime.LifecycleUnknown || view.ErrorCode != yorvaruntime.ErrorLifecycleOutputUnrecognized {
		t.Fatalf("view = %#v %v", view, err)
	}
	if errors.Is(err, manager.err) {
		t.Fatal("raw adapter error escaped")
	}
}
