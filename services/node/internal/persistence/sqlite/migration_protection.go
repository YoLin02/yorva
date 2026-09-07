package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	migrationStateFilename = "migration-state.json"
	protectionDirectory    = "migration-protections"
	maxProtectionFiles     = 3
	maxMigrationStateBytes = 16 * 1024
)

type MigrationErrorCode string

const (
	MigrationProtectionFailed MigrationErrorCode = "DATABASE_MIGRATION_PROTECTION_FAILED"
	MigrationFailedRecovered  MigrationErrorCode = "DATABASE_MIGRATION_FAILED_RECOVERED"
	MigrationRecoveryRequired MigrationErrorCode = "DATABASE_MIGRATION_RECOVERY_REQUIRED"
	MigrationUnsupported      MigrationErrorCode = "DATABASE_SCHEMA_UNSUPPORTED"
)

var (
	errDatabaseNewer = errors.New("database schema is newer than this YORVA build")
	protectionNameRE = regexp.MustCompile(`^schema-[0-9]{3}-to-[0-9]{3}-[0-9]{8}T[0-9]{9}Z-[0-9a-f]{12}\.db$`)
)

type MigrationError struct {
	Code          MigrationErrorCode
	SourceVersion int
	TargetVersion int
	Err           error
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf("%s (schema %d to %d)", e.Code, e.SourceVersion, e.TargetVersion)
}

func (e *MigrationError) Unwrap() error { return e.Err }

func newMigrationError(code MigrationErrorCode, source, target int, err error) error {
	return &MigrationError{Code: code, SourceVersion: source, TargetVersion: target, Err: err}
}

type migrationRun struct {
	SchemaVersion    int                `json:"schemaVersion"`
	State            string             `json:"state"`
	SourceVersion    int                `json:"sourceVersion"`
	TargetVersion    int                `json:"targetVersion"`
	ProtectionFile   string             `json:"protectionFile,omitempty"`
	ProtectionSHA256 string             `json:"protectionSha256,omitempty"`
	ErrorCode        MigrationErrorCode `json:"errorCode,omitempty"`
	StartedAt        time.Time          `json:"startedAt"`
	CompletedAt      *time.Time         `json:"completedAt,omitempty"`
}

func currentMigrationVersion(ctx context.Context, db *sql.DB) (int, error) {
	var ledger int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`).Scan(&ledger); err != nil {
		return 0, fmt.Errorf("inspect schema migration ledger: %w", err)
	}
	if ledger == 0 {
		var userTables int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Scan(&userTables); err != nil {
			return 0, fmt.Errorf("inspect empty database: %w", err)
		}
		if userTables != 0 {
			return 0, newMigrationError(MigrationUnsupported, 0, 0, errors.New("unversioned non-empty database"))
		}
		return 0, nil
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read current schema version: %w", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count schema migrations: %w", err)
	}
	if count != version {
		return 0, newMigrationError(MigrationUnsupported, version, version, errors.New("schema migration ledger is not contiguous"))
	}
	return version, nil
}

func beginMigrationProtection(ctx context.Context, db *sql.DB, dataDir string, source, target int) (*migrationRun, error) {
	run := migrationRun{SchemaVersion: 1, State: "RUNNING", SourceVersion: source, TargetVersion: target, StartedAt: time.Now().UTC()}
	if source > 0 {
		root := filepath.Join(dataDir, protectionDirectory)
		if err := os.MkdirAll(root, 0o700); err != nil {
			return nil, newMigrationError(MigrationProtectionFailed, source, target, err)
		}
		name, err := newProtectionName(source, target, run.StartedAt)
		if err != nil {
			return nil, newMigrationError(MigrationProtectionFailed, source, target, err)
		}
		path := filepath.Join(root, name)
		quoted := strings.ReplaceAll(path, "'", "''")
		if _, err := db.ExecContext(ctx, "VACUUM INTO '"+quoted+"'"); err != nil {
			return nil, newMigrationError(MigrationProtectionFailed, source, target, err)
		}
		digest, err := fileSHA256(path)
		if err != nil {
			_ = os.Remove(path)
			return nil, newMigrationError(MigrationProtectionFailed, source, target, err)
		}
		run.ProtectionFile, run.ProtectionSHA256 = name, digest
	}
	if err := writeMigrationState(dataDir, run); err != nil {
		if run.ProtectionFile != "" {
			_ = os.Remove(filepath.Join(dataDir, protectionDirectory, run.ProtectionFile))
		}
		return nil, newMigrationError(MigrationProtectionFailed, source, target, err)
	}
	return &run, nil
}

func completeMigration(ctx context.Context, db *sql.DB, dataDir string, run migrationRun) error {
	if _, err := db.ExecContext(ctx, `INSERT INTO database_migration_history
        (source_version, target_version, protection_sha256, state, started_at, completed_at)
        VALUES (?, ?, ?, 'SUCCEEDED', ?, ?)`, run.SourceVersion, run.TargetVersion,
		run.ProtectionSHA256, run.StartedAt.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record migration completion: %w", err)
	}
	now := time.Now().UTC()
	run.State, run.CompletedAt = "SUCCEEDED", &now
	if err := writeMigrationState(dataDir, run); err != nil {
		return fmt.Errorf("publish migration completion: %w", err)
	}
	return retainProtections(dataDir)
}

func verifyMigratedDatabase(ctx context.Context, db *sql.DB, target int) error {
	current, err := currentMigrationVersion(ctx, db)
	if err != nil || current != target {
		return fmt.Errorf("verify schema version %d: current=%d: %w", target, current, err)
	}
	var quick string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&quick); err != nil || quick != "ok" {
		return fmt.Errorf("verify database integrity: %s: %w", quick, err)
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("verify foreign keys: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("verify foreign keys: violation found")
	}
	return rows.Err()
}

func failAndRestoreMigration(dataDir string, run migrationRun, cause error) error {
	if err := restoreMigrationSource(dataDir, run); err != nil {
		run.State, run.ErrorCode = "RECOVERY_REQUIRED", MigrationRecoveryRequired
		now := time.Now().UTC()
		run.CompletedAt = &now
		_ = writeMigrationState(dataDir, run)
		return newMigrationError(MigrationRecoveryRequired, run.SourceVersion, run.TargetVersion, errors.Join(cause, err))
	}
	run.State, run.ErrorCode = "RESTORED", MigrationFailedRecovered
	now := time.Now().UTC()
	run.CompletedAt = &now
	if err := writeMigrationState(dataDir, run); err != nil {
		return newMigrationError(MigrationRecoveryRequired, run.SourceVersion, run.TargetVersion, errors.Join(cause, err))
	}
	return newMigrationError(MigrationFailedRecovered, run.SourceVersion, run.TargetVersion, cause)
}

func recoverInterruptedMigration(dataDir string) error {
	run, found, err := readMigrationState(dataDir)
	if err != nil {
		return newMigrationError(MigrationRecoveryRequired, 0, 0, err)
	}
	if !found || run.State == "SUCCEEDED" || run.State == "RESTORED" {
		return nil
	}
	if run.State == "RECOVERY_REQUIRED" {
		return newMigrationError(MigrationRecoveryRequired, run.SourceVersion, run.TargetVersion, errors.New("manual recovery is required"))
	}
	if run.State != "RUNNING" {
		return newMigrationError(MigrationRecoveryRequired, run.SourceVersion, run.TargetVersion, errors.New("unknown migration state"))
	}
	if err := restoreMigrationSource(dataDir, run); err != nil {
		return failRecoveryState(dataDir, run, err)
	}
	run.State, run.ErrorCode = "RESTORED", MigrationFailedRecovered
	now := time.Now().UTC()
	run.CompletedAt = &now
	if err := writeMigrationState(dataDir, run); err != nil {
		return failRecoveryState(dataDir, run, err)
	}
	return nil
}

func failRecoveryState(dataDir string, run migrationRun, cause error) error {
	run.State, run.ErrorCode = "RECOVERY_REQUIRED", MigrationRecoveryRequired
	now := time.Now().UTC()
	run.CompletedAt = &now
	_ = writeMigrationState(dataDir, run)
	return newMigrationError(MigrationRecoveryRequired, run.SourceVersion, run.TargetVersion, cause)
}

func restoreMigrationSource(dataDir string, run migrationRun) error {
	databasePath := filepath.Join(dataDir, databaseFilename)
	if run.SourceVersion == 0 {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if err := os.Remove(databasePath + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	}
	if !protectionNameRE.MatchString(run.ProtectionFile) || len(run.ProtectionSHA256) != 64 {
		return errors.New("migration protection identity is invalid")
	}
	protectionPath := filepath.Join(dataDir, protectionDirectory, run.ProtectionFile)
	digest, err := fileSHA256(protectionPath)
	if err != nil || digest != run.ProtectionSHA256 {
		return errors.New("migration protection checksum mismatch")
	}
	temp, err := os.CreateTemp(dataDir, ".yorva-db-restore-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	source, err := os.Open(protectionPath)
	if err != nil {
		_ = temp.Close()
		return err
	}
	_, copyErr := io.Copy(temp, source)
	closeSourceErr := source.Close()
	syncErr := temp.Sync()
	closeTempErr := temp.Close()
	if err := errors.Join(copyErr, closeSourceErr, syncErr, closeTempErr); err != nil {
		return err
	}
	if err := replaceFile(tempPath, databasePath); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(databasePath + suffix)
	}
	return nil
}

func newProtectionName(source, target int, now time.Time) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("schema-%03d-to-%03d-%s-%s.db", source, target, now.UTC().Format("20060102T150405000Z"), hex.EncodeToString(random)), nil
}

func fileSHA256(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("migration protection is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeMigrationState(dataDir string, run migrationRun) error {
	payload, err := json.Marshal(run)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(dataDir, ".migration-state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return replaceFile(tempPath, filepath.Join(dataDir, migrationStateFilename))
}

func readMigrationState(dataDir string) (migrationRun, bool, error) {
	path := filepath.Join(dataDir, migrationStateFilename)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return migrationRun{}, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxMigrationStateBytes {
		return migrationRun{}, false, errors.New("migration state file is invalid")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return migrationRun{}, false, err
	}
	var run migrationRun
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&run); err != nil || decoder.Decode(&struct{}{}) != io.EOF || run.SchemaVersion != 1 || run.SourceVersion < 0 || run.TargetVersion <= run.SourceVersion || run.StartedAt.IsZero() {
		return migrationRun{}, false, errors.New("migration state payload is invalid")
	}
	return run, true, nil
}

func retainProtections(dataDir string) error {
	root := filepath.Join(dataDir, protectionDirectory)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() && protectionNameRE.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) <= maxProtectionFiles {
		return nil
	}
	for _, name := range names[maxProtectionFiles:] {
		if err := os.Remove(filepath.Join(root, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
