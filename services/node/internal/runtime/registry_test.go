package runtime

import (
	"context"
	"testing"
)

type fakeDiscoverer struct{}

func (fakeDiscoverer) Detect(context.Context) (Discovery, error) {
	return Discovery{RuntimeKind: "test", State: DiscoveryNotInstalled}, nil
}

func TestRegistryRejectsDuplicateKinds(t *testing.T) {
	registry := NewRegistry()
	bundle := Bundle{Descriptor: Descriptor{Kind: "test", Name: "Test Runtime"}}
	if err := registry.Register("test", bundle); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := registry.Register("test", bundle); err == nil {
		t.Fatal("duplicate Register() succeeded")
	}
}

func TestRegistryPreservesDiscoveryCapability(t *testing.T) {
	registry := NewRegistry()
	discoverer := fakeDiscoverer{}
	bundle := Bundle{
		Descriptor: Descriptor{Kind: "test", Name: "Test Runtime"},
		Discoverer: discoverer,
	}
	if err := registry.Register("test", bundle); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	registered, ok := registry.Get("test")
	if !ok || registered.Discoverer == nil {
		t.Fatalf("registered discovery capability = %#v, want non-nil", registered.Discoverer)
	}
}

func TestRegistryRejectsDescriptorMismatch(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register("one", Bundle{Descriptor: Descriptor{Kind: "two", Name: "Two"}})
	if err == nil {
		t.Fatal("mismatched Register() succeeded")
	}
}
