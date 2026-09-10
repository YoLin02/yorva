package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestInstanceInventoryResolveManagementTarget(t *testing.T) {
	inventory, db, instanceID, accepted := newB3ManagementTargetFixture(t)

	target, err := inventory.ResolveManagementTarget(context.Background(), instanceID)
	if err != nil {
		t.Fatalf("ResolveManagementTarget() error = %v", err)
	}
	if target.NativeID != "coder" {
		t.Fatalf("NativeID = %q, want coder", target.NativeID)
	}
	if target.Installation.RuntimeKind != "hermes" || target.Installation.Path != accepted.InstallPath ||
		target.Installation.Version != accepted.Version || target.Installation.SupportState != yorvaruntime.DiscoverySupported {
		t.Fatalf("Installation = %#v", target.Installation)
	}
	if target.Bundle.Descriptor.Kind != "hermes" {
		t.Fatalf("Bundle descriptor = %#v", target.Bundle.Descriptor)
	}

	if _, err := inventory.ResolveManagementTarget(context.Background(), ""); !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("empty instance error = %v, want ErrInstanceNotFound", err)
	}
	if _, err := inventory.ResolveManagementTarget(context.Background(), "inst_missing"); !errors.Is(err, ErrInstanceNotFound) {
		t.Fatalf("missing instance error = %v, want ErrInstanceNotFound", err)
	}

	if err := db.ApplyInstanceSnapshot(context.Background(), accepted.ID, nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.ResolveManagementTarget(context.Background(), instanceID); !errors.Is(err, ErrInstanceNotAvailable) {
		t.Fatalf("missing native instance error = %v, want ErrInstanceNotAvailable", err)
	}
}

func TestInstanceInventoryResolveManagementTargetRejectsUnacceptedOrUnregisteredInstallation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*InstanceInventory, *sqlite.AcceptedInstallation)
	}{
		{
			name: "not accepted",
			mutate: func(_ *InstanceInventory, accepted *sqlite.AcceptedInstallation) {
				accepted.Status = "PENDING"
			},
		},
		{
			name: "unsupported",
			mutate: func(_ *InstanceInventory, accepted *sqlite.AcceptedInstallation) {
				accepted.SupportState = yorvaruntime.DiscoveryUnsupported
			},
		},
		{
			name: "unregistered",
			mutate: func(inventory *InstanceInventory, _ *sqlite.AcceptedInstallation) {
				inventory.discovery = NewRuntimeDiscovery(yorvaruntime.NewRegistry(), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventory, db, instanceID, accepted := newB3ManagementTargetFixture(t)
			tt.mutate(inventory, &accepted)
			if err := db.UpsertAcceptedInstallation(context.Background(), accepted); err != nil {
				t.Fatal(err)
			}
			if _, err := inventory.ResolveManagementTarget(context.Background(), instanceID); !errors.Is(err, ErrRuntimeNotSupported) {
				t.Fatalf("ResolveManagementTarget() error = %v, want ErrRuntimeNotSupported", err)
			}
		})
	}
}

func TestInstanceInventoryResolveManagementTargetContextAndQueryFailure(t *testing.T) {
	inventory, db, instanceID, _ := newB3ManagementTargetFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inventory.ResolveManagementTarget(ctx, instanceID); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled error = %v, want context.Canceled", err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := inventory.ResolveManagementTarget(context.Background(), instanceID); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("closed database error = %v, want ErrManagementQueryFailed", err)
	}
}

func TestInstanceInventoryResolveRuntimeManagementTargetUsesLiveAcceptedInstallation(t *testing.T) {
	inventory, _, _, accepted := newB3ManagementTargetFixture(t)

	target, err := inventory.ResolveRuntimeManagementTarget(context.Background(), "hermes")
	if err != nil {
		t.Fatalf("ResolveRuntimeManagementTarget() error = %v", err)
	}
	if target.InstallationID != accepted.ID || target.Installation.RuntimeKind != "hermes" ||
		target.Installation.Path != accepted.InstallPath || target.Installation.Version != accepted.Version ||
		target.Installation.SupportState != yorvaruntime.DiscoverySupported || target.Bundle.Descriptor.Kind != "hermes" ||
		target.Bundle.BackupRead == nil || target.Bundle.BackupMutate != nil || target.Bundle.Restore != nil {
		t.Fatalf("target = %#v", target)
	}
	backups, err := target.Bundle.BackupRead.ListBackups(context.Background(), target.Installation)
	if err != nil || len(backups) != 0 {
		t.Fatalf("bound Runtime backup index = %#v, %v", backups, err)
	}

	if _, err := inventory.ResolveRuntimeManagementTarget(context.Background(), ""); !errors.Is(err, ErrRuntimeNotSupported) {
		t.Fatalf("empty runtime error = %v, want ErrRuntimeNotSupported", err)
	}
	if _, err := inventory.ResolveRuntimeManagementTarget(context.Background(), "unknown"); !errors.Is(err, ErrRuntimeNotSupported) {
		t.Fatalf("unknown runtime error = %v, want ErrRuntimeNotSupported", err)
	}
}

func newB3ManagementTargetFixture(t *testing.T) (*InstanceInventory, *sqlite.Database, string, sqlite.AcceptedInstallation) {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	local, err := db.LoadOrCreateNode(ctx, node.LocalMetadata{
		Name: "TEST", Hostname: "TEST", Platform: "windows", Architecture: "amd64", NodeVersion: "0.0.0-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	accepted := sqlite.AcceptedInstallation{
		ID: "rtinst_b3", NodeID: local.ID, RuntimeKind: "hermes", InstallPath: `C:\hermes\hermes.exe`,
		Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported, Status: "ACCEPTED",
		LastDetectedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.UpsertAcceptedInstallation(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyInstanceSnapshot(ctx, accepted.ID, []sqlite.InstanceSnapshotEntry{{NativeID: "coder"}}, now); err != nil {
		t.Fatal(err)
	}
	instances, err := db.ListInstances(ctx, accepted.ID)
	if err != nil || len(instances) != 1 {
		t.Fatalf("ListInstances() = %#v, %v", instances, err)
	}

	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{
			RuntimeKind: "hermes",
			State:       yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{
				Path: accepted.InstallPath, Version: accepted.Version, State: yorvaruntime.DiscoverySupported,
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	inventory := NewInstanceInventory(NewRuntimeDiscovery(registry, nil), db, local.ID)
	return inventory, db, instances[0].ID, accepted
}
