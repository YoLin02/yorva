package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
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
