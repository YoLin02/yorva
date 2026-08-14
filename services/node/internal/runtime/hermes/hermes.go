// Package hermes owns the Phase 1 static Hermes Runtime registration.
//
// It intentionally performs no discovery and contains no Hermes CLI, PATH,
// Python, configuration, or filesystem integration.
package hermes

import yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"

const Kind yorvaruntime.Kind = "hermes"

func Register(registry *yorvaruntime.Registry) error {
	return registry.Register(Kind, yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{
			Kind:        Kind,
			Name:        "Hermes Agent",
			Description: "Hermes Agent Runtime",
		},
	})
}
