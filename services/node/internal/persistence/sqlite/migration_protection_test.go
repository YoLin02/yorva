package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestSchema16UpgradeCreatesVerifiedProtectionAndHistory(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	applyEmbeddedMigrationsThrough(t, ctx, dataDir, 16)

	db, err := Open(ctx, dataDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	var source, target int
	var digest, state string
	if err := db.db.QueryRowContext(ctx, `SELECT source_version, target_version, protection_sha256, state
        FROM database_migration_history ORDER BY id DESC LIMIT 1`).Scan(&source, &target, &digest, &state); err != nil {
		t.Fatalf("read migration history: %v", err)
	}
	if source != 16 || target != 17 || len(digest) != 64 || state != "SUCCEEDED" {
		t.Fatalf("history = source %d target %d digest %q state %q", source, target, digest, state)
	}

	run, found, err := readMigrationState(dataDir)
	if err != nil || !found || run.State != "SUCCEEDED" || run.ProtectionSHA256 != digest {
		t.Fatalf("migration state = %#v found=%v err=%v", run, found, err)
	}
	protectionPath := filepath.Join(dataDir, protectionDirectory, run.ProtectionFile)
	assertDatabaseVersion(t, protectionPath, 16)
	assertDatabaseVersion(t, filepath.Join(dataDir, databaseFilename), 17)
}

func TestFailedMigrationRestoresSchema16AndReturnsStableCode(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	applyEmbeddedMigrationsThrough(t, ctx, dataDir, 16)
	files := migrationsWith(t, "migrations/018_injected_failure.sql", "CREATE TABLE broken (")

	_, err := openWithMigrationFiles(ctx, dataDir, files)
	var migrationErr *MigrationError
	if !errors.As(err, &migrationErr) || migrationErr.Code != MigrationFailedRecovered {
		t.Fatalf("Open() error = %v, want %s", err, MigrationFailedRecovered)
	}
	assertDatabaseVersion(t, filepath.Join(dataDir, databaseFilename), 16)
	run, found, stateErr := readMigrationState(dataDir)
	if stateErr != nil || !found || run.State != "RESTORED" || run.ErrorCode != MigrationFailedRecovered {
		t.Fatalf("restored state = %#v found=%v err=%v", run, found, stateErr)
	}
}

func TestInterruptedMigrationRestoresProtectionBeforeRetry(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	applyEmbeddedMigrationsThrough(t, ctx, dataDir, 16)
	raw := openRawDatabase(t, dataDir)
	run, err := beginMigrationProtection(ctx, raw, dataDir, 16, 17)
	if err != nil {
		_ = raw.Close()
		t.Fatal(err)
	}
	if err := migrate(ctx, raw, migrationFiles); err != nil {
		_ = raw.Close()
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	assertDatabaseVersion(t, filepath.Join(dataDir, databaseFilename), 17)

	if err := recoverInterruptedMigration(dataDir); err != nil {
		t.Fatalf("recoverInterruptedMigration() error = %v", err)
	}
	assertDatabaseVersion(t, filepath.Join(dataDir, databaseFilename), 16)
	recovered, found, err := readMigrationState(dataDir)
	if err != nil || !found || recovered.State != "RESTORED" || recovered.ProtectionFile != run.ProtectionFile {
		t.Fatalf("recovered state = %#v found=%v err=%v", recovered, found, err)
	}
}

func TestTamperedInterruptedProtectionBlocksStartup(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	applyEmbeddedMigrationsThrough(t, ctx, dataDir, 16)
	raw := openRawDatabase(t, dataDir)
	run, err := beginMigrationProtection(ctx, raw, dataDir, 16, 17)
	if err != nil {
		_ = raw.Close()
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, protectionDirectory, run.ProtectionFile), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Open(ctx, dataDir)
	var migrationErr *MigrationError
	if !errors.As(err, &migrationErr) || migrationErr.Code != MigrationRecoveryRequired {
		t.Fatalf("Open() error = %v, want %s", err, MigrationRecoveryRequired)
	}
}

func TestProtectionRetentionDeletesOnlyOldKnownFiles(t *testing.T) {
	dataDir := t.TempDir()
	root := filepath.Join(dataDir, protectionDirectory)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	names := []string{
		"schema-016-to-017-20260903T090000000Z-000000000001.db",
		"schema-016-to-017-20260903T090100000Z-000000000002.db",
		"schema-016-to-017-20260903T090200000Z-000000000003.db",
		"schema-016-to-017-20260903T090300000Z-000000000004.db",
		"schema-016-to-017-20260903T090400000Z-000000000005.db",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "user-note.db"), []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := retainProtections(dataDir); err != nil {
		t.Fatal(err)
	}
	for index, name := range names {
		_, err := os.Stat(filepath.Join(root, name))
		if index < 2 && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("old protection %s was not removed: %v", name, err)
		}
		if index >= 2 && err != nil {
			t.Fatalf("recent protection %s missing: %v", name, err)
		}
	}
	if payload, err := os.ReadFile(filepath.Join(root, "user-note.db")); err != nil || string(payload) != "preserve" {
		t.Fatalf("unknown file changed: %q err=%v", payload, err)
	}
}

func applyEmbeddedMigrationsThrough(t *testing.T, ctx context.Context, dataDir string, through int) {
	t.Helper()
	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		version, err := migrationVersion(path)
		if err != nil {
			t.Fatal(err)
		}
		if version <= through {
			applyVersionedMigration(t, ctx, dataDir, filepath.Base(path))
		}
	}
}

func migrationsWith(t *testing.T, name, script string) fstest.MapFS {
	t.Helper()
	result := fstest.MapFS{}
	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		payload, err := fs.ReadFile(migrationFiles, path)
		if err != nil {
			t.Fatal(err)
		}
		result[path] = &fstest.MapFile{Data: payload, Mode: 0o600}
	}
	result[name] = &fstest.MapFile{Data: []byte(script), Mode: 0o600}
	return result
}

func openRawDatabase(t *testing.T, dataDir string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename)+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	return db
}

func assertDatabaseVersion(t *testing.T, path string, want int) {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var got int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("database %s version = %d, want %d", filepath.Base(path), got, want)
	}
}
