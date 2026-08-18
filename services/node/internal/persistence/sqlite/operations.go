package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrOperationNotFound       = errors.New("operation not found")
	ErrDuplicateIdempotency    = errors.New("idempotency key already exists")
	ErrActiveInstallExists     = errors.New("an active Runtime install already exists")
	ErrInvalidStatusTransition = errors.New("invalid operation status transition")
)

type OperationListFilter struct {
	TargetType operation.TargetType
	TargetID   string
	Limit      int
}

func NewOperationID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate operation ID: %w", err)
	}
	return "op_" + base64.RawURLEncoding.EncodeToString(random), nil
}

func (d *Database) CreateOperation(ctx context.Context, value operation.Operation) error {
	if value.Progress != nil {
		return errors.New("operation progress must remain unset")
	}
	if value.IdempotencyKey == "" {
		return errors.New("operation idempotency key is required")
	}
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO operations(
            id, operation_type, target_type, target_id, status, stage, progress, message,
            error_code, error_message, retryable, idempotency_key, correlation_id, source_pin,
            created_at, started_at, completed_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, value.ID, string(value.Type), string(value.TargetType), value.TargetID, string(value.Status),
		string(value.Stage), value.Message, string(value.ErrorCode), value.ErrorMessage,
		boolToInt(value.Retryable), value.IdempotencyKey, value.CorrelationID, value.SourcePin,
		formatTime(value.CreatedAt), formatOptionalTime(value.StartedAt),
		formatOptionalTime(value.CompletedAt), formatTime(value.UpdatedAt))
	if err != nil {
		return mapOperationWriteError(err)
	}
	return nil
}

func (d *Database) GetOperation(ctx context.Context, id string) (operation.Operation, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, ErrOperationNotFound
	}
	return value, err
}

func (d *Database) GetOperationByIdempotencyKey(ctx context.Context, key string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+" WHERE idempotency_key = ?", key))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) ActiveRuntimeInstall(ctx context.Context, runtimeKind string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+`
        WHERE operation_type = ? AND target_type = ? AND target_id = ? AND status IN ('PENDING', 'RUNNING')
    `, string(operation.TypeRuntimeInstall), string(operation.TargetRuntimeKind), runtimeKind))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) ActiveHermesPrerequisite(ctx context.Context, runtimeKind string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+`
        WHERE operation_type = ? AND target_type = ? AND target_id = ? AND status IN ('PENDING', 'RUNNING')
    `, string(operation.TypeHermesPrerequisites), string(operation.TargetRuntimeKind), runtimeKind))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) ActiveHermesHostMutation(ctx context.Context, runtimeKind string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+`
        WHERE target_type = ? AND target_id = ? AND status IN ('PENDING', 'RUNNING')
          AND operation_type IN (?, ?)
    `, string(operation.TargetRuntimeKind), runtimeKind, string(operation.TypeRuntimeInstall), string(operation.TypeHermesPrerequisites)))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) LatestRuntimeInstall(ctx context.Context, runtimeKind string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+`
        WHERE operation_type = ? AND target_type = ? AND target_id = ?
        ORDER BY created_at DESC
        LIMIT 1
    `, string(operation.TypeRuntimeInstall), string(operation.TargetRuntimeKind), runtimeKind))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) PreviousRuntimeInstall(ctx context.Context, runtimeKind, excludeID string) (operation.Operation, bool, error) {
	value, err := scanOperation(d.db.QueryRowContext(ctx, operationSelect+`
        WHERE operation_type = ? AND target_type = ? AND target_id = ? AND id != ?
        ORDER BY created_at DESC
        LIMIT 1
    `, string(operation.TypeRuntimeInstall), string(operation.TargetRuntimeKind), runtimeKind, excludeID))
	if errors.Is(err, sql.ErrNoRows) {
		return operation.Operation{}, false, nil
	}
	if err != nil {
		return operation.Operation{}, false, err
	}
	return value, true, nil
}

func (d *Database) ListOperations(ctx context.Context, targetType, targetID string, limit int) ([]operation.Operation, error) {
	filter := OperationListFilter{TargetType: operation.TargetType(targetType), TargetID: targetID, Limit: limit}
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	query := operationSelect + " WHERE 1 = 1"
	args := make([]any, 0, 3)
	if filter.TargetType != "" {
		query += " AND target_type = ?"
		args = append(args, string(filter.TargetType))
	}
	if filter.TargetID != "" {
		query += " AND target_id = ?"
		args = append(args, filter.TargetID)
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, filter.Limit)
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()
	result := make([]operation.Operation, 0)
	for rows.Next() {
		value, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (d *Database) UpdateOperation(ctx context.Context, current, next operation.Operation) error {
	if current.ID != next.ID {
		return errors.New("operation identity cannot change")
	}
	if current.Status != next.Status && !operation.ValidTransition(current.Status, next.Status) {
		return ErrInvalidStatusTransition
	}
	if next.Progress != nil {
		return errors.New("operation progress must remain unset")
	}
	result, err := d.db.ExecContext(ctx, `
        UPDATE operations
        SET status = ?, stage = ?, progress = NULL, message = ?, error_code = ?, error_message = ?,
            retryable = ?, started_at = ?, completed_at = ?, updated_at = ?
        WHERE id = ? AND status = ?
    `, string(next.Status), string(next.Stage), next.Message, string(next.ErrorCode), next.ErrorMessage,
		boolToInt(next.Retryable), formatOptionalTime(next.StartedAt), formatOptionalTime(next.CompletedAt),
		formatTime(next.UpdatedAt), next.ID, string(current.Status))
	if err != nil {
		return mapOperationWriteError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("operation update rows: %w", err)
	}
	if affected != 1 {
		return ErrInvalidStatusTransition
	}
	return nil
}

func (d *Database) InterruptActiveInstalls(ctx context.Context, now time.Time) ([]operation.Operation, error) {
	rows, err := d.db.QueryContext(ctx, operationSelect+`
        WHERE operation_type IN (?, ?) AND status IN ('PENDING', 'RUNNING')
    `, string(operation.TypeRuntimeInstall), string(operation.TypeHermesPrerequisites))
	if err != nil {
		return nil, fmt.Errorf("list interrupted operations: %w", err)
	}
	var active []operation.Operation
	for rows.Next() {
		current, err := scanOperation(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		active = append(active, current)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	interrupted := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		next := current
		completed := now.UTC()
		next.Status = operation.StatusFailed
		next.ErrorCode = yorvaruntime.ErrorOperationInterrupted
		next.ErrorMessage = ""
		next.Retryable = true
		next.CompletedAt = &completed
		next.UpdatedAt = completed
		if err := d.UpdateOperation(ctx, current, next); err != nil {
			return nil, err
		}
		interrupted = append(interrupted, next)
	}
	return interrupted, nil
}

const operationSelect = `
    SELECT id, operation_type, target_type, target_id, status, stage, progress, message,
           error_code, error_message, retryable, idempotency_key, correlation_id, source_pin,
           created_at, started_at, completed_at, updated_at
    FROM operations
`

type operationRow interface {
	Scan(dest ...any) error
}

func scanOperation(row operationRow) (operation.Operation, error) {
	var value operation.Operation
	var targetType, stage, errorCode string
	var progress sql.NullInt64
	var startedAt, completedAt sql.NullString
	var createdAt, updatedAt string
	var retryable int
	if err := row.Scan(
		&value.ID, &value.Type, &targetType, &value.TargetID, &value.Status, &stage, &progress,
		&value.Message, &errorCode, &value.ErrorMessage, &retryable, &value.IdempotencyKey,
		&value.CorrelationID, &value.SourcePin, &createdAt, &startedAt, &completedAt, &updatedAt,
	); err != nil {
		return operation.Operation{}, err
	}
	if progress.Valid {
		return operation.Operation{}, errors.New("stored operation unexpectedly has percentage progress")
	}
	value.TargetType = operation.TargetType(targetType)
	value.Stage = operation.Stage(stage)
	value.ErrorCode = yorvaruntime.ErrorCode(errorCode)
	value.Retryable = retryable == 1
	var err error
	value.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return operation.Operation{}, fmt.Errorf("parse operation created_at: %w", err)
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return operation.Operation{}, fmt.Errorf("parse operation updated_at: %w", err)
	}
	if startedAt.Valid {
		started, err := parseTime(startedAt.String)
		if err != nil {
			return operation.Operation{}, fmt.Errorf("parse operation started_at: %w", err)
		}
		value.StartedAt = &started
	}
	if completedAt.Valid {
		completed, err := parseTime(completedAt.String)
		if err != nil {
			return operation.Operation{}, fmt.Errorf("parse operation completed_at: %w", err)
		}
		value.CompletedAt = &completed
	}
	return value, nil
}

func formatOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func mapOperationWriteError(err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "operations_idempotency_key") || strings.Contains(message, "operations.idempotency_key") {
		return ErrDuplicateIdempotency
	}
	if strings.Contains(message, "operations_one_active_hermes_host_mutation") ||
		strings.Contains(message, "operations_one_active_runtime_install") ||
		(strings.Contains(message, "operations.target_type") && strings.Contains(message, "operations.target_id")) ||
		(strings.Contains(message, "operations.operation_type") && strings.Contains(message, "operations.target_id")) {
		return ErrActiveInstallExists
	}
	return fmt.Errorf("write operation: %w", err)
}
