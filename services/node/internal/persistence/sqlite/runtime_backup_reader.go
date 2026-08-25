package sqlite

import (
	"context"
	"database/sql"
	"errors"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// RuntimeBackupReader binds the Runtime-neutral read contract to one accepted
// Runtime installation's safe index. Reads do not access, decrypt, hash, or
// mutate the indexed artifact; State and VerifiedAt are last-observed facts.
type RuntimeBackupReader struct {
	database              *Database
	runtimeInstallationID string
}

func NewRuntimeBackupReader(database *Database, runtimeInstallationID string) yorvaruntime.BackupReader {
	if database == nil || runtimeInstallationID == "" {
		return nil
	}
	return &RuntimeBackupReader{database: database, runtimeInstallationID: runtimeInstallationID}
}

func (r *RuntimeBackupReader) ListBackups(ctx context.Context, installation yorvaruntime.Installation) ([]yorvaruntime.Backup, error) {
	if err := r.validate(installation); err != nil {
		return nil, err
	}
	rows, err := r.database.ListRuntimeBackups(ctx, r.runtimeInstallationID)
	if err != nil {
		return nil, err
	}
	backups := make([]yorvaruntime.Backup, 0, len(rows))
	for _, row := range rows {
		backup, err := projectRuntimeBackup(row)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	return backups, nil
}

func (r *RuntimeBackupReader) GetBackup(ctx context.Context, installation yorvaruntime.Installation, backupID string) (yorvaruntime.Backup, error) {
	if err := r.validate(installation); err != nil {
		return yorvaruntime.Backup{}, err
	}
	row, err := r.database.GetRuntimeBackup(ctx, r.runtimeInstallationID, backupID)
	if errors.Is(err, sql.ErrNoRows) {
		return yorvaruntime.Backup{}, yorvaruntime.ErrBackupNotFound
	}
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	return projectRuntimeBackup(row)
}

func (r *RuntimeBackupReader) validate(installation yorvaruntime.Installation) error {
	if r == nil || r.database == nil || r.runtimeInstallationID == "" ||
		installation.RuntimeKind == "" || installation.Path == "" || installation.Version == "" ||
		installation.SupportState != yorvaruntime.DiscoverySupported {
		return yorvaruntime.ErrInvalidManagementContract
	}
	return nil
}

func projectRuntimeBackup(row RuntimeBackupIndexEntry) (yorvaruntime.Backup, error) {
	backup := yorvaruntime.Backup{
		ID:             row.ID,
		State:          yorvaruntime.BackupState(row.State),
		FormatVersion:  row.FormatVersion,
		RuntimeVersion: row.RuntimeVersion,
		SizeBytes:      row.SizeBytes,
		ChecksumSHA256: row.ChecksumSHA256,
		CreatedAt:      row.CreatedAt,
		VerifiedAt:     row.VerifiedAt,
		KeyMode:        yorvaruntime.BackupKeyMode(row.KeyMode),
	}
	if err := backup.Validate(); err != nil || !backup.State.Indexed() {
		return yorvaruntime.Backup{}, yorvaruntime.ErrInvalidManagementContract
	}
	return backup, nil
}
