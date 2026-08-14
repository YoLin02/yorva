package app

import (
	"context"
	"errors"
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

	got, err := NewRuntimeDiscovery(registry).Detect(context.Background(), "hermes")
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

	service := NewRuntimeDiscovery(registry)
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

	if _, err := NewRuntimeDiscovery(registry).Detect(context.Background(), "hermes"); !errors.Is(err, wantErr) {
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
	service := NewRuntimeDiscovery(registry)
	service.timeout = time.Millisecond

	if _, err := service.Detect(context.Background(), "hermes"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Detect() error = %v, want context.DeadlineExceeded", err)
	}
	if got := <-finished; !errors.Is(got, context.DeadlineExceeded) {
		t.Fatalf("adapter context error = %v, want context.DeadlineExceeded", got)
	}
}
