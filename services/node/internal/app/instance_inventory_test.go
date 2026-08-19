package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeProfileSource struct {
	profiles []ProfileSnapshot
	err      error
	calls    int
}

func (f *fakeProfileSource) List(context.Context, string) ([]ProfileSnapshot, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return append([]ProfileSnapshot(nil), f.profiles...), nil
}

type inventoryDiscoverer struct {
	result yorvaruntime.Discovery
}

func (d inventoryDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	return d.result, nil
}

func TestListInstancesReconcilesAvailabilityAndIdentity(t *testing.T) {
	ctx := context.Background()
	inventory, source := newTestInventory(t, []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, nil)

	first, err := inventory.ListInstances(ctx, "hermes")
	if err != nil {
		t.Fatal(err)
	}
	if first.Freshness != "FRESH" || first.Capabilities.Lifecycle || !first.Capabilities.Instances {
		t.Fatalf("first list = %#v", first)
	}
	if len(first.Instances) != 2 || first.Instances[0].Name != "default" || !first.Instances[0].Protected {
		t.Fatalf("default row = %#v", first.Instances)
	}
	var coderID string
	for _, row := range first.Instances {
		if row.Name == "coder" {
			coderID = row.InstanceID
			if row.Availability != instance.Available {
				t.Fatalf("coder = %#v", row)
			}
		}
		if row.InstanceID == row.Name {
			t.Fatal("instanceId leaked nativeId")
		}
	}

	source.profiles = []ProfileSnapshot{{NativeID: "default", Default: true}}
	second, err := inventory.ListInstances(ctx, "hermes")
	if err != nil {
		t.Fatal(err)
	}
	got, err := inventory.GetInstance(ctx, coderID)
	if err != nil || got.Availability != instance.Missing || got.InstanceID != coderID {
		t.Fatalf("external remove = %#v, %v", got, err)
	}
	if len(second.Instances) != 2 {
		t.Fatalf("tombstone dropped: %#v", second.Instances)
	}

	source.profiles = []ProfileSnapshot{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}
	if _, err := inventory.ListInstances(ctx, "hermes"); err != nil {
		t.Fatal(err)
	}
	restored, err := inventory.GetInstance(ctx, coderID)
	if err != nil || restored.Availability != instance.Available || restored.InstanceID != coderID {
		t.Fatalf("reappear = %#v, %v", restored, err)
	}
}

func TestListInstancesQueryFailureIsUnknownNotMissing(t *testing.T) {
	ctx := context.Background()
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	if _, err := inventory.ListInstances(ctx, "hermes"); err != nil {
		t.Fatal(err)
	}
	source.err = ErrInstanceOutputUnrecognized
	failed, err := inventory.ListInstances(ctx, "hermes")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Freshness != "UNKNOWN" || failed.ErrorCode != yorvaruntime.ErrorInstanceOutputUnrecognized {
		t.Fatalf("failed list = %#v", failed)
	}
	if len(failed.Instances) != 1 || failed.Instances[0].Availability != instance.Unknown {
		t.Fatalf("must not infer MISSING: %#v", failed.Instances)
	}
}

func TestListInstancesRejectsUnsupportedRuntime(t *testing.T) {
	inventory, _ := newTestInventory(t, nil, nil)
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoveryNotInstalled}},
	}); err != nil {
		t.Fatal(err)
	}
	inventory.discovery = NewRuntimeDiscovery(registry, nil)
	if _, err := inventory.ListInstances(context.Background(), "hermes"); !errors.Is(err, ErrRuntimeNotSupported) {
		t.Fatalf("unsupported = %v", err)
	}
	if _, err := inventory.ListInstances(context.Background(), "other"); !errors.Is(err, ErrInstanceRuntimeNotFound) {
		t.Fatalf("unknown runtime id = %v", err)
	}
	if _, err := inventory.GetInstance(context.Background(), "inst_missing"); !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("missing instance = %v", err)
	}
}

func newTestInventory(t *testing.T, profiles []ProfileSnapshot, listErr error) (*InstanceInventory, *fakeProfileSource) {
	t.Helper()
	db, err := sqlite.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	local, err := db.LoadOrCreateNode(context.Background(), node.LocalMetadata{
		Name: "TEST", Hostname: "TEST", Platform: "windows", Architecture: "amd64", NodeVersion: "0.0.0-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{
			RuntimeKind: "hermes",
			State:       yorvaruntime.DiscoverySupported,
			Selected:    &yorvaruntime.Candidate{Path: `C:\hermes\bin\hermes.exe`, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	source := &fakeProfileSource{profiles: profiles, err: listErr}
	inventory := NewInstanceInventory(NewRuntimeDiscovery(registry, nil), db, source, local.ID)
	inventory.now = func() time.Time { return time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC) }
	return inventory, source
}
