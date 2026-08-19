package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestInstanceSnapshotRetainsTombstonesAndIdentity(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	installationID := seedInstallation(t, db)
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, now); err != nil {
		t.Fatal(err)
	}
	first, err := db.ListInstances(ctx, installationID)
	if err != nil || len(first) != 2 {
		t.Fatalf("first list = %#v, %v", first, err)
	}
	var coderID string
	for _, row := range first {
		if row.NativeID == "coder" {
			coderID = row.ID
		}
		if row.Availability != instance.Available {
			t.Fatalf("first availability = %s", row.Availability)
		}
	}

	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{
		{NativeID: "default", Default: true},
	}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	second, err := db.ListInstances(ctx, installationID)
	if err != nil || len(second) != 2 {
		t.Fatalf("tombstone list = %#v, %v", second, err)
	}
	var missing instance.Instance
	for _, row := range second {
		if row.NativeID == "coder" {
			missing = row
		}
	}
	if missing.ID != coderID || missing.Availability != instance.Missing {
		t.Fatalf("missing tombstone = %#v", missing)
	}

	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{
		{NativeID: "default", Default: true},
		{NativeID: "coder", Default: false},
	}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	restored, err := db.GetInstance(ctx, coderID)
	if err != nil || restored.Availability != instance.Available || restored.ID != coderID {
		t.Fatalf("reappear = %#v, %v", restored, err)
	}

	if err := db.MarkInstancesUnknown(ctx, installationID, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	unknown, err := db.ListInstances(ctx, installationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range unknown {
		if row.Availability != instance.Unknown {
			t.Fatalf("unknown availability = %#v", row)
		}
		if row.ID == "" || (row.NativeID != "default" && row.NativeID != "coder") {
			t.Fatalf("unknown identity lost: %#v", row)
		}
	}

	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{
		{NativeID: "default", Default: true},
		{NativeID: "default", Default: true},
	}, now); err == nil {
		t.Fatal("duplicate native identity must fail closed")
	}
}

func TestInstanceUniquenessAndMigrationFromPhase3(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	installationID := seedInstallation(t, db)
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{{NativeID: "default", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "unused.db"))
	if err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	if _, err := db.db.ExecContext(ctx, `
        INSERT INTO instances(id, runtime_installation_id, native_id, name, is_default, is_protected, availability, created_at, updated_at)
        VALUES ('inst_dup', ?, 'default', 'default', 1, 1, 'AVAILABLE', ?, ?)
    `, installationID, formatTime(now), formatTime(now)); err == nil {
		t.Fatal("duplicate (installation, native_id) must be rejected")
	}
}

func openInstanceTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.LoadOrCreateNode(context.Background(), node.LocalMetadata{
		Name: "TEST", Hostname: "TEST", Platform: "windows", Architecture: "amd64", NodeVersion: "0.0.0-test",
	}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedInstallation(t *testing.T, db *Database) string {
	t.Helper()
	now := time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	id, err := NewInstallationID()
	if err != nil {
		t.Fatal(err)
	}
	nodeRow, err := db.LoadOrCreateNode(context.Background(), node.LocalMetadata{
		Name: "TEST", Hostname: "TEST", Platform: "windows", Architecture: "amd64", NodeVersion: "0.0.0-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAcceptedInstallation(context.Background(), AcceptedInstallation{
		ID: id, NodeID: nodeRow.ID, RuntimeKind: yorvaruntime.Kind("hermes"),
		InstallPath: `C:\hermes\gen\bin\hermes.exe`, Version: "0.20.2",
		SupportState: yorvaruntime.DiscoverySupported, Status: "ACCEPTED",
		LastDetectedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetAcceptedInstallationByID(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestInstancesMigrateFromPhase3Schema(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	for _, name := range []string{
		"001_initial.sql",
		"002_operations_and_installations.sql",
		"003_hermes_host_mutation.sql",
		"004_operation_source_pin.sql",
		"005_operation_ownership_nonce.sql",
		"006_operation_transaction_id.sql",
	} {
		applyVersionedMigration(t, ctx, dir, name)
	}
	assertMigrationCount(t, dir, 6)
	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationCount(t, dir, 8)
	raw, err := sql.Open("sqlite", filepath.Join(dir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var count int
	if err := raw.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'instances'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("instances table after Phase 3 upgrade: count=%d err=%v", count, err)
	}
}

func applyVersionedMigration(t *testing.T, ctx context.Context, dataDir, name string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TEXT NOT NULL
        )
    `); err != nil {
		t.Fatal(err)
	}
	script, err := migrationFiles.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(script)); err != nil {
		t.Fatal(err)
	}
	version, err := migrationVersion("migrations/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)", version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
}

func TestGetInstanceNotFound(t *testing.T) {
	_, err := openInstanceTestDB(t).GetInstance(context.Background(), "inst_missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetInstance missing = %v", err)
	}
}
