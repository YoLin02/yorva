package sqlite

import (
	"context"
	"time"
)

type ManagedMCPBinding struct {
	InstanceID string
	ServerID   string
	PresetID   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (d *Database) UpsertManagedMCPBinding(ctx context.Context, binding ManagedMCPBinding) error {
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO managed_mcp_bindings(instance_id, server_id, preset_id, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(instance_id, server_id) DO UPDATE SET preset_id = excluded.preset_id, updated_at = excluded.updated_at`,
		binding.InstanceID, binding.ServerID, binding.PresetID,
		binding.CreatedAt.UTC().Format(time.RFC3339Nano), binding.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (d *Database) ListManagedMCPBindings(ctx context.Context, instanceID string) ([]ManagedMCPBinding, error) {
	rows, err := d.db.QueryContext(ctx, `
        SELECT instance_id, server_id, preset_id, created_at, updated_at
        FROM managed_mcp_bindings WHERE instance_id = ? ORDER BY server_id`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ManagedMCPBinding, 0)
	for rows.Next() {
		var item ManagedMCPBinding
		var createdAt, updatedAt string
		if err := rows.Scan(&item.InstanceID, &item.ServerID, &item.PresetID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *Database) DeleteManagedMCPBinding(ctx context.Context, instanceID, serverID string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM managed_mcp_bindings WHERE instance_id = ? AND server_id = ?`, instanceID, serverID)
	return err
}
