package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
)

func (d *Database) LoadOrCreateNode(ctx context.Context, metadata node.LocalMetadata) (node.Node, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return node.Node{}, fmt.Errorf("begin local node transaction: %w", err)
	}
	defer tx.Rollback()

	current, err := selectLocalNode(ctx, tx)
	if errors.Is(err, sql.ErrNoRows) {
		current, err = newNode(metadata)
		if err != nil {
			return node.Node{}, err
		}
		_, err = tx.ExecContext(ctx, `
            INSERT INTO nodes(id, name, hostname, platform, architecture, node_version, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        `, current.ID, current.Name, current.Hostname, current.Platform, current.Architecture, current.NodeVersion, formatTime(current.CreatedAt), formatTime(current.UpdatedAt))
		if err != nil {
			return node.Node{}, fmt.Errorf("insert local node: %w", err)
		}
	} else if err != nil {
		return node.Node{}, err
	} else {
		current.Name = metadata.Name
		current.Hostname = metadata.Hostname
		current.Platform = metadata.Platform
		current.Architecture = metadata.Architecture
		current.NodeVersion = metadata.NodeVersion
		current.UpdatedAt = time.Now().UTC()
		_, err = tx.ExecContext(ctx, `
            UPDATE nodes
            SET name = ?, hostname = ?, platform = ?, architecture = ?, node_version = ?, updated_at = ?
            WHERE id = ?
        `, current.Name, current.Hostname, current.Platform, current.Architecture, current.NodeVersion, formatTime(current.UpdatedAt), current.ID)
		if err != nil {
			return node.Node{}, fmt.Errorf("refresh local node: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return node.Node{}, fmt.Errorf("commit local node transaction: %w", err)
	}
	return current, nil
}

func selectLocalNode(ctx context.Context, tx *sql.Tx) (node.Node, error) {
	rows, err := tx.QueryContext(ctx, `
        SELECT id, name, hostname, platform, architecture, node_version, created_at, updated_at
        FROM nodes
        ORDER BY created_at
        LIMIT 2
    `)
	if err != nil {
		return node.Node{}, fmt.Errorf("query local node: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return node.Node{}, fmt.Errorf("iterate local node: %w", err)
		}
		return node.Node{}, sql.ErrNoRows
	}

	var result node.Node
	var createdAt, updatedAt string
	if err := rows.Scan(&result.ID, &result.Name, &result.Hostname, &result.Platform, &result.Architecture, &result.NodeVersion, &createdAt, &updatedAt); err != nil {
		return node.Node{}, fmt.Errorf("scan local node: %w", err)
	}
	if rows.Next() {
		return node.Node{}, errors.New("multiple local node identities found")
	}
	result.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return node.Node{}, fmt.Errorf("parse local node created_at: %w", err)
	}
	result.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return node.Node{}, fmt.Errorf("parse local node updated_at: %w", err)
	}
	return result, nil
}

func newNode(metadata node.LocalMetadata) (node.Node, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return node.Node{}, fmt.Errorf("generate local node ID: %w", err)
	}
	now := time.Now().UTC()
	return node.Node{
		ID:           "node_" + base64.RawURLEncoding.EncodeToString(random),
		Name:         metadata.Name,
		Hostname:     metadata.Hostname,
		Platform:     metadata.Platform,
		Architecture: metadata.Architecture,
		NodeVersion:  metadata.NodeVersion,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
