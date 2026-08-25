package runtime_test

import (
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

func TestHermesManagementCapabilitiesRemainClosedWithoutQualifiedWiring(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	if err := hermes.Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	bundle, ok := registry.Get(hermes.Kind)
	if !ok {
		t.Fatal("Hermes bundle was not registered")
	}
	if got := bundle.ManagementCapabilities(); got != (yorvaruntime.ManagementCapabilities{}) {
		t.Fatalf("unqualified Hermes management capabilities = %#v, want all false", got)
	}
}
