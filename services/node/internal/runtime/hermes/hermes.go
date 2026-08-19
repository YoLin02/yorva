// Package hermes owns the Hermes Runtime integration boundary.
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
		Discoverer: NewDetector(),
		Models:     NewModelManager(),
	})
}
