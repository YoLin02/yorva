package hermes

import (
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestRegisterAddsDescriptorAndDiscoveryCapability(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	bundle, ok := registry.Get(Kind)
	if !ok {
		t.Fatal("Hermes descriptor was not registered")
	}
	if bundle.Descriptor.Name != "Hermes Agent" || len(registry.Kinds()) != 1 {
		t.Fatalf("unexpected Hermes registration: %#v", bundle)
	}
	if bundle.Discoverer == nil {
		t.Fatal("Hermes discovery capability was not registered")
	}
}
