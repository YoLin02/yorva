package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

type fakeMutator struct {
	mu      sync.Mutex
	source  *fakeProfileSource
	err     error
	calls   int
	last    string
	created []string
}

func (f *fakeMutator) Delete(_ context.Context, _ string, nativeID string) error {
	f.mu.Lock()
	f.calls++
	f.last = nativeID
	err := f.err
	source := f.source
	f.mu.Unlock()
	if err != nil {
		return err
	}
	if source != nil {
		source.removeProfile(nativeID)
	}
	return nil
}

func (f *fakeMutator) Create(_ context.Context, _ string, name string) error {
	f.mu.Lock()
	f.calls++
	f.last = name
	err := f.err
	source := f.source
	f.mu.Unlock()
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.created = append(f.created, name)
	f.mu.Unlock()
	if source != nil {
		source.addProfile(ProfileSnapshot{NativeID: name})
	}
	return nil
}

func (f *fakeMutator) snapshot() (int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.last
}

func TestStartCreateRequiresValidNameAndIdempotency(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	mutator := &fakeMutator{}
	inventory.WithMutator(mutator)

	if _, err := inventory.StartCreate(context.Background(), "hermes", "default", "key-1"); !errors.Is(err, ErrInstanceInvalidName) {
		t.Fatalf("default name = %v", err)
	}
	if calls, _ := mutator.snapshot(); calls != 0 {
		t.Fatal("invalid name started a process")
	}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "coder", "has space"); !errors.Is(err, ErrInvalidIdempotencyKey) {
		t.Fatalf("bad key = %v", err)
	}

	source.setProfiles([]ProfileSnapshot{{NativeID: "default", Default: true}, {NativeID: "coder", Default: false}})
	if _, err := inventory.StartCreate(context.Background(), "hermes", "coder", "key-2"); !errors.Is(err, ErrInstanceAlreadyExists) {
		t.Fatalf("duplicate = %v", err)
	}
}

func TestStartCreateIdempotencyAndSuccess(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	source.setProfiles([]ProfileSnapshot{{NativeID: "default", Default: true}})

	first, err := inventory.StartCreate(context.Background(), "hermes", "coder", "same-key")
	if err != nil || !first.Created {
		t.Fatalf("first = %#v %v", first, err)
	}
	replay, err := inventory.StartCreate(context.Background(), "hermes", "coder", "same-key")
	if err != nil || replay.Operation.ID != first.Operation.ID || replay.Created {
		t.Fatalf("replay = %#v %v", replay, err)
	}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "other", "same-key"); !errors.Is(err, ErrInstanceConflict) {
		t.Fatalf("payload conflict = %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), first.Operation.ID)
		if err == nil && got.Status == operation.StatusFailed {
			t.Fatalf("create failed: %#v", got)
		}
		if err == nil && got.Status == operation.StatusSucceeded {
			if calls, last := mutator.snapshot(); last != "coder" || calls != 1 {
				t.Fatalf("mutator calls=%d last=%s", calls, last)
			}
			listed, err := inventory.ListInstances(context.Background(), "hermes")
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range listed.Instances {
				if item.Name == "coder" && item.Availability == instance.Available {
					found = true
				}
			}
			if !found {
				t.Fatalf("authoritative list missing coder: %#v", listed.Instances)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("create operation did not succeed")
}
