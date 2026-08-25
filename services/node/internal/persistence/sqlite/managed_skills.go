package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrManagedSkillNotFound  = errors.New("managed Skill not found")
	ErrManagedSkillConflict  = errors.New("managed Skill identity already exists")
	ErrManagedSkillOwnership = errors.New("managed Skill ownership cannot change")
	ErrManagedSkillStale     = errors.New("managed Skill changed concurrently")
	ErrInvalidManagedSkill   = errors.New("invalid managed Skill")
)

// ManagedSkill is YORVA-owned lifecycle metadata. ManagedRelativePath and
// ProjectionRelativePath are rooted by trusted application/adapter code; they
// are never caller-selected filesystem paths.
type ManagedSkill struct {
	ID                     string
	InstanceID             string
	SkillID                string
	SourceID               string
	SourceVersion          string
	ContentSHA256          string
	ManagedRelativePath    string
	ProjectionRelativePath string
	DeploymentID           string
	DesiredEnabled         bool
	ProjectionState        yorvaruntime.SkillProjectionState
	InstalledAt            time.Time
	UpdatedAt              time.Time
}

func NewManagedSkillID() (string, error) {
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate managed Skill ID: %w", err)
	}
	return "msk_" + base64.RawURLEncoding.EncodeToString(random), nil
}

func (d *Database) CreateManagedSkill(ctx context.Context, value ManagedSkill) error {
	if err := validateManagedSkill(value); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, `
        INSERT INTO managed_skills(
            id, instance_id, skill_id, source_id, source_version, content_sha256,
            managed_relative_path, projection_relative_path, deployment_id,
            desired_enabled, projection_state, installed_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, value.ID, value.InstanceID, value.SkillID, value.SourceID, value.SourceVersion,
		value.ContentSHA256, value.ManagedRelativePath, value.ProjectionRelativePath,
		value.DeploymentID, boolToInt(value.DesiredEnabled), string(value.ProjectionState),
		formatTime(value.InstalledAt), formatTime(value.UpdatedAt))
	if err != nil {
		return mapManagedSkillWriteError("create managed Skill", err)
	}
	return nil
}

func (d *Database) GetManagedSkill(ctx context.Context, instanceID, skillID string) (ManagedSkill, error) {
	value, err := scanManagedSkill(d.db.QueryRowContext(ctx, managedSkillSelect+`
        WHERE instance_id = ? AND skill_id = ?
    `, instanceID, skillID))
	if errors.Is(err, sql.ErrNoRows) {
		return ManagedSkill{}, ErrManagedSkillNotFound
	}
	if err != nil {
		return ManagedSkill{}, fmt.Errorf("get managed Skill: %w", err)
	}
	return value, nil
}

func (d *Database) ListManagedSkills(ctx context.Context, instanceID string) ([]ManagedSkill, error) {
	rows, err := d.db.QueryContext(ctx, managedSkillSelect+`
        WHERE instance_id = ?
        ORDER BY skill_id ASC
    `, instanceID)
	if err != nil {
		return nil, fmt.Errorf("list managed Skills: %w", err)
	}
	defer rows.Close()

	result := make([]ManagedSkill, 0)
	for rows.Next() {
		value, err := scanManagedSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("list managed Skills: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list managed Skills: %w", err)
	}
	return result, nil
}

// UpdateManagedSkill is the ordinary state update and cannot replace package
// or ownership evidence. A verified new release uses ReplaceManagedSkillRelease.
func (d *Database) UpdateManagedSkill(ctx context.Context, value ManagedSkill) error {
	if err := validateManagedSkill(value); err != nil {
		return err
	}
	stored, err := d.getManagedSkillByID(ctx, value.ID)
	if err != nil {
		return err
	}
	if !sameManagedSkillOwnership(stored, value) {
		return ErrManagedSkillOwnership
	}
	result, err := d.db.ExecContext(ctx, `
        UPDATE managed_skills
        SET desired_enabled = ?, projection_state = ?, updated_at = ?
        WHERE id = ?
    `, boolToInt(value.DesiredEnabled), string(value.ProjectionState),
		formatTime(value.UpdatedAt), value.ID)
	if err != nil {
		return mapManagedSkillWriteError("update managed Skill", err)
	}
	return requireOneManagedSkill(result)
}

// ReplaceManagedSkillRelease atomically replaces verified release evidence.
// Stable YORVA ownership (row, Instance, Skill, source and install time) cannot
// change. The complete previous value is a CAS precondition so a stale worker
// cannot overwrite a newer release.
func (d *Database) ReplaceManagedSkillRelease(ctx context.Context, current, next ManagedSkill) error {
	if err := validateManagedSkill(current); err != nil {
		return err
	}
	if err := validateManagedSkill(next); err != nil {
		return err
	}
	if !sameManagedSkillStableIdentity(current, next) {
		return ErrManagedSkillOwnership
	}
	result, err := d.db.ExecContext(ctx, `
        UPDATE managed_skills
        SET source_version = ?, content_sha256 = ?, managed_relative_path = ?,
            projection_relative_path = ?, deployment_id = ?, desired_enabled = ?,
            projection_state = ?, updated_at = ?
        WHERE id = ? AND instance_id = ? AND skill_id = ? AND source_id = ?
          AND source_version = ? AND content_sha256 = ?
          AND managed_relative_path = ? AND projection_relative_path = ?
          AND deployment_id = ? AND desired_enabled = ? AND projection_state = ?
          AND installed_at = ? AND updated_at = ?
    `, next.SourceVersion, next.ContentSHA256, next.ManagedRelativePath,
		next.ProjectionRelativePath, next.DeploymentID, boolToInt(next.DesiredEnabled),
		string(next.ProjectionState), formatTime(next.UpdatedAt),
		current.ID, current.InstanceID, current.SkillID, current.SourceID,
		current.SourceVersion, current.ContentSHA256, current.ManagedRelativePath,
		current.ProjectionRelativePath, current.DeploymentID, boolToInt(current.DesiredEnabled),
		string(current.ProjectionState), formatTime(current.InstalledAt), formatTime(current.UpdatedAt))
	if err != nil {
		return mapManagedSkillWriteError("replace managed Skill release", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count replaced managed Skill releases: %w", err)
	}
	if affected != 1 {
		return ErrManagedSkillStale
	}
	return nil
}

// UpdateManagedSkillProjection changes only desired/observed projection state.
// Package source, digest, deployment identity and both relative paths remain
// immutable through this narrower reconciliation operation.
func (d *Database) UpdateManagedSkillProjection(
	ctx context.Context,
	instanceID string,
	skillID string,
	desiredEnabled bool,
	state yorvaruntime.SkillProjectionState,
	updatedAt time.Time,
) error {
	if instanceID == "" || skillID == "" || !state.Valid() || updatedAt.IsZero() {
		return ErrInvalidManagedSkill
	}
	result, err := d.db.ExecContext(ctx, `
        UPDATE managed_skills
        SET desired_enabled = ?, projection_state = ?, updated_at = ?
        WHERE instance_id = ? AND skill_id = ?
    `, boolToInt(desiredEnabled), string(state), formatTime(updatedAt), instanceID, skillID)
	if err != nil {
		return mapManagedSkillWriteError("update managed Skill projection", err)
	}
	return requireOneManagedSkill(result)
}

func (d *Database) DeleteManagedSkill(ctx context.Context, instanceID, skillID string) error {
	result, err := d.db.ExecContext(ctx, `
        DELETE FROM managed_skills WHERE instance_id = ? AND skill_id = ?
    `, instanceID, skillID)
	if err != nil {
		return fmt.Errorf("delete managed Skill: %w", err)
	}
	return requireOneManagedSkill(result)
}

func (d *Database) getManagedSkillByID(ctx context.Context, id string) (ManagedSkill, error) {
	value, err := scanManagedSkill(d.db.QueryRowContext(ctx, managedSkillSelect+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return ManagedSkill{}, ErrManagedSkillNotFound
	}
	if err != nil {
		return ManagedSkill{}, fmt.Errorf("get managed Skill by ID: %w", err)
	}
	return value, nil
}

func validateManagedSkill(value ManagedSkill) error {
	if !validManagedSkillID(value.ID) || !boundedMetadata(value.InstanceID, 128) ||
		!validManagedSkillID(value.SkillID) || !validManagedSkillID(value.SourceID) ||
		!boundedMetadata(value.SourceVersion, 4096) || !validManagedSkillID(value.DeploymentID) ||
		!validLowerSHA256(value.ContentSHA256) ||
		!validManagedRelativePath(value.ManagedRelativePath) ||
		!validManagedRelativePath(value.ProjectionRelativePath) ||
		!value.ProjectionState.Valid() || value.InstalledAt.IsZero() || value.UpdatedAt.IsZero() {
		return ErrInvalidManagedSkill
	}
	return nil
}

func validManagedSkillID(value string) bool {
	if !boundedMetadata(value, 128) || value == "." || value == ".." {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func validManagedRelativePath(value string) bool {
	return boundedMetadata(value, 4096) && path.Clean(value) == value && value != "." &&
		!strings.HasPrefix(value, "/") && !strings.Contains(value, `\`) &&
		!strings.Contains(value, ":") && value != ".." && !strings.HasPrefix(value, "../") &&
		!strings.Contains(value, "/../") && !strings.HasSuffix(value, "/..")
}

func sameManagedSkillOwnership(left, right ManagedSkill) bool {
	return sameManagedSkillStableIdentity(left, right) &&
		left.SourceVersion == right.SourceVersion && left.ContentSHA256 == right.ContentSHA256 &&
		left.ManagedRelativePath == right.ManagedRelativePath &&
		left.ProjectionRelativePath == right.ProjectionRelativePath &&
		left.DeploymentID == right.DeploymentID
}

func sameManagedSkillStableIdentity(left, right ManagedSkill) bool {
	return left.ID == right.ID && left.InstanceID == right.InstanceID &&
		left.SkillID == right.SkillID && left.SourceID == right.SourceID &&
		left.InstalledAt.Equal(right.InstalledAt)
}

func requireOneManagedSkill(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count managed Skill rows: %w", err)
	}
	if affected != 1 {
		return ErrManagedSkillNotFound
	}
	return nil
}

func mapManagedSkillWriteError(action string, err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "managed_skills.id") || strings.Contains(message, "managed_skills.deployment_id") ||
		(strings.Contains(message, "managed_skills.instance_id") && strings.Contains(message, "managed_skills.skill_id")) {
		return ErrManagedSkillConflict
	}
	return fmt.Errorf("%s: %w", action, err)
}

const managedSkillSelect = `
    SELECT id, instance_id, skill_id, source_id, source_version, content_sha256,
           managed_relative_path, projection_relative_path, deployment_id,
           desired_enabled, projection_state, installed_at, updated_at
    FROM managed_skills`

type managedSkillScanner interface {
	Scan(...any) error
}

func scanManagedSkill(row managedSkillScanner) (ManagedSkill, error) {
	var value ManagedSkill
	var desiredEnabled int
	var installedAt, updatedAt string
	if err := row.Scan(
		&value.ID, &value.InstanceID, &value.SkillID, &value.SourceID,
		&value.SourceVersion, &value.ContentSHA256, &value.ManagedRelativePath,
		&value.ProjectionRelativePath, &value.DeploymentID, &desiredEnabled,
		&value.ProjectionState, &installedAt, &updatedAt,
	); err != nil {
		return ManagedSkill{}, err
	}
	value.DesiredEnabled = desiredEnabled == 1
	var err error
	value.InstalledAt, err = parseTime(installedAt)
	if err != nil {
		return ManagedSkill{}, err
	}
	value.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return ManagedSkill{}, err
	}
	return value, nil
}
