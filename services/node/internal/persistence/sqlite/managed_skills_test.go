package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestManagedSkillsCRUDAndProjectionUpdatePreserveOwnership(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	instanceID := seedManagedSkillInstance(t, db)
	installedAt := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	value := testManagedSkill(instanceID, installedAt)

	if err := db.CreateManagedSkill(ctx, value); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetManagedSkill(ctx, instanceID, value.SkillID)
	if err != nil || got != value {
		t.Fatalf("GetManagedSkill() = %#v, %v; want %#v", got, err, value)
	}
	listed, err := db.ListManagedSkills(ctx, instanceID)
	if err != nil || len(listed) != 1 || listed[0] != value {
		t.Fatalf("ListManagedSkills() = %#v, %v", listed, err)
	}

	updated := value
	updated.SourceVersion = "1.1.0"
	updated.ContentSHA256 = strings.Repeat("b", 64)
	updated.ManagedRelativePath = "writer/1.1.0"
	updated.DeploymentID = "deployment-writer-1-1"
	updated.ProjectionState = yorvaruntime.SkillProjectionDriftMissing
	updated.UpdatedAt = installedAt.Add(time.Minute)
	if err := db.UpdateManagedSkill(ctx, updated); !errors.Is(err, ErrManagedSkillOwnership) {
		t.Fatalf("ordinary release rewrite = %v", err)
	}
	if err := db.ReplaceManagedSkillRelease(ctx, value, updated); err != nil {
		t.Fatal(err)
	}
	stale := updated
	stale.SourceVersion = "1.2.0"
	stale.ContentSHA256 = strings.Repeat("c", 64)
	stale.ManagedRelativePath = "writer/1.2.0"
	stale.DeploymentID = "deployment-writer-1-2"
	stale.UpdatedAt = installedAt.Add(2 * time.Minute)
	if err := db.ReplaceManagedSkillRelease(ctx, value, stale); !errors.Is(err, ErrManagedSkillStale) {
		t.Fatalf("stale release replacement = %v", err)
	}

	changedOwnership := updated
	changedOwnership.DeploymentID = "deployment-other"
	if err := db.UpdateManagedSkill(ctx, changedOwnership); !errors.Is(err, ErrManagedSkillOwnership) {
		t.Fatalf("ownership update error = %v", err)
	}

	projectionAt := installedAt.Add(3 * time.Minute)
	if err := db.UpdateManagedSkillProjection(ctx, instanceID, value.SkillID, false, yorvaruntime.SkillProjectionNotProjected, projectionAt); err != nil {
		t.Fatal(err)
	}
	projected, err := db.GetManagedSkill(ctx, instanceID, value.SkillID)
	if err != nil {
		t.Fatal(err)
	}
	if projected.DesiredEnabled || projected.ProjectionState != yorvaruntime.SkillProjectionNotProjected || !projected.UpdatedAt.Equal(projectionAt) {
		t.Fatalf("projection update = %#v", projected)
	}
	if projected.SourceVersion != updated.SourceVersion || projected.ContentSHA256 != updated.ContentSHA256 ||
		projected.ManagedRelativePath != updated.ManagedRelativePath || projected.ProjectionRelativePath != updated.ProjectionRelativePath ||
		projected.DeploymentID != updated.DeploymentID {
		t.Fatalf("projection update changed package/ownership metadata: %#v", projected)
	}

	if err := db.DeleteManagedSkill(ctx, instanceID, value.SkillID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetManagedSkill(ctx, instanceID, value.SkillID); !errors.Is(err, ErrManagedSkillNotFound) {
		t.Fatalf("GetManagedSkill() after delete = %v", err)
	}
	if err := db.DeleteManagedSkill(ctx, instanceID, value.SkillID); !errors.Is(err, ErrManagedSkillNotFound) {
		t.Fatalf("second DeleteManagedSkill() = %v", err)
	}
}

func TestManagedSkillsUniquenessValidationAndInstanceCascade(t *testing.T) {
	ctx := context.Background()
	db := openInstanceTestDB(t)
	instanceID := seedManagedSkillInstance(t, db)
	now := time.Date(2026, 8, 25, 13, 0, 0, 0, time.UTC)
	first := testManagedSkill(instanceID, now)
	if err := db.CreateManagedSkill(ctx, first); err != nil {
		t.Fatal(err)
	}

	duplicateSkill := first
	duplicateSkill.ID = "msk_duplicate"
	duplicateSkill.DeploymentID = "deployment-duplicate"
	if err := db.CreateManagedSkill(ctx, duplicateSkill); !errors.Is(err, ErrManagedSkillConflict) {
		t.Fatalf("duplicate (instance, Skill) = %v", err)
	}
	duplicateDeployment := first
	duplicateDeployment.ID = "msk_second"
	duplicateDeployment.SkillID = "second-skill"
	duplicateDeployment.ManagedRelativePath = "second-skill/1.0.0"
	duplicateDeployment.ProjectionRelativePath = "skills/second-skill"
	if err := db.CreateManagedSkill(ctx, duplicateDeployment); !errors.Is(err, ErrManagedSkillConflict) {
		t.Fatalf("duplicate deployment ID = %v", err)
	}

	invalid := first
	invalid.ID = "msk_invalid"
	invalid.SkillID = "invalid-skill"
	invalid.DeploymentID = "deployment-invalid"
	invalid.ManagedRelativePath = "../escape"
	if err := db.CreateManagedSkill(ctx, invalid); !errors.Is(err, ErrInvalidManagedSkill) {
		t.Fatalf("unsafe managed path = %v", err)
	}
	invalid = first
	invalid.ID = "msk_hash"
	invalid.SkillID = "invalid-hash"
	invalid.DeploymentID = "deployment-hash"
	invalid.ContentSHA256 = strings.Repeat("A", 64)
	if err := db.CreateManagedSkill(ctx, invalid); !errors.Is(err, ErrInvalidManagedSkill) {
		t.Fatalf("uppercase content SHA-256 = %v", err)
	}
	if _, err := db.db.ExecContext(ctx, "UPDATE managed_skills SET projection_state = 'BROKEN' WHERE id = ?", first.ID); err == nil {
		t.Fatal("migration accepted an open projection state")
	}

	if _, err := db.db.ExecContext(ctx, "DELETE FROM instances WHERE id = ?", instanceID); err != nil {
		t.Fatal(err)
	}
	listed, err := db.ListManagedSkills(ctx, instanceID)
	if err != nil || len(listed) != 0 {
		t.Fatalf("cascade list = %#v, %v", listed, err)
	}
}

func TestManagedSkillsMigrationUpgradesPriorSchema(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	for _, name := range []string{
		"001_initial.sql",
		"002_operations_and_installations.sql",
		"003_hermes_host_mutation.sql",
		"004_operation_source_pin.sql",
		"005_operation_ownership_nonce.sql",
		"006_operation_transaction_id.sql",
		"007_instances.sql",
		"008_instance_operations.sql",
		"009_instance_lifecycle_operations.sql",
		"010_channel_bindings.sql",
		"011_runtime_backup_index.sql",
	} {
		applyVersionedMigration(t, ctx, dir, name)
	}
	assertMigrationCount(t, dir, 11)

	db, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationCount(t, dir, 16)

	raw, err := sql.Open("sqlite", filepath.Join(dir, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var tableCount int
	if err := raw.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'managed_skills'").Scan(&tableCount); err != nil || tableCount != 1 {
		t.Fatalf("managed_skills table count = %d, %v", tableCount, err)
	}
	var indexSQL string
	if err := raw.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'operations_one_active_instance_runtime_mutation'").Scan(&indexSQL); err != nil {
		t.Fatal(err)
	}
	for _, operationType := range []string{"skill.install", "skill.update", "skill.enable", "skill.disable", "skill.remove"} {
		if !strings.Contains(indexSQL, operationType) {
			t.Fatalf("active mutation index does not include %q: %s", operationType, indexSQL)
		}
	}
}

func seedManagedSkillInstance(t *testing.T, db *Database) string {
	t.Helper()
	ctx := context.Background()
	installationID := seedInstallation(t, db)
	now := time.Date(2026, 8, 25, 11, 0, 0, 0, time.UTC)
	if err := db.ApplyInstanceSnapshot(ctx, installationID, []InstanceSnapshotEntry{{NativeID: "default", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	instances, err := db.ListInstances(ctx, installationID)
	if err != nil || len(instances) != 1 {
		t.Fatalf("ListInstances() = %#v, %v", instances, err)
	}
	return instances[0].ID
}

func testManagedSkill(instanceID string, now time.Time) ManagedSkill {
	return ManagedSkill{
		ID:                     "msk_writer",
		InstanceID:             instanceID,
		SkillID:                "writer",
		SourceID:               "reviewed-writer",
		SourceVersion:          "1.0.0",
		ContentSHA256:          strings.Repeat("a", 64),
		ManagedRelativePath:    "writer/1.0.0",
		ProjectionRelativePath: "skills/writer",
		DeploymentID:           "deployment-writer",
		DesiredEnabled:         true,
		ProjectionState:        yorvaruntime.SkillProjectionProjected,
		InstalledAt:            now,
		UpdatedAt:              now,
	}
}
