package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestOperationsAndInstallationsMigrateFromEmptyAndPhase2(t *testing.T) {
	ctx := context.Background()
	emptyDir := t.TempDir()
	emptyDB, err := Open(ctx, emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := emptyDB.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationCount(t, emptyDir, 8)

	phase2Dir := t.TempDir()
	applyNamedMigration(t, ctx, phase2Dir, "001_initial.sql")
	assertMigrationCount(t, phase2Dir, 1)
	phase2DB, err := Open(ctx, phase2Dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := phase2DB.Close(); err != nil {
		t.Fatal(err)
	}
	assertMigrationCount(t, phase2Dir, 8)
}

func TestSimultaneousSameKeyCreateReturnsDuplicateIdempotency(t *testing.T) {
	ctx := context.Background()
	db := openTestDatabase(t)
	defer db.Close()
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	const workers = 8
	errs := make([]error, workers)
	var started sync.WaitGroup
	started.Add(workers)
	begin := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer started.Done()
			<-begin
			op := newTestOperation("op_race_"+strconv.Itoa(i), "same-persistent-key", operation.StatusPending, now)
			errs[i] = db.CreateOperation(ctx, op)
		}(i)
	}
	close(begin)
	started.Wait()

	success := 0
	duplicates := 0
	for _, err := range errs {
		if err == nil {
			success++
			continue
		}
		if errors.Is(err, ErrDuplicateIdempotency) || errors.Is(err, ErrActiveInstallExists) {
			duplicates++
			continue
		}
		t.Fatalf("unexpected create error = %v", err)
	}
	if success != 1 || duplicates != workers-1 {
		t.Fatalf("success=%d duplicates=%d, want 1/%d", success, duplicates, workers-1)
	}
	got, ok, err := db.GetOperationByIdempotencyKey(ctx, "same-persistent-key")
	if err != nil || !ok || got.IdempotencyKey != "same-persistent-key" {
		t.Fatalf("GetOperationByIdempotencyKey() = %#v ok=%v err=%v", got, ok, err)
	}
}

func TestOperationIdempotencyAndOneActiveInstall(t *testing.T) {
	ctx := context.Background()
	db := openTestDatabase(t)
	defer db.Close()
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	first := newTestOperation("op_1", "key-1", operation.StatusPending, now)
	first.SourcePin = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
	if err := db.CreateOperation(ctx, first); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.GetOperation(ctx, first.ID)
	if err != nil || loaded.SourcePin != first.SourcePin {
		t.Fatalf("source pin = %#v, %v", loaded, err)
	}
	if err := db.CreateOperation(ctx, newTestOperation("op_3", "key-2", operation.StatusPending, now)); !errors.Is(err, ErrActiveInstallExists) {
		t.Fatalf("second active install error = %v", err)
	}

	running := first
	started := now.Add(time.Second)
	running.Status = operation.StatusRunning
	running.StartedAt = &started
	running.UpdatedAt = started
	if err := db.UpdateOperation(ctx, first, running); err != nil {
		t.Fatal(err)
	}
	failed := running
	completed := now.Add(2 * time.Second)
	failed.Status = operation.StatusFailed
	failed.ErrorCode = yorvaruntime.ErrorOperationInterrupted
	failed.Retryable = true
	failed.CompletedAt = &completed
	failed.UpdatedAt = completed
	if err := db.UpdateOperation(ctx, running, failed); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateOperation(ctx, newTestOperation("op_2", "key-1", operation.StatusPending, completed)); !errors.Is(err, ErrDuplicateIdempotency) {
		t.Fatalf("duplicate key error = %v", err)
	}
	if err := db.CreateOperation(ctx, newTestOperation("op_4", "key-3", operation.StatusPending, completed)); err != nil {
		t.Fatalf("create after terminal error = %v", err)
	}
}

func TestHermesHostMutationIsExclusiveAcrossOperationTypes(t *testing.T) {
	ctx := context.Background()
	db := openTestDatabase(t)
	defer db.Close()
	now := time.Date(2026, 8, 17, 14, 0, 0, 0, time.UTC)
	install := newTestOperation("op_install", "key-install", operation.StatusPending, now)
	if err := db.CreateOperation(ctx, install); err != nil {
		t.Fatal(err)
	}
	prereq := operation.Operation{
		ID:             "op_prereq",
		Type:           operation.TypeHermesPrerequisites,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		IdempotencyKey: "key-prereq",
		CorrelationID:  "cor_prereq",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateOperation(ctx, prereq); !errors.Is(err, ErrActiveInstallExists) {
		t.Fatalf("cross-type create error = %v", err)
	}
	secondPrereq := prereq
	secondPrereq.ID = "op_prereq_2"
	secondPrereq.IdempotencyKey = "key-prereq-2"
	if err := db.CreateOperation(ctx, secondPrereq); !errors.Is(err, ErrActiveInstallExists) {
		t.Fatalf("second prerequisite create error = %v", err)
	}
	active, ok, err := db.ActiveHermesHostMutation(ctx, "hermes")
	if err != nil || !ok || active.ID != install.ID {
		t.Fatalf("ActiveHermesHostMutation() = %#v ok=%v err=%v", active, ok, err)
	}
}

func TestInterruptActiveInstallsAndAcceptedInstallationUniqueness(t *testing.T) {
	ctx := context.Background()
	db := openTestDatabase(t)
	defer db.Close()
	local, err := db.LoadOrCreateNode(ctx, node.LocalMetadata{
		Name: "TEST", Hostname: "TEST", Platform: "windows", Architecture: "amd64", NodeVersion: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 17, 13, 0, 0, 0, time.UTC)
	pending := newTestOperation("op_int", "key-int", operation.StatusPending, now)
	if err := db.CreateOperation(ctx, pending); err != nil {
		t.Fatal(err)
	}
	interrupted, err := db.InterruptActiveInstalls(ctx, now.Add(time.Minute))
	if err != nil || len(interrupted) != 1 || interrupted[0].Status != operation.StatusFailed {
		t.Fatalf("InterruptActiveInstalls() = %#v, %v", interrupted, err)
	}

	accepted := AcceptedInstallation{
		ID: "rtinst_1", NodeID: local.ID, RuntimeKind: "hermes",
		InstallPath: `C:\Users\test\AppData\Local\hermes\hermes-agent`, Version: "0.20.2",
		SupportState: yorvaruntime.DiscoverySupported, Status: "accepted",
		LastDetectedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.UpsertAcceptedInstallation(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	accepted.Version = "0.20.2"
	accepted.UpdatedAt = now.Add(time.Minute)
	if err := db.UpsertAcceptedInstallation(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	count, err := db.CountAcceptedInstallations(ctx)
	if err != nil || count != 1 {
		t.Fatalf("CountAcceptedInstallations() = %d, %v", count, err)
	}
}

func openTestDatabase(t *testing.T) *Database {
	t.Helper()
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func newTestOperation(id, key string, status operation.Status, now time.Time) operation.Operation {
	return operation.Operation{
		ID:             id,
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         status,
		Stage:          operation.StagePreflight,
		IdempotencyKey: key,
		CorrelationID:  "cor_test",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func assertMigrationCount(t *testing.T, dataDir string, want int) {
	t.Helper()
	raw, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var count int
	if err := raw.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("migration count = %d, want %d", count, want)
	}
}

func applyNamedMigration(t *testing.T, ctx context.Context, dataDir, name string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, databaseFilename)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TEXT NOT NULL
        )
    `); err != nil {
		t.Fatal(err)
	}
	script, err := migrationFiles.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(script)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (1, ?)", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
}
