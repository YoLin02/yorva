package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type AcceptedInstallation struct {
	ID             string
	NodeID         string
	RuntimeKind    yorvaruntime.Kind
	InstallPath    string
	Version        string
	SupportState   yorvaruntime.DiscoveryState
	Status         string
	LastDetectedAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewInstallationID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate installation ID: %w", err)
	}
	return "rtinst_" + base64.RawURLEncoding.EncodeToString(random), nil
}

func (d *Database) UpsertAcceptedInstallation(ctx context.Context, value AcceptedInstallation) error {
	if value.InstallPath == "" || value.Version == "" {
		return errors.New("accepted installation requires path and version")
	}
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO runtime_installations(
            id, node_id, runtime_kind, install_path, version, support_state, status,
            metadata_json, last_detected_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)
        ON CONFLICT(node_id, runtime_kind, install_path) DO UPDATE SET
            version = excluded.version,
            support_state = excluded.support_state,
            status = excluded.status,
            last_detected_at = excluded.last_detected_at,
            updated_at = excluded.updated_at
    `, value.ID, value.NodeID, string(value.RuntimeKind), value.InstallPath, value.Version,
		string(value.SupportState), value.Status, formatTime(value.LastDetectedAt),
		formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert accepted installation: %w", err)
	}
	return nil
}

func (d *Database) GetAcceptedInstallation(ctx context.Context, nodeID string, kind yorvaruntime.Kind, path string) (AcceptedInstallation, error) {
	row := d.db.QueryRowContext(ctx, `
        SELECT id, node_id, runtime_kind, install_path, version, support_state, status,
               last_detected_at, created_at, updated_at
        FROM runtime_installations
        WHERE node_id = ? AND runtime_kind = ? AND install_path = ?
    `, nodeID, string(kind), path)
	var value AcceptedInstallation
	var lastDetectedAt, createdAt, updatedAt string
	if err := row.Scan(
		&value.ID, &value.NodeID, &value.RuntimeKind, &value.InstallPath, &value.Version,
		&value.SupportState, &value.Status, &lastDetectedAt, &createdAt, &updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AcceptedInstallation{}, err
		}
		return AcceptedInstallation{}, fmt.Errorf("scan accepted installation: %w", err)
	}
	var err error
	value.LastDetectedAt, err = parseTime(lastDetectedAt)
	if err != nil {
		return AcceptedInstallation{}, err
	}
	value.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return AcceptedInstallation{}, err
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return AcceptedInstallation{}, err
	}
	return value, nil
}

func (d *Database) CompleteInstallSuccess(ctx context.Context, current, next operation.Operation, installation AcceptedInstallation) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin install success transaction: %w", err)
	}
	defer tx.Rollback()
	if current.Status != next.Status && !operation.ValidStatusChange(current.Status, next.Status) {
		return ErrInvalidStatusTransition
	}
	result, err := tx.ExecContext(ctx, `
        UPDATE operations
        SET status = ?, stage = ?, progress = NULL, message = ?, error_code = ?, error_message = ?,
            retryable = ?, started_at = ?, completed_at = ?, updated_at = ?
        WHERE id = ? AND status = ?
    `, string(next.Status), string(next.Stage), next.Message, string(next.ErrorCode), next.ErrorMessage,
		boolToInt(next.Retryable), formatOptionalTime(next.StartedAt), formatOptionalTime(next.CompletedAt),
		formatTime(next.UpdatedAt), next.ID, string(current.Status))
	if err != nil {
		return fmt.Errorf("complete operation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrInvalidStatusTransition
	}
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO runtime_installations(
            id, node_id, runtime_kind, install_path, version, support_state, status,
            metadata_json, last_detected_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)
        ON CONFLICT(node_id, runtime_kind, install_path) DO UPDATE SET
            version = excluded.version,
            support_state = excluded.support_state,
            status = excluded.status,
            last_detected_at = excluded.last_detected_at,
            updated_at = excluded.updated_at
    `, installation.ID, installation.NodeID, string(installation.RuntimeKind), installation.InstallPath,
		installation.Version, string(installation.SupportState), installation.Status,
		formatTime(installation.LastDetectedAt), formatTime(installation.CreatedAt), formatTime(installation.UpdatedAt)); err != nil {
		return fmt.Errorf("persist accepted installation: %w", err)
	}
	return tx.Commit()
}

func (d *Database) CountAcceptedInstallations(ctx context.Context) (int, error) {
	var count int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM runtime_installations").Scan(&count); err != nil {
		return 0, fmt.Errorf("count accepted installations: %w", err)
	}
	return count, nil
}
