package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	_ "modernc.org/sqlite"
)

func TestRuntimeBackupReaderProjectsOnlySafeLastObservedMetadata(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	runtimeID := seedInstallation(t, db)
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	entry := RuntimeBackupIndexEntry{
		ID: "backup_safe", RuntimeInstallationID: runtimeID,
		FormatVersion: "yorva.hermes.runtime-backup.v1", RuntimeVersion: "0.20.5",
		ArtifactPath: filepath.Join(t.TempDir(), "not-present.age"), SizeBytes: 4096,
		ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		State:          RuntimeBackupAvailable, KeyMode: RuntimeBackupDeviceKey, KeyRef: "private-key-reference",
		CreatedAt: now, VerifiedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}
	if err := db.InsertVerifiedRuntimeBackup(ctx, entry); err != nil {
		t.Fatal(err)
	}
	reader := NewRuntimeBackupReader(db, runtimeID)
	installation := yorvaruntime.Installation{RuntimeKind: "hermes", Path: "managed-hermes", Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported}
	items, err := reader.ListBackups(ctx, installation)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListBackups() = %#v, %v", items, err)
	}
	if items[0].State != yorvaruntime.BackupAvailable || items[0].KeyMode != yorvaruntime.BackupKeyDevice || !items[0].VerifiedAt.Equal(entry.VerifiedAt) {
		t.Fatalf("safe projection = %#v", items[0])
	}
	// The indexed file deliberately does not exist. A normal read must retain
	// the last-observed state rather than probing the destination or guessing.
	stored, err := db.GetRuntimeBackup(ctx, runtimeID, entry.ID)
	if err != nil || stored.State != RuntimeBackupAvailable || stored.KeyRef != entry.KeyRef || stored.ArtifactPath != entry.ArtifactPath {
		t.Fatalf("ordinary read mutated index = %#v, %v", stored, err)
	}
	got, err := reader.GetBackup(ctx, installation, entry.ID)
	if err != nil || got.ID != entry.ID {
		t.Fatalf("GetBackup() = %#v, %v", got, err)
	}
	if _, err := reader.GetBackup(ctx, installation, "backup_missing"); !errors.Is(err, yorvaruntime.ErrBackupNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func TestRuntimeBackupIndexPreservesVerifiedIdentityAndReconcilesState(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	runtimeID := seedInstallation(t, db)
	created := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	entry := RuntimeBackupIndexEntry{
		ID: "backup_01", RuntimeInstallationID: runtimeID,
		FormatVersion: "1", RuntimeVersion: "0.20.5",
		ArtifactPath: `C:\Backups\hermes-20260825.yorva-backup.age`,
		SizeBytes:    4096, ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		State: RuntimeBackupAvailable, KeyMode: RuntimeBackupDeviceKey, KeyRef: "backup-key-01",
		CreatedAt: created, VerifiedAt: created.Add(time.Minute), UpdatedAt: created.Add(time.Minute),
	}
	if err := db.InsertVerifiedRuntimeBackup(ctx, entry); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetRuntimeBackup(ctx, runtimeID, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != entry {
		t.Fatalf("stored entry = %#v, want %#v", got, entry)
	}
	if err := db.InsertVerifiedRuntimeBackup(ctx, entry); err == nil {
		t.Fatal("duplicate artifact identity must not overwrite the verified row")
	}

	changedAt := created.Add(2 * time.Minute)
	if err := db.ReconcileRuntimeBackupState(ctx, runtimeID, entry.ID, RuntimeBackupChanged, nil, changedAt); err != nil {
		t.Fatal(err)
	}
	changed, err := db.GetRuntimeBackup(ctx, runtimeID, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if changed.State != RuntimeBackupChanged || !changed.UpdatedAt.Equal(changedAt) || !changed.VerifiedAt.Equal(entry.VerifiedAt) {
		t.Fatalf("changed entry = %#v", changed)
	}
	if changed.ArtifactPath != entry.ArtifactPath || changed.ChecksumSHA256 != entry.ChecksumSHA256 || changed.KeyRef != entry.KeyRef {
		t.Fatalf("reconciliation changed immutable identity: %#v", changed)
	}

	reverifiedAt := created.Add(3 * time.Minute)
	if err := db.ReconcileRuntimeBackupState(ctx, runtimeID, entry.ID, RuntimeBackupAvailable, &reverifiedAt, reverifiedAt); err != nil {
		t.Fatal(err)
	}
	reverified, err := db.GetRuntimeBackup(ctx, runtimeID, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reverified.State != RuntimeBackupAvailable || !reverified.VerifiedAt.Equal(reverifiedAt) {
		t.Fatalf("reverified entry = %#v", reverified)
	}
}

func TestRuntimeBackupIndexEnforcesRuntimeScopeAndSecretBoundary(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	runtimeID := seedInstallation(t, db)
	now := time.Date(2026, 8, 25, 11, 0, 0, 0, time.UTC)
	base := RuntimeBackupIndexEntry{
		ID: "backup_portable", RuntimeInstallationID: runtimeID,
		FormatVersion: "1", RuntimeVersion: "0.20.5", ArtifactPath: `C:\Backups\portable.age`,
		SizeBytes: 1024, ChecksumSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		State: RuntimeBackupAvailable, KeyMode: RuntimeBackupPassphrase,
		CreatedAt: now, VerifiedAt: now, UpdatedAt: now,
	}
	if err := db.InsertVerifiedRuntimeBackup(ctx, base); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetRuntimeBackup(ctx, runtimeID, base.ID)
	if err != nil || got.KeyRef != "" || got.KeyMode != RuntimeBackupPassphrase {
		t.Fatalf("portable key authority = %#v, %v", got, err)
	}

	invalid := base
	invalid.ID = "backup_secret"
	invalid.ArtifactPath = `C:\Backups\secret.age`
	invalid.KeyRef = "must-not-persist-a-passphrase"
	if err := db.InsertVerifiedRuntimeBackup(ctx, invalid); err == nil {
		t.Fatal("portable passphrase material must not enter the index")
	}
	invalid = base
	invalid.ID = "backup_unverified"
	invalid.ArtifactPath = `C:\Backups\unverified.age`
	invalid.State = RuntimeBackupUnknown
	if err := db.InsertVerifiedRuntimeBackup(ctx, invalid); err == nil {
		t.Fatal("unverified artifact must not enter the index")
	}
	if err := db.ReconcileRuntimeBackupState(ctx, runtimeID, base.ID, RuntimeBackupAvailable, nil, now); err == nil {
		t.Fatal("available reconciliation without fresh verification must fail")
	}
	if err := db.ReconcileRuntimeBackupState(ctx, "different-runtime", base.ID, RuntimeBackupMissing, nil, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-Runtime reconciliation = %v, want sql.ErrNoRows", err)
	}
}

func TestRuntimeBackupMigrationPreservesLegacyInstanceRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	for _, name := range []string{
		"001_initial.sql",
		"002_operations_and_installations.sql",
		"003_hermes_host_mutation.sql",
		"004_operation_source_pin.sql",
		"005_operation_ownership_nonce.sql",
		"006_operation_transaction_id.sql",
		"007_instances.sql",
		"008_instance_operations.sql",
		"009_instance_lifecycle_operations.sql",
		"010_channel_bindings.sql",
	} {
		applyVersionedMigration(t, ctx, dir, name)
	}

	raw, err := sql.Open("sqlite", filepath.Join(dir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `
        CREATE TABLE backups (
            id TEXT PRIMARY KEY,
            instance_id TEXT NOT NULL REFERENCES instances(id),
            kind TEXT NOT NULL,
            path TEXT NOT NULL,
            size_bytes INTEGER,
            checksum TEXT,
            state TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        )
    `); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `
        INSERT INTO nodes(id, name, hostname, platform, architecture, node_version, created_at, updated_at)
        VALUES ('node_legacy', 'legacy', 'legacy', 'windows', 'amd64', '0.1', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z');
        INSERT INTO runtime_installations(id, node_id, runtime_kind, install_path, version, support_state, status, created_at, updated_at)
        VALUES ('runtime_legacy', 'node_legacy', 'hermes', 'legacy-path', '0.20.2', 'SUPPORTED', 'ACCEPTED', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z');
        INSERT INTO instances(id, runtime_installation_id, native_id, name, availability, created_at, updated_at)
        VALUES ('instance_legacy', 'runtime_legacy', 'default', 'default', 'AVAILABLE', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z');
        INSERT INTO backups(id, instance_id, kind, path, size_bytes, checksum, state, created_at, updated_at)
        VALUES ('backup_legacy', 'instance_legacy', 'full', 'legacy.zip', 123, 'legacy-sum', 'AVAILABLE', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z');
    `); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err = sql.Open("sqlite", filepath.Join(dir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var legacyID, legacyInstanceID, legacyPath string
	if err := raw.QueryRowContext(ctx, "SELECT id, instance_id, path FROM backups WHERE id = 'backup_legacy'").Scan(&legacyID, &legacyInstanceID, &legacyPath); err != nil {
		t.Fatal(err)
	}
	if legacyID != "backup_legacy" || legacyInstanceID != "instance_legacy" || legacyPath != "legacy.zip" {
		t.Fatalf("legacy row changed: id=%q instance=%q path=%q", legacyID, legacyInstanceID, legacyPath)
	}
	var runtimeRows int
	if err := raw.QueryRowContext(ctx, "SELECT COUNT(*) FROM runtime_backups").Scan(&runtimeRows); err != nil {
		t.Fatal(err)
	}
	if runtimeRows != 0 {
		t.Fatalf("legacy rows were reinterpreted: runtime backup count = %d", runtimeRows)
	}
	assertMigrationCount(t, dir, 15)
}
