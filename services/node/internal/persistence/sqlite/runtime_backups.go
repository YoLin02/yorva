package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type RuntimeBackupIndexState string

const (
	RuntimeBackupAvailable     RuntimeBackupIndexState = "AVAILABLE"
	RuntimeBackupMissing       RuntimeBackupIndexState = "MISSING"
	RuntimeBackupChanged       RuntimeBackupIndexState = "CHANGED"
	RuntimeBackupUndecryptable RuntimeBackupIndexState = "UNDECRYPTABLE"
	RuntimeBackupMalformed     RuntimeBackupIndexState = "MALFORMED"
	RuntimeBackupUnknown       RuntimeBackupIndexState = "UNKNOWN"
)

func (s RuntimeBackupIndexState) valid() bool {
	switch s {
	case RuntimeBackupAvailable, RuntimeBackupMissing, RuntimeBackupChanged,
		RuntimeBackupUndecryptable, RuntimeBackupMalformed, RuntimeBackupUnknown:
		return true
	default:
		return false
	}
}

type RuntimeBackupKeyMode string

const (
	RuntimeBackupDeviceKey  RuntimeBackupKeyMode = "DEVICE"
	RuntimeBackupPassphrase RuntimeBackupKeyMode = "PASSPHRASE"
)

// RuntimeBackupIndexEntry is YORVA-owned safe inventory metadata for one
// verified Runtime-scoped encrypted artifact. ArtifactPath and KeyRef are
// persistence-only values and must not be projected through ordinary read APIs.
type RuntimeBackupIndexEntry struct {
	ID                    string
	RuntimeInstallationID string
	FormatVersion         string
	RuntimeVersion        string
	ArtifactPath          string
	SizeBytes             int64
	ChecksumSHA256        string
	State                 RuntimeBackupIndexState
	KeyMode               RuntimeBackupKeyMode
	KeyRef                string
	CreatedAt             time.Time
	VerifiedAt            time.Time
	UpdatedAt             time.Time
}

// InsertVerifiedRuntimeBackup inserts a new immutable artifact identity only
// after the caller has verified the final encrypted file. Reconciliation may
// subsequently change only State and the verification/update timestamps.
func (d *Database) InsertVerifiedRuntimeBackup(ctx context.Context, value RuntimeBackupIndexEntry) error {
	if err := validateVerifiedRuntimeBackup(value); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO runtime_backups(
            id, runtime_installation_id, scope_type, format_version, runtime_version,
            artifact_path, size_bytes, checksum_sha256, state, key_mode, key_ref,
            created_at, verified_at, updated_at
        ) VALUES (?, ?, 'RUNTIME', ?, ?, ?, ?, ?, 'AVAILABLE', ?, ?, ?, ?, ?)
    `, value.ID, value.RuntimeInstallationID, value.FormatVersion, value.RuntimeVersion,
		value.ArtifactPath, value.SizeBytes, value.ChecksumSHA256, string(value.KeyMode),
		nullableKeyRef(value), formatTime(value.CreatedAt), formatTime(value.VerifiedAt),
		formatTime(value.UpdatedAt))
	if err != nil {
		return fmt.Errorf("insert verified Runtime backup: %w", err)
	}
	return nil
}

func (d *Database) GetRuntimeBackup(ctx context.Context, runtimeInstallationID, backupID string) (RuntimeBackupIndexEntry, error) {
	row := d.db.QueryRowContext(ctx, runtimeBackupSelect+`
        WHERE runtime_installation_id = ? AND id = ?
    `, runtimeInstallationID, backupID)
	return scanRuntimeBackup(row)
}

func (d *Database) ListRuntimeBackups(ctx context.Context, runtimeInstallationID string) ([]RuntimeBackupIndexEntry, error) {
	rows, err := d.db.QueryContext(ctx, runtimeBackupSelect+`
        WHERE runtime_installation_id = ?
        ORDER BY created_at DESC, id ASC
    `, runtimeInstallationID)
	if err != nil {
		return nil, fmt.Errorf("list Runtime backups: %w", err)
	}
	defer rows.Close()

	var out []RuntimeBackupIndexEntry
	for rows.Next() {
		value, err := scanRuntimeBackup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list Runtime backups: %w", err)
	}
	return out, nil
}

// ReconcileRuntimeBackupState records observed artifact truth without changing
// its immutable path, checksum, format, Runtime version, or key authority.
// AVAILABLE is accepted only after a fresh successful verification timestamp.
func (d *Database) ReconcileRuntimeBackupState(
	ctx context.Context,
	runtimeInstallationID string,
	backupID string,
	state RuntimeBackupIndexState,
	verifiedAt *time.Time,
	updatedAt time.Time,
) error {
	if runtimeInstallationID == "" || backupID == "" || !state.valid() || updatedAt.IsZero() {
		return errors.New("invalid Runtime backup reconciliation")
	}
	if state == RuntimeBackupAvailable && (verifiedAt == nil || verifiedAt.IsZero()) {
		return errors.New("available Runtime backup requires fresh verification")
	}
	if state != RuntimeBackupAvailable && verifiedAt != nil {
		return errors.New("non-available Runtime backup cannot claim fresh verification")
	}

	var result sql.Result
	var err error
	if state == RuntimeBackupAvailable {
		result, err = d.db.ExecContext(ctx, `
            UPDATE runtime_backups
            SET state = ?, verified_at = ?, updated_at = ?
            WHERE runtime_installation_id = ? AND id = ?
        `, string(state), formatTime(*verifiedAt), formatTime(updatedAt), runtimeInstallationID, backupID)
	} else {
		result, err = d.db.ExecContext(ctx, `
            UPDATE runtime_backups
            SET state = ?, updated_at = ?
            WHERE runtime_installation_id = ? AND id = ?
        `, string(state), formatTime(updatedAt), runtimeInstallationID, backupID)
	}
	if err != nil {
		return fmt.Errorf("reconcile Runtime backup: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count reconciled Runtime backups: %w", err)
	}
	if affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func validateVerifiedRuntimeBackup(value RuntimeBackupIndexEntry) error {
	if value.ID == "" || len(value.ID) > 256 || value.RuntimeInstallationID == "" || len(value.RuntimeInstallationID) > 256 {
		return errors.New("verified Runtime backup requires bounded identities")
	}
	if !boundedMetadata(value.FormatVersion, 64) || !boundedMetadata(value.RuntimeVersion, 64) {
		return errors.New("verified Runtime backup requires bounded versions")
	}
	if value.ArtifactPath == "" || len(value.ArtifactPath) > 32767 || strings.IndexByte(value.ArtifactPath, 0) >= 0 {
		return errors.New("verified Runtime backup requires a bounded artifact path")
	}
	if value.SizeBytes <= 0 || !validLowerSHA256(value.ChecksumSHA256) {
		return errors.New("verified Runtime backup requires size and SHA-256")
	}
	if value.State != RuntimeBackupAvailable {
		return errors.New("only verified available Runtime backups may be inserted")
	}
	switch value.KeyMode {
	case RuntimeBackupDeviceKey:
		if value.KeyRef == "" || len(value.KeyRef) > 256 || strings.IndexByte(value.KeyRef, 0) >= 0 {
			return errors.New("device-managed Runtime backup requires a bounded key reference")
		}
	case RuntimeBackupPassphrase:
		if value.KeyRef != "" {
			return errors.New("portable Runtime backup must not persist a key reference")
		}
	default:
		return errors.New("verified Runtime backup requires an approved key mode")
	}
	if value.CreatedAt.IsZero() || value.VerifiedAt.IsZero() || value.UpdatedAt.IsZero() {
		return errors.New("verified Runtime backup requires timestamps")
	}
	return nil
}

func boundedMetadata(value string, max int) bool {
	return value != "" && len(value) <= max && strings.TrimSpace(value) == value && strings.IndexByte(value, 0) < 0
}

func validLowerSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func nullableKeyRef(value RuntimeBackupIndexEntry) any {
	if value.KeyMode == RuntimeBackupPassphrase {
		return nil
	}
	return value.KeyRef
}

const runtimeBackupSelect = `
    SELECT id, runtime_installation_id, format_version, runtime_version,
           artifact_path, size_bytes, checksum_sha256, state, key_mode, key_ref,
           created_at, verified_at, updated_at
    FROM runtime_backups`

type runtimeBackupScanner interface {
	Scan(...any) error
}

func scanRuntimeBackup(row runtimeBackupScanner) (RuntimeBackupIndexEntry, error) {
	var value RuntimeBackupIndexEntry
	var keyRef sql.NullString
	var createdAt, verifiedAt, updatedAt string
	if err := row.Scan(
		&value.ID, &value.RuntimeInstallationID, &value.FormatVersion, &value.RuntimeVersion,
		&value.ArtifactPath, &value.SizeBytes, &value.ChecksumSHA256, &value.State,
		&value.KeyMode, &keyRef, &createdAt, &verifiedAt, &updatedAt,
	); err != nil {
		return RuntimeBackupIndexEntry{}, err
	}
	if keyRef.Valid {
		value.KeyRef = keyRef.String
	}
	var err error
	value.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return RuntimeBackupIndexEntry{}, err
	}
	value.VerifiedAt, err = parseTime(verifiedAt)
	if err != nil {
		return RuntimeBackupIndexEntry{}, err
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return RuntimeBackupIndexEntry{}, err
	}
	return value, nil
}
