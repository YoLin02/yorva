package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestManagedMCPBindingOwnershipLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	localNode, err := db.LoadOrCreateNode(ctx, node.LocalMetadata{Name: "test", Hostname: "localhost", Platform: "windows", Architecture: "amd64", NodeVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.UpsertAcceptedInstallation(ctx, AcceptedInstallation{ID: "rtinst-mcp", NodeID: localNode.ID, RuntimeKind: "hermes", InstallPath: t.TempDir(), Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported, Status: "ACCEPTED", LastDetectedAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyInstanceSnapshot(ctx, "rtinst-mcp", []InstanceSnapshotEntry{{NativeID: "profile-a", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	instances, err := db.ListInstances(ctx, "rtinst-mcp")
	if err != nil || len(instances) != 1 {
		t.Fatalf("instances = %#v, %v", instances, err)
	}
	binding := ManagedMCPBinding{InstanceID: instances[0].ID, ServerID: "yorva-mcp-test", PresetID: "yorva-mcp-test", CreatedAt: now, UpdatedAt: now}
	if err := db.UpsertManagedMCPBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListManagedMCPBindings(ctx, instances[0].ID)
	if err != nil || len(items) != 1 || items[0].ServerID != binding.ServerID {
		t.Fatalf("bindings = %#v, %v", items, err)
	}
	if err := db.DeleteManagedMCPBinding(ctx, instances[0].ID, binding.ServerID); err != nil {
		t.Fatal(err)
	}
	items, err = db.ListManagedMCPBindings(ctx, instances[0].ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("bindings after delete = %#v, %v", items, err)
	}
}
