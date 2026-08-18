package app

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestSimultaneousSameKeyStartReturnsSameOperation(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	var lookups sync.WaitGroup
	lookups.Add(2)
	store.afterLookup = func() {
		lookups.Done()
		lookups.Wait()
		store.afterLookup = nil
	}

	results := make([]InstallStartResult, 2)
	errs := make([]error, 2)
	var started sync.WaitGroup
	started.Add(2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer started.Done()
			results[i], errs[i] = service.Start(context.Background(), "hermes", "same-key")
		}(i)
	}
	started.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("simultaneous Start() errors = %v, %v", errs[0], errs[1])
	}
	if results[0].Operation.ID == "" || results[0].Operation.ID != results[1].Operation.ID {
		t.Fatalf("simultaneous Start() returned different operations: %#v %#v", results[0], results[1])
	}
	createdCount := 0
	if results[0].Created {
		createdCount++
	}
	if results[1].Created {
		createdCount++
	}
	if createdCount != 1 {
		t.Fatalf("created count = %d, want exactly one winner", createdCount)
	}
	listed, err := service.List(context.Background(), string(operation.TargetRuntimeKind), "hermes", 10)
	if err != nil || len(listed) != 1 || listed[0].ID != results[0].Operation.ID {
		t.Fatalf("List() = %#v, %v", listed, err)
	}
}

func TestSimultaneousSameKeyPrerequisitesReturnSameOperation(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	var lookups sync.WaitGroup
	lookups.Add(2)
	store.afterLookup = func() {
		lookups.Done()
		lookups.Wait()
		store.afterLookup = nil
	}

	results := make([]InstallStartResult, 2)
	errs := make([]error, 2)
	var started sync.WaitGroup
	started.Add(2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer started.Done()
			results[i], errs[i] = service.StartPrerequisites(context.Background(), "same-prereq-key")
		}(i)
	}
	started.Wait()
	if errs[0] != nil || errs[1] != nil || results[0].Operation.ID != results[1].Operation.ID {
		t.Fatalf("simultaneous StartPrerequisites() = %#v/%v %#v/%v", results[0], errs[0], results[1], errs[1])
	}
	if results[0].Created == results[1].Created {
		t.Fatal("expected exactly one prerequisite create winner")
	}
}

func TestSameKeyDifferentTypeIsRejected(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	first, err := service.Start(context.Background(), "hermes", "shared-key")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.StartPrerequisites(context.Background(), "shared-key")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorIdempotencyKeyConflict {
		t.Fatalf("same key on prerequisite = %v", err)
	}
	if rejected.ActiveID != "" {
		t.Fatalf("key conflict must not impersonate host-mutation conflict: %#v", rejected)
	}
	got, err := service.Get(context.Background(), first.Operation.ID)
	if err != nil || got.Type != operation.TypeRuntimeInstall {
		t.Fatalf("original operation changed: %#v, %v", got, err)
	}
}

func TestSameKeyDifferentTargetIsRejected(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	if _, err := service.Start(context.Background(), "hermes", "target-key"); err != nil {
		t.Fatal(err)
	}
	_, err := service.replayIdempotent(operation.Operation{
		Type:       operation.TypeRuntimeInstall,
		TargetType: operation.TargetRuntimeKind,
		TargetID:   "other",
	}, operation.TypeRuntimeInstall, "hermes")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorIdempotencyKeyConflict {
		t.Fatalf("different target replay = %v", err)
	}
}

func TestCrossKeyHostMutationStillConflicts(t *testing.T) {
	store := newMemoryOperationStore()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled})
	first, err := service.Start(context.Background(), "hermes", "key-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Start(context.Background(), "hermes", "key-b")
	var rejected InstallRejection
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorRuntimeInstallInProgress || rejected.ActiveID != first.Operation.ID {
		t.Fatalf("cross-key Start() = %v", err)
	}
	_, err = service.StartPrerequisites(context.Background(), "key-c")
	if !errors.As(err, &rejected) || rejected.Code != yorvaruntime.ErrorRuntimeInstallInProgress || rejected.ActiveID != first.Operation.ID {
		t.Fatalf("cross-key prerequisites = %v", err)
	}
}
