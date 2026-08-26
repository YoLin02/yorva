// Package hermes owns the Hermes Runtime integration boundary.
package hermes

import (
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/skillsmanagement"
)

const Kind yorvaruntime.Kind = "hermes"

func Register(registry *yorvaruntime.Registry) error {
	return RegisterConfigured(registry, ManagementBindings{})
}

type ManagementBindings struct {
	MCPRead      yorvaruntime.MCPReader
	MCPMutate    yorvaruntime.MCPManager
	BackupRead   yorvaruntime.BackupReader
	BackupMutate yorvaruntime.BackupManager
	Restore      yorvaruntime.RestoreManager
	Upgrade      yorvaruntime.RuntimeUpgrader
	Rollback     yorvaruntime.RuntimeRollbacker
}

func RegisterConfigured(registry *yorvaruntime.Registry, bindings ManagementBindings) error {
	if bindings.MCPRead == nil {
		manager := NewProfileMCPManager()
		bindings.MCPRead = manager
		bindings.MCPMutate = manager
	}
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
		MCPRead:         bindings.MCPRead,
		MCPMutate:       bindings.MCPMutate,
		BackupRead:      bindings.BackupRead,
		BackupMutate:    bindings.BackupMutate,
		Restore:         bindings.Restore,
		Upgrade:         bindings.Upgrade,
		Rollback:        bindings.Rollback,
		SkillProjection: skillsmanagement.NewRuntimeProjector(),
		NativeSkillCapabilities: yorvaruntime.NativeSkillCapabilities{
			Inventory:            yorvaruntime.NativeSkillCapability{Supported: true, Reason: "dynamic_instance_readback"},
			NativeInstall:        yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeUpdate:         yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeRemove:         yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeEnableDisable:  yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
			NativeProfileBinding: yorvaruntime.NativeSkillCapability{Supported: false, Reason: "deferred_upstream"},
		},
		InstanceManagement: newManagementResolverWithMCP(bindings.MCPRead),
	})
}
