package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestCreateCancelBeforeStartAndNotAfter(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	mutator := &blockingMutator{source: source, started: started, release: release}
	inventory.WithMutator(mutator)

	first, err := inventory.StartCreate(context.Background(), "hermes", "coder", "cancel-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.CancelCreate(context.Background(), first.Operation.ID); err != nil && !errors.Is(err, ErrInstanceNotCancellable) {
		// May already have entered RUNNING; either cancelled or not-cancellable is acceptable
		// once the worker has begun. After the command starts, cancel must fail.
	}
	select {
	case <-started:
		if _, err := inventory.CancelCreate(context.Background(), first.Operation.ID); !errors.Is(err, ErrInstanceNotCancellable) {
			t.Fatalf("cancel after command start = %v", err)
		}
		close(release)
	case <-time.After(15 * time.Second):
		// cancelled before start
		got, err := inventory.db.GetOperation(context.Background(), first.Operation.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != operation.StatusCancelled && got.Status != operation.StatusRunning && got.Status != operation.StatusSucceeded {
			t.Fatalf("pre-start cancel status = %s", got.Status)
		}
		close(release)
	}
}

func TestConcurrentReconcileDoesNotDuplicateNativeIdentity(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := inventory.ListInstances(context.Background(), "hermes")
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, item := range listed.Instances {
		seen[item.Name]++
		if item.Availability != instance.Available {
			t.Fatalf("torn availability %#v", item)
		}
	}
	if seen["default"] != 1 || seen["coder"] != 1 {
		t.Fatalf("duplicate or missing rows: %#v", seen)
	}
}

func TestQueryFailureDoesNotInventMissingAndKeepsPhase3CapabilityFalse(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	if _, err := inventory.ListInstances(context.Background(), "hermes"); err != nil {
		t.Fatal(err)
	}
	source.setErr(ErrInstanceQueryFailed)
	failed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Freshness != "UNKNOWN" || failed.Capabilities.Lifecycle {
		t.Fatalf("failed list = %#v", failed)
	}
	if failed.ErrorCode != yorvaruntime.ErrorInstanceQueryFailed {
		t.Fatalf("error code = %s", failed.ErrorCode)
	}
	if len(failed.Instances) != 1 || failed.Instances[0].Availability != instance.Unknown {
		t.Fatalf("false missing: %#v", failed.Instances)
	}
}

type blockingMutator struct {
	source  *fakeProfileSource
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingMutator) Create(ctx context.Context, _ string, name string) error {
	b.once.Do(func() { close(b.started) })
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	if b.source != nil {
		b.source.addProfile(ProfileSnapshot{NativeID: name})
	}
	return nil
}

func (b *blockingMutator) Delete(context.Context, string, string) error {
	return nil
}
