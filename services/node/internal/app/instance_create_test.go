package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

type fakeMutator struct {
	source  *fakeProfileSource
	err     error
	calls   int
	last    string
	created []string
}

func (f *fakeMutator) Create(_ context.Context, _ string, name string) error {
	f.calls++
	f.last = name
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, name)
	if f.source != nil {
		f.source.profiles = append(append([]ProfileSnapshot(nil), f.source.profiles...), ProfileSnapshot{NativeID: name})
	}
	return nil
}

func TestStartCreateRequiresValidNameAndIdempotency(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	mutator := &fakeMutator{}
	inventory.WithMutator(mutator)

	if _, err := inventory.StartCreate(context.Background(), "hermes", "default", "key-1"); !errors.Is(err, ErrInstanceInvalidName) {
		t.Fatalf("default name = %v", err)
	}
	if mutator.calls != 0 {
		t.Fatal("invalid name started a process")
	}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "coder", "has space"); !errors.Is(err, ErrInvalidIdempotencyKey) {
		t.Fatalf("bad key = %v", err)
	}

	source.profiles = []ProfileSnapshot{{NativeID: "default", Default: true}, {NativeID: "coder", Default: false}}
	if _, err := inventory.StartCreate(context.Background(), "hermes", "coder", "key-2"); !errors.Is(err, ErrInstanceAlreadyExists) {
		t.Fatalf("duplicate = %v", err)
	}
}

func TestStartCreateIdempotencyAndSuccess(t *testing.T) {
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	mutator := &fakeMutator{source: source}
	inventory.WithMutator(mutator)
	source.profiles = []ProfileSnapshot{{NativeID: "default", Default: true}}

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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := inventory.db.GetOperation(context.Background(), first.Operation.ID)
		if err == nil && got.Status == operation.StatusFailed {
			t.Fatalf("create failed: %#v", got)
		}
		if err == nil && got.Status == operation.StatusSucceeded {
			if mutator.last != "coder" || mutator.calls != 1 {
				t.Fatalf("mutator = %#v", mutator)
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
