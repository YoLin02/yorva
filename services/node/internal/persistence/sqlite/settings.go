package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (d *Database) GetAppSetting(ctx context.Context, key string) ([]byte, bool, error) {
	var value string
	err := d.db.QueryRowContext(ctx, "SELECT value_json FROM app_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get app setting: %w", err)
	}
	return []byte(value), true, nil
}

func (d *Database) PutAppSetting(ctx context.Context, key string, value []byte, updatedAt time.Time) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO app_settings(key, value_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at
	`, key, string(value), updatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("put app setting: %w", err)
	}
	return nil
}
func (d *Database) DeleteAppSetting(ctx context.Context, key string) error {
	if _, err := d.db.ExecContext(ctx, "DELETE FROM app_settings WHERE key = ?", key); err != nil {
		return fmt.Errorf("delete app setting: %w", err)
	}
	return nil
}
