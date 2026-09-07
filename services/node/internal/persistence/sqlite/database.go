package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const databaseFilename = "yorva.db"

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Database struct {
	db *sql.DB
}

func Open(ctx context.Context, dataDir string) (*Database, error) {
	return openWithMigrationFiles(ctx, dataDir, migrationFiles)
}

func openWithMigrationFiles(ctx context.Context, dataDir string, files fs.FS) (*Database, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	if err := recoverInterruptedMigration(dataDir); err != nil {
		return nil, err
	}

	databasePath := filepath.Join(dataDir, databaseFilename)
	dsn := databasePath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	closeOnError := func(err error) (*Database, error) {
		_ = db.Close()
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return closeOnError(fmt.Errorf("connect SQLite: %w", err))
	}
	targetVersion, err := latestMigrationVersion(files)
	if err != nil {
		return closeOnError(err)
	}
	sourceVersion, err := currentMigrationVersion(ctx, db)
	if err != nil {
		return closeOnError(err)
	}
	if sourceVersion > targetVersion {
		return closeOnError(newMigrationError(MigrationUnsupported, sourceVersion, targetVersion, errDatabaseNewer))
	}
	var run *migrationRun
	if sourceVersion < targetVersion {
		run, err = beginMigrationProtection(ctx, db, dataDir, sourceVersion, targetVersion)
		if err != nil {
			return closeOnError(err)
		}
	}
	if err := migrate(ctx, db, files); err != nil {
		_ = db.Close()
		if run == nil {
			return nil, err
		}
		return nil, failAndRestoreMigration(dataDir, *run, err)
	}
	if err := verifyMigratedDatabase(ctx, db, targetVersion); err != nil {
		_ = db.Close()
		if run == nil {
			return nil, err
		}
		return nil, failAndRestoreMigration(dataDir, *run, err)
	}
	if run != nil {
		if err := completeMigration(ctx, db, dataDir, *run); err != nil {
			_ = db.Close()
			return nil, failAndRestoreMigration(dataDir, *run, err)
		}
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) SchemaVersion(ctx context.Context) (int, error) {
	return currentMigrationVersion(ctx, d.db)
}

func migrate(ctx context.Context, db *sql.DB, files fs.FS) error {
	if _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TEXT NOT NULL
        )
    `); err != nil {
		return fmt.Errorf("create schema migration ledger: %w", err)
	}

	entries, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list embedded migrations: %w", err)
	}
	sort.Strings(entries)
	for _, path := range entries {
		version, err := migrationVersion(path)
		if err != nil {
			return err
		}
		var applied int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied != 0 {
			continue
		}

		script, err := fs.ReadFile(files, path)
		if err != nil {
			return fmt.Errorf("read migration %d: %w", version, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)", version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}
	return nil
}

func latestMigrationVersion(files fs.FS) (int, error) {
	entries, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return 0, fmt.Errorf("list embedded migrations: %w", err)
	}
	if len(entries) == 0 {
		return 0, errors.New("no embedded migrations")
	}
	sort.Strings(entries)
	latest := 0
	for _, path := range entries {
		version, err := migrationVersion(path)
		if err != nil {
			return 0, err
		}
		if version != latest+1 {
			return 0, fmt.Errorf("migration sequence is not contiguous at %d", version)
		}
		latest = version
	}
	return latest, nil
}

func migrationVersion(path string) (int, error) {
	name := filepath.Base(path)
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("invalid migration filename %q", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid migration filename %q", name)
	}
	return version, nil
}
