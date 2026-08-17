package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
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
	mu    sync.Mutex
	byID  map[string]operation.Operation
	byKey map[string]string
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
	if _, exists := s.byKey[value.IdempotencyKey]; exists {
		return errors.New("duplicate idempotency key")
	}
	for _, current := range s.byID {
		if current.Type == value.Type && current.TargetID == value.TargetID && !operation.IsTerminal(current.Status) {
			return errors.New("active install exists")
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
	defer s.mu.Unlock()
	id, ok := s.byKey[key]
	if !ok {
		return operation.Operation{}, false, nil
	}
	return s.byID[id], true, nil
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
		if current.Type != operation.TypeRuntimeInstall || operation.IsTerminal(current.Status) {
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
