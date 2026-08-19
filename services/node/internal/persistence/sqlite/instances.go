package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
)

type InstanceSnapshotEntry struct {
	NativeID string
	Default  bool
}

func NewInstanceID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate instance ID: %w", err)
	}
	return instance.NewID(base64.RawURLEncoding.EncodeToString(random)), nil
}

func (d *Database) GetAcceptedInstallationByID(ctx context.Context, id string) (AcceptedInstallation, error) {
	row := d.db.QueryRowContext(ctx, `
        SELECT id, node_id, runtime_kind, install_path, version, support_state, status,
               last_detected_at, created_at, updated_at
        FROM runtime_installations
        WHERE id = ?
    `, id)
	return scanAcceptedInstallation(row)
}

func (d *Database) GetAcceptedInstallationByKind(ctx context.Context, nodeID string, kind string) (AcceptedInstallation, error) {
	row := d.db.QueryRowContext(ctx, `
        SELECT id, node_id, runtime_kind, install_path, version, support_state, status,
               last_detected_at, created_at, updated_at
        FROM runtime_installations
        WHERE node_id = ? AND runtime_kind = ?
        ORDER BY updated_at DESC
        LIMIT 1
    `, nodeID, kind)
	return scanAcceptedInstallation(row)
}

func scanAcceptedInstallation(row *sql.Row) (AcceptedInstallation, error) {
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

func (d *Database) ApplyInstanceSnapshot(ctx context.Context, installationID string, present []InstanceSnapshotEntry, now time.Time) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin instance snapshot: %w", err)
	}
	defer tx.Rollback()

	existing, err := listInstancesTx(ctx, tx, installationID)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(present))
	stamp := formatTime(now)
	for _, entry := range present {
		if entry.NativeID == "" {
			return errors.New("instance snapshot contains an empty native identity")
		}
		if _, dup := seen[entry.NativeID]; dup {
			return errors.New("instance snapshot contains a duplicate native identity")
		}
		seen[entry.NativeID] = struct{}{}
		row, ok := existing[entry.NativeID]
		if !ok {
			id, err := NewInstanceID()
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
                INSERT INTO instances(
                    id, runtime_installation_id, native_id, name, is_default, is_protected,
                    availability, last_synced_at, created_at, updated_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            `, id, installationID, entry.NativeID, entry.NativeID, boolToInt(entry.Default), boolToInt(entry.Default),
				string(instance.Available), stamp, stamp, stamp); err != nil {
				return fmt.Errorf("insert instance: %w", err)
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE instances
            SET name = ?, is_default = ?, is_protected = ?, availability = ?, last_synced_at = ?, updated_at = ?
            WHERE id = ?
        `, entry.NativeID, boolToInt(entry.Default), boolToInt(entry.Default), string(instance.Available), stamp, stamp, row.ID); err != nil {
			return fmt.Errorf("update instance: %w", err)
		}
	}
	for nativeID, row := range existing {
		if _, still := seen[nativeID]; still {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
            UPDATE instances
            SET availability = ?, last_synced_at = ?, updated_at = ?
            WHERE id = ?
        `, string(instance.Missing), stamp, stamp, row.ID); err != nil {
			return fmt.Errorf("mark instance missing: %w", err)
		}
	}
	return tx.Commit()
}

func (d *Database) MarkInstancesUnknown(ctx context.Context, installationID string, now time.Time) error {
	_, err := d.db.ExecContext(ctx, `
        UPDATE instances
        SET availability = ?, updated_at = ?
        WHERE runtime_installation_id = ?
    `, string(instance.Unknown), formatTime(now), installationID)
	if err != nil {
		return fmt.Errorf("mark instances unknown: %w", err)
	}
	return nil
}

func (d *Database) ListInstances(ctx context.Context, installationID string) ([]instance.Instance, error) {
	rows, err := d.db.QueryContext(ctx, `
        SELECT id, runtime_installation_id, native_id, name, is_default, is_protected,
               availability, last_synced_at, created_at, updated_at
        FROM instances
        WHERE runtime_installation_id = ?
        ORDER BY is_default DESC, native_id ASC
    `, installationID)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer rows.Close()
	var out []instance.Instance
	for rows.Next() {
		row, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d *Database) GetInstance(ctx context.Context, id string) (instance.Instance, error) {
	row := d.db.QueryRowContext(ctx, `
        SELECT id, runtime_installation_id, native_id, name, is_default, is_protected,
               availability, last_synced_at, created_at, updated_at
        FROM instances
        WHERE id = ?
    `, id)
	value, err := scanInstance(row)
	if errors.Is(err, sql.ErrNoRows) {
		return instance.Instance{}, err
	}
	return value, err
}

func listInstancesTx(ctx context.Context, tx *sql.Tx, installationID string) (map[string]instance.Instance, error) {
	rows, err := tx.QueryContext(ctx, `
        SELECT id, runtime_installation_id, native_id, name, is_default, is_protected,
               availability, last_synced_at, created_at, updated_at
        FROM instances
        WHERE runtime_installation_id = ?
    `, installationID)
	if err != nil {
		return nil, fmt.Errorf("list instances for snapshot: %w", err)
	}
	defer rows.Close()
	out := make(map[string]instance.Instance)
	for rows.Next() {
		row, err := scanInstance(rows)
		if err != nil {
			return nil, err
		}
		out[row.NativeID] = row
	}
	return out, rows.Err()
}

type instanceScanner interface {
	Scan(dest ...any) error
}

func scanInstance(row instanceScanner) (instance.Instance, error) {
	var value instance.Instance
	var lastSynced sql.NullString
	var createdAt, updatedAt string
	var isDefault, isProtected int
	if err := row.Scan(
		&value.ID, &value.RuntimeInstallationID, &value.NativeID, &value.Name,
		&isDefault, &isProtected, &value.Availability, &lastSynced, &createdAt, &updatedAt,
	); err != nil {
		return instance.Instance{}, err
	}
	value.Default = isDefault == 1
	value.Protected = isProtected == 1
	if lastSynced.Valid {
		parsed, err := parseTime(lastSynced.String)
		if err != nil {
			return instance.Instance{}, err
		}
		value.LastSyncedAt = &parsed
	}
	var err error
	value.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return instance.Instance{}, err
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return instance.Instance{}, err
	}
	return value, nil
}
