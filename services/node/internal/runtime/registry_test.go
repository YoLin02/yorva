package runtime

import "testing"

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

func TestRegistryRejectsDescriptorMismatch(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register("one", Bundle{Descriptor: Descriptor{Kind: "two", Name: "Two"}})
	if err == nil {
		t.Fatal("mismatched Register() succeeded")
	}
}
