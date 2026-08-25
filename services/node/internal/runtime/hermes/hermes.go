// Package hermes owns the Hermes Runtime integration boundary.
package hermes

import (
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/skillsmanagement"
)

const Kind yorvaruntime.Kind = "hermes"

func Register(registry *yorvaruntime.Registry) error {
	return registry.Register(Kind, yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{
			Kind:        Kind,
			Name:        "Hermes Agent",
			Description: "Hermes Agent Runtime",
		},
		Discoverer:      NewDetector(),
		Models:          NewModelManager(),
		Lifecycle:       NewLifecycleManager(),
		Channels:        NewChannelManager(),
		UpgradePlan:     NewUpgradePlanner(),
		SkillProjection: skillsmanagement.NewRuntimeProjector(),
		NativeSkillCapabilities: yorvaruntime.NativeSkillCapabilities{
			Inventory:            yorvaruntime.NativeSkillCapability{Supported: true, Reason: "dynamic_instance_readback"},
			NativeInstall:        yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeUpdate:         yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeRemove:         yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeEnableDisable:  yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeProfileBinding: yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		},
		InstanceManagement: NewAPIManagementReader(),
	})
}
