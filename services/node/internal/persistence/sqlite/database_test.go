package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	_ "modernc.org/sqlite"
)

func TestMigrationsAreIdempotentAndNodeIdentityPersists(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	metadata := node.LocalMetadata{
		Name:         "TEST-NODE",
		Hostname:     "TEST-NODE",
		Platform:     "windows",
		Architecture: "amd64",
		NodeVersion:  "0.0.0-test",
	}

	firstDB, err := Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	var foreignKeys, busyTimeout int
	if err := firstDB.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys pragma: %v", err)
	}
	if err := firstDB.db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("read busy_timeout pragma: %v", err)
	}
	if foreignKeys != 1 || busyTimeout != 5000 {
		t.Fatalf("SQLite pragmas foreign_keys=%d busy_timeout=%d", foreignKeys, busyTimeout)
	}
	assertPragmasSurviveReconnect(t, ctx, firstDB.db)
	firstNode, err := firstDB.LoadOrCreateNode(ctx, metadata)
	if err != nil {
		t.Fatalf("first LoadOrCreateNode() error = %v", err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}

	secondDB, err := Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	secondNode, err := secondDB.LoadOrCreateNode(ctx, metadata)
	if err != nil {
		t.Fatalf("second LoadOrCreateNode() error = %v", err)
	}
	if firstNode.ID != secondNode.ID || !firstNode.CreatedAt.Equal(secondNode.CreatedAt) {
		t.Fatalf("node identity changed: first=%#v second=%#v", firstNode, secondNode)
	}
	if err := secondDB.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	raw, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename))
	if err != nil {
		t.Fatalf("open raw database: %v", err)
	}
	defer raw.Close()
	var migrations int
	if err := raw.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&migrations); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrations != 17 {
		t.Fatalf("migration count = %d, want 17", migrations)
	}
	var migrationRuns int
	if err := raw.QueryRow("SELECT COUNT(*) FROM database_migration_history").Scan(&migrationRuns); err != nil {
		t.Fatalf("count migration history: %v", err)
	}
	if migrationRuns != 1 {
		t.Fatalf("migration history count = %d, want 1 after repeat-safe reopen", migrationRuns)
	}
	for _, table := range []string{"schema_migrations", "nodes", "app_settings", "operations", "runtime_installations", "instances", "channel_bindings", "runtime_backups", "managed_skills", "managed_mcp_bindings", "model_provider_connections", "model_profiles", "runtime_model_defaults", "instance_model_bindings", "database_migration_history"} {
		var count int
		if err := raw.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %s count = %d, want 1", table, count)
		}
	}
}

func TestMigration16RepairsDatabaseThatRecordedEarlierVersion15WithoutSharedModels(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		version, err := migrationVersion(path)
		if err != nil {
			t.Fatal(err)
		}
		if version <= 14 {
			applyVersionedMigration(t, ctx, dataDir, filepath.Base(path))
		}
	}

	raw, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `CREATE TABLE managed_mcp_definitions (id TEXT PRIMARY KEY)`); err != nil {
		_ = raw.Close()
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (15, ?)", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		_ = raw.Close()
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	repaired, err := Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("Open() repair error = %v", err)
	}
	defer repaired.Close()
	for _, table := range []string{"model_provider_connections", "model_profiles", "runtime_model_defaults", "instance_model_bindings"} {
		var count int
		if err := repaired.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("repaired table %s count = %d err = %v", table, count, err)
		}
	}
	assertMigrationCount(t, dataDir, 17)
}

func assertPragmasSurviveReconnect(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	db.SetConnMaxLifetime(time.Nanosecond)

	for attempt := 0; attempt < 2; attempt++ {
		connection, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("acquire SQLite connection %d: %v", attempt+1, err)
		}
		var foreignKeys, busyTimeout int
		if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			_ = connection.Close()
			t.Fatalf("read foreign_keys on connection %d: %v", attempt+1, err)
		}
		if err := connection.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			_ = connection.Close()
			t.Fatalf("read busy_timeout on connection %d: %v", attempt+1, err)
		}
		if err := connection.Close(); err != nil {
			t.Fatalf("close SQLite connection %d: %v", attempt+1, err)
		}
		if foreignKeys != 1 || busyTimeout != 5000 {
			t.Fatalf("connection %d pragmas foreign_keys=%d busy_timeout=%d", attempt+1, foreignKeys, busyTimeout)
		}
		time.Sleep(time.Millisecond)
	}
	if db.Stats().MaxLifetimeClosed == 0 {
		t.Fatal("test did not force SQLite to replace the physical connection")
	}
	db.SetConnMaxLifetime(0)
}
