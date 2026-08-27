package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type ModelProviderConnection struct {
	ID                    string
	RuntimeInstallationID string
	ProviderPresetID      string
	DisplayName           string
	SecretRef             string
	Status                string
	Revision              int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ModelProfile struct {
	ID                    string
	RuntimeInstallationID string
	ProviderConnectionID  string
	DisplayName           string
	SelectedModelIDs      []string
	DefaultModelID        string
	Revision              int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type RuntimeModelDefault struct {
	RuntimeInstallationID string
	ModelProfileID        string
	AppliedRevision       int
	UpdatedAt             time.Time
}

type InstanceModelBinding struct {
	InstanceID      string
	ModelProfileID  string
	Mode            string
	AppliedRevision int
	State           string
	ErrorCode       string
	UpdatedAt       time.Time
}

func (d *Database) InsertModelProviderConnection(ctx context.Context, value ModelProviderConnection) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO model_provider_connections(
        id, runtime_installation_id, provider_preset_id, display_name, secret_ref, status, revision, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.RuntimeInstallationID, value.ProviderPresetID,
		value.DisplayName, value.SecretRef, value.Status, value.Revision, formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
	return err
}

func (d *Database) ListModelProviderConnections(ctx context.Context, installationID string) ([]ModelProviderConnection, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT id, runtime_installation_id, provider_preset_id, display_name,
        secret_ref, status, revision, created_at, updated_at FROM model_provider_connections
        WHERE runtime_installation_id = ? ORDER BY created_at, id`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ModelProviderConnection, 0)
	for rows.Next() {
		value, err := scanModelProviderConnection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (d *Database) GetModelProviderConnection(ctx context.Context, id string) (ModelProviderConnection, error) {
	return scanModelProviderConnection(d.db.QueryRowContext(ctx, `SELECT id, runtime_installation_id, provider_preset_id,
        display_name, secret_ref, status, revision, created_at, updated_at FROM model_provider_connections WHERE id = ?`, id))
}

func (d *Database) DeleteModelProviderConnection(ctx context.Context, id string) error {
	result, err := d.db.ExecContext(ctx, `DELETE FROM model_provider_connections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *Database) InsertModelProfile(ctx context.Context, value ModelProfile) error {
	selected, err := json.Marshal(value.SelectedModelIDs)
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, `INSERT INTO model_profiles(
        id, runtime_installation_id, provider_connection_id, display_name, selected_model_ids_json,
        default_model_id, revision, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, value.ID, value.RuntimeInstallationID, value.ProviderConnectionID,
		value.DisplayName, string(selected), value.DefaultModelID, value.Revision, formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
	return err
}

func (d *Database) ListModelProfiles(ctx context.Context, installationID string) ([]ModelProfile, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT id, runtime_installation_id, provider_connection_id, display_name,
        selected_model_ids_json, default_model_id, revision, created_at, updated_at FROM model_profiles
        WHERE runtime_installation_id = ? ORDER BY created_at, id`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ModelProfile, 0)
	for rows.Next() {
		value, err := scanModelProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (d *Database) GetModelProfile(ctx context.Context, id string) (ModelProfile, error) {
	return scanModelProfile(d.db.QueryRowContext(ctx, `SELECT id, runtime_installation_id, provider_connection_id,
        display_name, selected_model_ids_json, default_model_id, revision, created_at, updated_at FROM model_profiles WHERE id = ?`, id))
}

func (d *Database) DeleteModelProfile(ctx context.Context, id string) error {
	result, err := d.db.ExecContext(ctx, `DELETE FROM model_profiles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *Database) PutRuntimeModelDefault(ctx context.Context, value RuntimeModelDefault) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO runtime_model_defaults(runtime_installation_id, model_profile_id, applied_revision, updated_at)
        VALUES (?, ?, ?, ?) ON CONFLICT(runtime_installation_id) DO UPDATE SET
        model_profile_id = excluded.model_profile_id, applied_revision = excluded.applied_revision, updated_at = excluded.updated_at`,
		value.RuntimeInstallationID, value.ModelProfileID, value.AppliedRevision, formatTime(value.UpdatedAt))
	return err
}

func (d *Database) GetRuntimeModelDefault(ctx context.Context, installationID string) (RuntimeModelDefault, bool, error) {
	var value RuntimeModelDefault
	var updatedAt string
	err := d.db.QueryRowContext(ctx, `SELECT runtime_installation_id, model_profile_id, applied_revision, updated_at
        FROM runtime_model_defaults WHERE runtime_installation_id = ?`, installationID).Scan(
		&value.RuntimeInstallationID, &value.ModelProfileID, &value.AppliedRevision, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return RuntimeModelDefault{}, false, nil
	}
	if err != nil {
		return RuntimeModelDefault{}, false, err
	}
	value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	return value, err == nil, err
}

func (d *Database) DeleteRuntimeModelDefault(ctx context.Context, installationID string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM runtime_model_defaults WHERE runtime_installation_id = ?`, installationID)
	return err
}

func (d *Database) UpsertInstanceModelBinding(ctx context.Context, value InstanceModelBinding) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO instance_model_bindings(
        instance_id, model_profile_id, mode, applied_revision, state, error_code, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(instance_id) DO UPDATE SET
        model_profile_id = excluded.model_profile_id, mode = excluded.mode, applied_revision = excluded.applied_revision,
        state = excluded.state, error_code = excluded.error_code, updated_at = excluded.updated_at`,
		value.InstanceID, value.ModelProfileID, value.Mode, value.AppliedRevision, value.State, value.ErrorCode, formatTime(value.UpdatedAt))
	return err
}

func (d *Database) ListInstanceModelBindings(ctx context.Context, installationID string) ([]InstanceModelBinding, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT b.instance_id, b.model_profile_id, b.mode, b.applied_revision,
        b.state, b.error_code, b.updated_at FROM instance_model_bindings b
        JOIN instances i ON i.id = b.instance_id WHERE i.runtime_installation_id = ? ORDER BY i.native_id`, installationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]InstanceModelBinding, 0)
	for rows.Next() {
		var value InstanceModelBinding
		var updatedAt string
		if err := rows.Scan(&value.InstanceID, &value.ModelProfileID, &value.Mode, &value.AppliedRevision, &value.State, &value.ErrorCode, &updatedAt); err != nil {
			return nil, err
		}
		value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanModelProviderConnection(row rowScanner) (ModelProviderConnection, error) {
	var value ModelProviderConnection
	var createdAt, updatedAt string
	err := row.Scan(&value.ID, &value.RuntimeInstallationID, &value.ProviderPresetID, &value.DisplayName,
		&value.SecretRef, &value.Status, &value.Revision, &createdAt, &updatedAt)
	if err != nil {
		return ModelProviderConnection{}, err
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err == nil {
		value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	}
	return value, err
}

func scanModelProfile(row rowScanner) (ModelProfile, error) {
	var value ModelProfile
	var selected, createdAt, updatedAt string
	err := row.Scan(&value.ID, &value.RuntimeInstallationID, &value.ProviderConnectionID, &value.DisplayName,
		&selected, &value.DefaultModelID, &value.Revision, &createdAt, &updatedAt)
	if err != nil {
		return ModelProfile{}, err
	}
	if err = json.Unmarshal([]byte(selected), &value.SelectedModelIDs); err != nil {
		return ModelProfile{}, err
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err == nil {
		value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	}
	return value, err
}
