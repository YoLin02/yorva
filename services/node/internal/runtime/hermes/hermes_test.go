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
	if bundle.Discoverer == nil || bundle.Models == nil {
		t.Fatal("Hermes discovery and model capabilities were not registered")
	}
	if bundle.SkillProjection == nil {
		t.Fatal("Hermes managed Skill projection was not registered")
	}
	capabilities := bundle.NativeSkillCapabilities
	if !capabilities.Inventory.Supported || capabilities.Inventory.Reason != "dynamic_instance_readback" {
		t.Fatalf("native inventory capability = %#v", capabilities.Inventory)
	}
	for name, capability := range map[string]yorvaruntime.NativeSkillCapability{
		"install": capabilities.NativeInstall, "update": capabilities.NativeUpdate,
		"remove": capabilities.NativeRemove, "enable-disable": capabilities.NativeEnableDisable,
		"profile-binding": capabilities.NativeProfileBinding,
	} {
		if capability.Supported || capability.Reason != "deferred_upstream" {
			t.Fatalf("native %s capability = %#v", name, capability)
		}
	}
}
