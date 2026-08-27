package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeDiscoverer struct {
	result yorvaruntime.Discovery
	err    error
}

func (d fakeDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	return d.result, d.err
}

func TestRuntimeDiscoveryDispatchesThroughRegistry(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	want := yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported}
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: fakeDiscoverer{result: want},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := NewRuntimeDiscovery(registry, nil).Detect(context.Background(), "hermes")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got.State != want.State || got.RuntimeKind != want.RuntimeKind {
		t.Fatalf("Detect() = %#v, want %#v", got, want)
	}
}

func TestRuntimeDiscoveryRejectsUnknownOrUnavailableKind(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("empty", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "empty", Name: "Empty"},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	service := NewRuntimeDiscovery(registry, nil)
	for _, kind := range []yorvaruntime.Kind{"missing", "empty"} {
		if _, err := service.Detect(context.Background(), kind); !errors.Is(err, ErrRuntimeKindNotFound) {
			t.Fatalf("Detect(%q) error = %v, want ErrRuntimeKindNotFound", kind, err)
		}
	}
}

func TestRuntimeDiscoveryPropagatesAdapterError(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	wantErr := errors.New("adapter failed")
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: fakeDiscoverer{err: wantErr},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := NewRuntimeDiscovery(registry, nil).Detect(context.Background(), "hermes"); !errors.Is(err, wantErr) {
		t.Fatalf("Detect() error = %v, want %v", err, wantErr)
	}
}

type waitingDiscoverer struct {
	finished chan error
}

type countedDiscoverer struct {
	calls    atomic.Int32
	entered  chan struct{}
	release  chan struct{}
	finished chan error
	result   yorvaruntime.Discovery
}

func (d *countedDiscoverer) Detect(ctx context.Context) (yorvaruntime.Discovery, error) {
	d.calls.Add(1)
	if d.entered != nil {
		d.entered <- struct{}{}
	}
	if d.release == nil {
		return d.result, nil
	}
	select {
	case <-d.release:
		if d.finished != nil {
			d.finished <- nil
		}
		return d.result, nil
	case <-ctx.Done():
		if d.finished != nil {
			d.finished <- ctx.Err()
		}
		return yorvaruntime.Discovery{}, ctx.Err()
	}
}

func newRuntimeDiscoveryWithDiscoverer(t *testing.T, discoverer yorvaruntime.Discoverer) *RuntimeDiscovery {
	t.Helper()
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: discoverer,
	}); err != nil {
		t.Fatal(err)
	}
	return NewRuntimeDiscovery(registry, nil)
}

func TestRuntimeDiscoveryCoalescesConcurrentRequests(t *testing.T) {
	discoverer := &countedDiscoverer{
		entered: make(chan struct{}, 8), release: make(chan struct{}),
		result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported},
	}
	service := newRuntimeDiscoveryWithDiscoverer(t, discoverer)
	const callers = 8
	var group sync.WaitGroup
	errorsSeen := make(chan error, callers)
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			_, err := service.Detect(context.Background(), "hermes")
			errorsSeen <- err
		}()
	}
	select {
	case <-discoverer.entered:
	case <-time.After(time.Second):
		t.Fatal("discovery did not start")
	}
	close(discoverer.release)
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
	}
	if got := discoverer.calls.Load(); got != 1 {
		t.Fatalf("adapter calls = %d, want one coalesced probe", got)
	}
}

func TestRuntimeDiscoveryCachesOnlySupportedResults(t *testing.T) {
	supported := &countedDiscoverer{result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported}}
	service := newRuntimeDiscoveryWithDiscoverer(t, supported)
	for range 2 {
		if _, err := service.Detect(context.Background(), "hermes"); err != nil {
			t.Fatal(err)
		}
	}
	if got := supported.calls.Load(); got != 1 {
		t.Fatalf("supported adapter calls = %d, want cached result", got)
	}

	timedOut := &countedDiscoverer{result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoveryTimedOut, ErrorCode: yorvaruntime.ErrorRuntimeDiscoveryTimeout}}
	service = newRuntimeDiscoveryWithDiscoverer(t, timedOut)
	for range 2 {
		if _, err := service.Detect(context.Background(), "hermes"); err != nil {
			t.Fatal(err)
		}
	}
	if got := timedOut.calls.Load(); got != 2 {
		t.Fatalf("timed-out adapter calls = %d, want a fresh retry", got)
	}
}

func TestRuntimeDiscoveryKeepsSharedProbeUntilLastWaiterCancels(t *testing.T) {
	discoverer := &countedDiscoverer{
		entered: make(chan struct{}, 1), release: make(chan struct{}), finished: make(chan error, 1),
		result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported},
	}
	service := newRuntimeDiscoveryWithDiscoverer(t, discoverer)
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	secondResult := make(chan error, 1)
	go func() { _, err := service.Detect(firstCtx, "hermes"); firstResult <- err }()
	<-discoverer.entered
	go func() { _, err := service.Detect(context.Background(), "hermes"); secondResult <- err }()
	deadline := time.Now().Add(time.Second)
	for {
		service.mu.Lock()
		waiters := service.flights["hermes"].waiters
		service.mu.Unlock()
		if waiters == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second caller did not join the shared probe")
		}
		time.Sleep(time.Millisecond)
	}
	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("first Detect() error = %v, want cancellation", err)
	}
	close(discoverer.release)
	if err := <-secondResult; err != nil {
		t.Fatalf("shared Detect() error = %v", err)
	}
	if err := <-discoverer.finished; err != nil {
		t.Fatalf("shared adapter was cancelled while a waiter remained: %v", err)
	}
	if got := discoverer.calls.Load(); got != 1 {
		t.Fatalf("adapter calls = %d, want one", got)
	}
}

func TestRuntimeDiscoveryStartsFreshProbeAfterOnlyWaiterCancels(t *testing.T) {
	discoverer := &countedDiscoverer{
		entered: make(chan struct{}, 2), release: make(chan struct{}), finished: make(chan error, 2),
		result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported},
	}
	service := newRuntimeDiscoveryWithDiscoverer(t, discoverer)
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() { _, err := service.Detect(firstCtx, "hermes"); firstResult <- err }()
	<-discoverer.entered
	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("first Detect() error = %v, want cancellation", err)
	}

	secondResult := make(chan error, 1)
	go func() { _, err := service.Detect(context.Background(), "hermes"); secondResult <- err }()
	select {
	case <-discoverer.entered:
	case <-time.After(time.Second):
		t.Fatal("replacement discovery did not start")
	}
	close(discoverer.release)
	if err := <-secondResult; err != nil {
		t.Fatalf("replacement Detect() error = %v", err)
	}
	if got := discoverer.calls.Load(); got != 2 {
		t.Fatalf("adapter calls = %d, want cancelled and replacement probes", got)
	}
}

func (d waitingDiscoverer) Detect(ctx context.Context) (yorvaruntime.Discovery, error) {
	<-ctx.Done()
	d.finished <- ctx.Err()
	return yorvaruntime.Discovery{}, ctx.Err()
}

func TestRuntimeDiscoveryOwnsOverallDeadline(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	finished := make(chan error, 1)
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: waitingDiscoverer{finished: finished},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	service := NewRuntimeDiscovery(registry, nil)
	service.timeout = time.Millisecond

	if _, err := service.Detect(context.Background(), "hermes"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Detect() error = %v, want context.DeadlineExceeded", err)
	}
	if got := <-finished; !errors.Is(got, context.DeadlineExceeded) {
		t.Fatalf("adapter context error = %v, want context.DeadlineExceeded", got)
	}
}

func TestRuntimeDiscoveryLogsSafeStructuredOutcomes(t *testing.T) {
	states := []yorvaruntime.DiscoveryState{
		yorvaruntime.DiscoveryNotInstalled,
		yorvaruntime.DiscoverySupported,
		yorvaruntime.DiscoveryUnsupported,
		yorvaruntime.DiscoveryBrokenExecutable,
		yorvaruntime.DiscoveryMalformedVersion,
		yorvaruntime.DiscoveryTimedOut,
		yorvaruntime.DiscoveryAmbiguous,
	}
	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			var output bytes.Buffer
			registry := yorvaruntime.NewRegistry()
			result := yorvaruntime.Discovery{
				RuntimeKind: "hermes",
				State:       state,
				Candidates: []yorvaruntime.Candidate{{
					Path: `C:\\Users\\private-user\\hermes.exe`,
				}},
				Warnings: []yorvaruntime.Warning{{Code: "SAFE_CODE", Message: "provider-secret-value"}},
			}
			if err := registry.Register("hermes", yorvaruntime.Bundle{
				Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
				Discoverer: fakeDiscoverer{result: result},
			}); err != nil {
				t.Fatal(err)
			}
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			if _, err := NewRuntimeDiscovery(registry, logger).Detect(context.Background(), "hermes"); err != nil {
				t.Fatal(err)
			}

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatalf("decode log: %v\n%s", err, output.String())
			}
			if record["state"] != string(state) || record["runtimeKind"] != "hermes" || record["candidateCount"] != float64(1) {
				t.Fatalf("structured outcome = %#v", record)
			}
			if strings.Contains(output.String(), "private-user") || strings.Contains(output.String(), "provider-secret-value") {
				t.Fatalf("structured outcome leaked sensitive detail: %s", output.String())
			}
		})
	}
}

func TestRuntimeDiscoveryLogsNormalizedFailureWithoutRawError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		code      string
		timedOut  bool
		cancelled bool
	}{
		{name: "adapter", err: errors.New("provider-secret-value"), code: "RUNTIME_DISCOVERY_FAILED"},
		{name: "timeout", err: context.DeadlineExceeded, code: string(yorvaruntime.ErrorRuntimeDiscoveryTimeout), timedOut: true},
		{name: "cancelled", err: context.Canceled, code: string(yorvaruntime.ErrorRuntimeDiscoveryCancelled), cancelled: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			registry := yorvaruntime.NewRegistry()
			if err := registry.Register("hermes", yorvaruntime.Bundle{
				Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
				Discoverer: fakeDiscoverer{err: test.err},
			}); err != nil {
				t.Fatal(err)
			}
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			if _, err := NewRuntimeDiscovery(registry, logger).Detect(context.Background(), "hermes"); err == nil {
				t.Fatal("Detect() unexpectedly succeeded")
			}
			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["errorCode"] != test.code || record["timedOut"] != test.timedOut || record["cancelled"] != test.cancelled {
				t.Fatalf("normalized failure = %#v", record)
			}
			if strings.Contains(output.String(), "provider-secret-value") {
				t.Fatalf("failure log leaked raw error: %s", output.String())
			}
		})
	}
}
