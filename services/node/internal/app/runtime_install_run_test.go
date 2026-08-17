package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestRuntimeInstallWorkerSucceedsOnlyAfterSupportedPostcheck(t *testing.T) {
	store := newMemoryOperationStore()
	applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent\venv\Scripts\python.exe`, version: "0.20.2"}
	completer := &fakeCompleter{store: store}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
		{
			State:    yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: applier.dir, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		},
	}, applier, completer)

	started, err := service.Start(context.Background(), "hermes", "install-success")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, service, started.Operation.ID, operation.StatusSucceeded)
	saved := completer.snapshot()
	if saved.Version != "0.20.2" || saved.InstallPath != applier.dir {
		t.Fatalf("accepted installation = %#v", saved)
	}
}

func TestRuntimeInstallWorkerRejectsFailedPostcheckWithoutInstallation(t *testing.T) {
	store := newMemoryOperationStore()
	applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent`, version: "0.20.2"}
	completer := &fakeCompleter{store: store}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
		{State: yorvaruntime.DiscoveryBrokenExecutable},
	}, applier, completer)
	started, err := service.Start(context.Background(), "hermes", "install-postcheck")
	if err != nil {
		t.Fatal(err)
	}
	got := waitForStatus(t, service, started.Operation.ID, operation.StatusFailed)
	if got.ErrorCode != yorvaruntime.ErrorRuntimeInstallPostcheckFailed || completer.snapshot().ID != "" {
		t.Fatalf("failed postcheck = %#v installation=%#v", got, completer.snapshot())
	}
}

func TestRuntimeInstallWorkerCancelBeforeSuccess(t *testing.T) {
	store := newMemoryOperationStore()
	startedApply := make(chan struct{})
	applier := &fakeApplier{
		dir: `C:\Users\a\AppData\Local\hermes\hermes-agent`, version: "0.20.2",
		block: startedApply,
	}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
	}, applier, &fakeCompleter{})
	started, err := service.Start(context.Background(), "hermes", "install-cancel-run")
	if err != nil {
		t.Fatal(err)
	}
	<-startedApply
	if _, err := service.Cancel(context.Background(), started.Operation.ID); err != nil {
		t.Fatal(err)
	}
	got := waitForStatus(t, service, started.Operation.ID, operation.StatusCancelled)
	if !got.Retryable {
		t.Fatalf("cancelled operation = %#v", got)
	}
}

func newOrchestratedInstall(store *memoryOperationStore, discoveries []yorvaruntime.Discovery, applier *fakeApplier, completer *fakeCompleter) *RuntimeInstall {
	registry := yorvaruntime.NewRegistry()
	_ = registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes Agent"},
		Discoverer: &sequenceDiscoverer{results: discoveries},
	})
	service := NewRuntimeInstall(NewRuntimeDiscovery(registry, slog.New(slog.NewTextHandler(io.Discard, nil))), store)
	service.now = func() time.Time { return time.Now().UTC() }
	service.allowNonWindows = true
	return service.WithHost(applier, completer, "node_test")
}

func waitForStatus(t *testing.T, service *RuntimeInstall, id string, want operation.Status) operation.Operation {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last operation.Operation
	for time.Now().Before(deadline) {
		got, err := service.Get(context.Background(), id)
		if err == nil && got.Status == want {
			return got
		}
		if err == nil {
			last = got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %s status = %#v, want %s", id, last, want)
	return last
}

type sequenceDiscoverer struct {
	mu      sync.Mutex
	results []yorvaruntime.Discovery
	index   int
}

func (s *sequenceDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index >= len(s.results) {
		return s.results[len(s.results)-1], nil
	}
	result := s.results[s.index]
	s.index++
	return result, nil
}

type fakeApplier struct {
	dir     string
	version string
	block   chan struct{}
}

func (f *fakeApplier) PlatformSupported() bool   { return true }
func (f *fakeApplier) ValidateTarget(bool) error { return nil }
func (f *fakeApplier) ManagedInstallDir() string { return f.dir }
func (f *fakeApplier) ExpectedVersion() string   { return f.version }
func (f *fakeApplier) ContainsManagedPath(path string) bool {
	return path == f.dir
}
func (f *fakeApplier) Apply(ctx context.Context, _ string, report func(operation.Stage)) error {
	if report != nil {
		report(operation.StageSourceDownload)
	}
	if f.block != nil {
		close(f.block)
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

type fakeCompleter struct {
	mu    sync.Mutex
	store *memoryOperationStore
	saved sqlite.AcceptedInstallation
}

func (f *fakeCompleter) CompleteInstallSuccess(ctx context.Context, current, next operation.Operation, installation sqlite.AcceptedInstallation) error {
	f.mu.Lock()
	f.saved = installation
	f.mu.Unlock()
	return f.store.UpdateOperation(ctx, current, next)
}

func (f *fakeCompleter) snapshot() sqlite.AcceptedInstallation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saved
}
