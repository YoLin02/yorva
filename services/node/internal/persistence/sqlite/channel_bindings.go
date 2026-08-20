package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
)

func NewChannelBindingID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate channel binding ID: %w", err)
	}
	return "chb_" + base64.RawURLEncoding.EncodeToString(random), nil
}

func (d *Database) ListChannelBindings(ctx context.Context, instanceID string) ([]channel.Binding, error) {
	rows, err := d.db.QueryContext(ctx, channelBindingSelect+" WHERE instance_id = ? ORDER BY channel_type", instanceID)
	if err != nil {
		return nil, fmt.Errorf("list channel bindings: %w", err)
	}
	defer rows.Close()
	var out []channel.Binding
	for rows.Next() {
		value, scanErr := scanChannelBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (d *Database) GetChannelBinding(ctx context.Context, instanceID string, kind channel.Type) (channel.Binding, bool, error) {
	value, err := scanChannelBinding(d.db.QueryRowContext(ctx, channelBindingSelect+" WHERE instance_id = ? AND channel_type = ?", instanceID, string(kind)))
	if errors.Is(err, sql.ErrNoRows) {
		return channel.Binding{}, false, nil
	}
	return value, err == nil, err
}

func (d *Database) UpsertChannelBinding(ctx context.Context, value channel.Binding) error {
	if value.ID == "" || value.InstanceID == "" || !channel.ValidType(value.Type) || !channel.ValidState(value.State) {
		return errors.New("invalid channel binding")
	}
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO channel_bindings(
            id, instance_id, channel_type, state, account_label, external_id,
            metadata_json, last_checked_at, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, '{}', ?, ?, ?)
        ON CONFLICT(instance_id, channel_type) DO UPDATE SET
            state = excluded.state,
            account_label = excluded.account_label,
            external_id = excluded.external_id,
            metadata_json = '{}',
            last_checked_at = excluded.last_checked_at,
            updated_at = excluded.updated_at
    `, value.ID, value.InstanceID, string(value.Type), string(value.State), value.AccountLabel, value.ExternalID,
		formatOptionalTime(value.LastCheckedAt), formatTime(value.CreatedAt), formatTime(value.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert channel binding: %w", err)
	}
	return nil
}

const channelBindingSelect = `
    SELECT id, instance_id, channel_type, state, account_label, external_id,
           last_checked_at, created_at, updated_at
    FROM channel_bindings`

type channelBindingScanner interface {
	Scan(...any) error
}

func scanChannelBinding(row channelBindingScanner) (channel.Binding, error) {
	var value channel.Binding
	var checked sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&value.ID, &value.InstanceID, &value.Type, &value.State, &value.AccountLabel, &value.ExternalID, &checked, &createdAt, &updatedAt); err != nil {
		return channel.Binding{}, err
	}
	if checked.Valid {
		parsed, err := parseTime(checked.String)
		if err != nil {
			return channel.Binding{}, err
		}
		value.LastCheckedAt = &parsed
	}
	var err error
	value.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return channel.Binding{}, err
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	return value, err
}

func channelBindingAt(instanceID string, kind channel.Type, state channel.State, now time.Time) (channel.Binding, error) {
	id, err := NewChannelBindingID()
	if err != nil {
		return channel.Binding{}, err
	}
	return channel.Binding{ID: id, InstanceID: instanceID, Type: kind, State: state, CreatedAt: now, UpdatedAt: now}, nil
}
