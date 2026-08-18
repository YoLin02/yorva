package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestDecideInstallPreflight(t *testing.T) {
	retry := operation.Operation{
		ID:        "op_retry",
		Type:      operation.TypeRuntimeInstall,
		Status:    operation.StatusFailed,
		Retryable: true,
		ErrorCode: yorvaruntime.ErrorOperationInterrupted,
	}
	tests := []struct {
		name      string
		state     yorvaruntime.DiscoveryState
		latest    operation.Operation
		wantAllow bool
		wantCode  yorvaruntime.ErrorCode
	}{
		{name: "not installed", state: yorvaruntime.DiscoveryNotInstalled, wantAllow: true},
		{name: "supported", state: yorvaruntime.DiscoverySupported, wantCode: yorvaruntime.ErrorRuntimeInstallAlreadyPresent},
		{name: "unsupported", state: yorvaruntime.DiscoveryUnsupported, wantCode: yorvaruntime.ErrorRuntimeInstallStateConflict},
		{name: "ambiguous", state: yorvaruntime.DiscoveryAmbiguous, wantCode: yorvaruntime.ErrorRuntimeInstallStateConflict},
		{name: "malformed", state: yorvaruntime.DiscoveryMalformedVersion, wantCode: yorvaruntime.ErrorRuntimeInstallStateConflict},
		{name: "broken without retry", state: yorvaruntime.DiscoveryBrokenExecutable, wantCode: yorvaruntime.ErrorRuntimeInstallStateConflict},
		{name: "broken with retry", state: yorvaruntime.DiscoveryBrokenExecutable, latest: retry, wantAllow: true},
		{name: "timed out", state: yorvaruntime.DiscoveryTimedOut, wantCode: yorvaruntime.ErrorRuntimeDiscoveryTimeout},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DecideInstallPreflight(yorvaruntime.Discovery{State: test.state}, test.latest)
			if got.Allow != test.wantAllow || got.Code != test.wantCode {
				t.Fatalf("DecideInstallPreflight() = %#v, want allow=%v code=%s", got, test.wantAllow, test.wantCode)
			}
		})
	}
}

func TestRuntimeInstallStartIdempotencyAndConcurrency(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})

	first, err := service.Start(context.Background(), "hermes", "install-1")
	if err != nil || !first.Created || first.Operation.Status != operation.StatusPending {
		t.Fatalf("first Start() = %#v, %v", first, err)
	}
	repeat, err := service.Start(context.Background(), "hermes", "install-1")
	if err != nil || repeat.Created || repeat.Operation.ID != first.Operation.ID {
		t.Fatalf("repeat Start() = %#v, %v", repeat, err)
	}
	conflict, err := service.Start(context.Background(), "hermes", "install-2")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorRuntimeInstallInProgress || rejected.ActiveID != first.Operation.ID {
		t.Fatalf("second key Start() = %#v, %v", conflict, err)
	}
}

func TestRuntimeInstallAndPrerequisitesShareHostMutationLock(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	first, err := service.Start(context.Background(), "hermes", "install-lock")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.StartPrerequisites(context.Background(), "prereq-lock")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.ActiveID != first.Operation.ID {
		t.Fatalf("prerequisite during install = %v", err)
	}
	if _, err := service.Cancel(context.Background(), first.Operation.ID); err != nil {
		t.Fatal(err)
	}
	prereq, err := service.StartPrerequisites(context.Background(), "prereq-after")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Start(context.Background(), "hermes", "install-after-prereq")
	if !errors.As(err, &rejected) || rejected.ActiveID != prereq.Operation.ID {
		t.Fatalf("install during prerequisite = %v", err)
	}
}

func TestRuntimeInstallRejectsUnsupportedDiscovery(t *testing.T) {
	service := newTestRuntimeInstall(newMemoryOperationStore(), yorvaruntime.Discovery{State: yorvaruntime.DiscoverySupported})
	_, err := service.Start(context.Background(), "hermes", "install-supported")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorRuntimeInstallAlreadyPresent {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestRuntimeInstallCancelAndInterrupt(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	started, err := service.Start(context.Background(), "hermes", "install-cancel")
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.Cancel(context.Background(), started.Operation.ID)
	if err != nil || cancelled.Status != operation.StatusCancelled || !cancelled.Retryable {
		t.Fatalf("Cancel() = %#v, %v", cancelled, err)
	}
	if _, err := service.Cancel(context.Background(), started.Operation.ID); !errors.As(err, new(InstallRejection)) {
		t.Fatalf("second Cancel() error = %v", err)
	}

	second, err := service.Start(context.Background(), "hermes", "install-interrupt")
	if err != nil {
		t.Fatal(err)
	}
	interrupted, err := service.InterruptStale(context.Background())
	if err != nil || len(interrupted) != 1 || interrupted[0].ID != second.Operation.ID {
		t.Fatalf("InterruptStale() = %#v, %v", interrupted, err)
	}
	if interrupted[0].Status != operation.StatusFailed || interrupted[0].ErrorCode != yorvaruntime.ErrorOperationInterrupted {
		t.Fatalf("interrupted operation = %#v", interrupted[0])
	}
}

func TestDiscoveryRemainsAvailableDuringActiveHermesMutation(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	if _, err := service.Start(context.Background(), "hermes", "install-discover"); err != nil {
		t.Fatal(err)
	}
	got, err := service.discovery.Detect(context.Background(), "hermes")
	if err != nil || got.State != yorvaruntime.DiscoveryNotInstalled {
		t.Fatalf("discovery during install = %#v, %v", got, err)
	}
}

func TestRuntimeInstallLogsRejectedPreflight(t *testing.T) {
	var output bytes.Buffer
	service := newTestRuntimeInstall(newMemoryOperationStore(), yorvaruntime.Discovery{State: yorvaruntime.DiscoverySupported})
	service.WithLogger(slog.New(slog.NewJSONHandler(&output, nil)))
	_, err := service.Start(context.Background(), "hermes", "install-already")
	if err == nil {
		t.Fatal("expected already-present rejection")
	}
	var payload map[string]any
	if decodeErr := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &payload); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if payload["event"] != "rejected" || payload["errorCode"] != string(yorvaruntime.ErrorRuntimeInstallAlreadyPresent) {
		t.Fatalf("install log = %#v", payload)
	}
	if payload["stage"] != "preflight" || payload["runtimeKind"] != "hermes" {
		t.Fatalf("install log missing test location fields: %#v", payload)
	}
}

func TestRetryEligibleForPinRejectsStaleOperation(t *testing.T) {
	latest := operation.Operation{
		ID:        "op_stale",
		Type:      operation.TypeRuntimeInstall,
		Status:    operation.StatusFailed,
		Retryable: true,
		SourcePin: "cccccccccccccccccccccccccccccccccccccccc",
	}
	if !RetryEligible(latest) {
		t.Fatal("retryable history should remain retryable without a pin check")
	}
	if RetryEligibleForPin(latest, "df4b65147d7ddd74dd449f9067aabbca5aef0ec7") {
		t.Fatal("stale operation pin was treated as retry-eligible")
	}
	latest.SourcePin = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"
	if !RetryEligibleForPin(latest, "df4b65147d7ddd74dd449f9067aabbca5aef0ec7") {
		t.Fatal("matching operation pin was rejected")
	}
}

func TestOperationDeadlineIsSixtyMinutes(t *testing.T) {
	if operationDeadline != 60*time.Minute {
		t.Fatalf("operationDeadline = %s", operationDeadline)
	}
}

func TestValidateIdempotencyKey(t *testing.T) {
	if err := ValidateIdempotencyKey(""); err == nil {
		t.Fatal("empty key unexpectedly valid")
	}
	if err := ValidateIdempotencyKey("ok-key_1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateIdempotencyKey("has space"); err == nil {
		t.Fatal("whitespace key unexpectedly valid")
	}
}

func newTestRuntimeInstall(store *memoryOperationStore, discovery yorvaruntime.Discovery) *RuntimeInstall {
	registry := yorvaruntime.NewRegistry()
	_ = registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes Agent"},
		Discoverer: staticDiscoverer{discovery: discovery},
	})
	service := NewRuntimeInstall(NewRuntimeDiscovery(registry, slog.New(slog.NewTextHandler(io.Discard, nil))), store)
	service.now = func() time.Time { return time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC) }
	return service
}

type staticDiscoverer struct {
	discovery yorvaruntime.Discovery
}

func (s staticDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	return s.discovery, nil
}

type memoryOperationStore struct {
	mu           sync.Mutex
	byID         map[string]operation.Operation
	byKey        map[string]string
	beforeCreate func()
	afterLookup  func()
}

func newMemoryOperationStore() *memoryOperationStore {
	return &memoryOperationStore{
		byID:  make(map[string]operation.Operation),
		byKey: make(map[string]string),
	}
}

func (s *memoryOperationStore) CreateOperation(_ context.Context, value operation.Operation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.beforeCreate != nil {
		s.mu.Unlock()
		s.beforeCreate()
		s.mu.Lock()
	}
	if _, exists := s.byKey[value.IdempotencyKey]; exists {
		return sqlite.ErrDuplicateIdempotency
	}
	for _, current := range s.byID {
		if current.TargetID == value.TargetID && !operation.IsTerminal(current.Status) && hermesHostMutation(current.Type) && hermesHostMutation(value.Type) {
			return sqlite.ErrActiveInstallExists
		}
	}
	s.byID[value.ID] = value
	s.byKey[value.IdempotencyKey] = value.ID
	return nil
}

func (s *memoryOperationStore) GetOperation(_ context.Context, id string) (operation.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.byID[id]
	if !ok {
		return operation.Operation{}, errors.New("operation not found")
	}
	return value, nil
}

func (s *memoryOperationStore) GetOperationByIdempotencyKey(_ context.Context, key string) (operation.Operation, bool, error) {
	s.mu.Lock()
	id, ok := s.byKey[key]
	var value operation.Operation
	if ok {
		value = s.byID[id]
	}
	s.mu.Unlock()
	if s.afterLookup != nil {
		s.afterLookup()
	}
	if !ok {
		return operation.Operation{}, false, nil
	}
	return value, true, nil
}

func (s *memoryOperationStore) ActiveRuntimeInstall(_ context.Context, runtimeKind string) (operation.Operation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.byID {
		if value.Type == operation.TypeRuntimeInstall && value.TargetID == runtimeKind && !operation.IsTerminal(value.Status) {
			return value, true, nil
		}
	}
	return operation.Operation{}, false, nil
}

func (s *memoryOperationStore) ActiveHermesPrerequisite(_ context.Context, runtimeKind string) (operation.Operation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.byID {
		if value.Type == operation.TypeHermesPrerequisites && value.TargetID == runtimeKind && !operation.IsTerminal(value.Status) {
			return value, true, nil
		}
	}
	return operation.Operation{}, false, nil
}

func (s *memoryOperationStore) ActiveHermesHostMutation(_ context.Context, runtimeKind string) (operation.Operation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.byID {
		if value.TargetID == runtimeKind && !operation.IsTerminal(value.Status) && hermesHostMutation(value.Type) {
			return value, true, nil
		}
	}
	return operation.Operation{}, false, nil
}

func hermesHostMutation(value operation.Type) bool {
	return value == operation.TypeRuntimeInstall || value == operation.TypeHermesPrerequisites
}

func (s *memoryOperationStore) LatestRuntimeInstall(_ context.Context, runtimeKind string) (operation.Operation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest operation.Operation
	found := false
	for _, value := range s.byID {
		if value.Type != operation.TypeRuntimeInstall || value.TargetID != runtimeKind {
			continue
		}
		if !found || value.CreatedAt.After(latest.CreatedAt) {
			latest = value
			found = true
		}
	}
	return latest, found, nil
}

func (s *memoryOperationStore) ListOperations(_ context.Context, targetType, targetID string, limit int) ([]operation.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	result := make([]operation.Operation, 0)
	for _, value := range s.byID {
		if targetType != "" && string(value.TargetType) != targetType {
			continue
		}
		if targetID != "" && value.TargetID != targetID {
			continue
		}
		result = append(result, value)
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *memoryOperationStore) PreviousRuntimeInstall(_ context.Context, runtimeKind, excludeID string) (operation.Operation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest operation.Operation
	found := false
	for _, value := range s.byID {
		if value.Type != operation.TypeRuntimeInstall || value.TargetID != runtimeKind || value.ID == excludeID {
			continue
		}
		if !found || value.CreatedAt.After(latest.CreatedAt) {
			latest = value
			found = true
		}
	}
	return latest, found, nil
}

func (s *memoryOperationStore) UpdateOperation(_ context.Context, current, next operation.Operation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.byID[current.ID]
	if !ok || stored.Status != current.Status {
		return errors.New("operation update conflict")
	}
	if stored.Status != next.Status && !operation.ValidTransition(stored.Status, next.Status) {
		return errors.New("invalid transition")
	}
	s.byID[next.ID] = next
	return nil
}

func (s *memoryOperationStore) InterruptActiveInstalls(_ context.Context, now time.Time) ([]operation.Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []operation.Operation
	for id, current := range s.byID {
		if !hermesHostMutation(current.Type) || operation.IsTerminal(current.Status) {
			continue
		}
		next := current
		next.Status = operation.StatusFailed
		next.ErrorCode = yorvaruntime.ErrorOperationInterrupted
		next.Retryable = true
		completed := now
		next.CompletedAt = &completed
		next.UpdatedAt = now
		s.byID[id] = next
		result = append(result, next)
	}
	return result, nil
}
