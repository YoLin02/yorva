package runtime

import (
	"context"
	"errors"
	"sort"
	"sync"
)

type Kind string

type Descriptor struct {
	Kind        Kind
	Name        string
	Description string
}

type Bundle struct {
	Descriptor  Descriptor
	Discoverer  Discoverer
	Instances   InstanceManager
	Models      ModelConfigurator
	Lifecycle   LifecycleManager
	Channels    ChannelManager
	Health      HealthInspector
	Logs        LogReader
	Security    SecurityAuditor
	SkillRead   SkillReader
	SkillMutate SkillManager
	// SkillProjection is the YORVA-managed lifecycle. SkillMutate remains the
	// independent Runtime-native mutation surface.
	SkillProjection         SkillProjector
	NativeSkillCapabilities NativeSkillCapabilities
	MCPRead                 MCPReader
	MCPMutate               MCPManager
	BackupRead              BackupReader
	BackupMutate            BackupManager
	Restore                 RestoreManager
	UpgradePlan             UpgradePlanner
	Upgrade                 RuntimeUpgrader
	Rollback                RuntimeRollbacker
	// InstanceManagement resolves management readers whose availability depends
	// on exact Runtime/Profile state. The registered Bundle keeps these fields
	// nil so a version-wide static capability cannot over-claim support.
	InstanceManagement InstanceManagementResolver
}

type InstanceManagementFeatures struct {
	// DisableLifecycle restricts the registered lifecycle for an externally
	// owned or otherwise unqualified target; it can never enable a mutation.
	DisableLifecycle bool
	Health           HealthInspector
	Logs             LogReader
	Security         SecurityAuditor
	SkillRead        SkillReader
	MCPRead          MCPReader
}

// InstanceManagementResolver may only add read capabilities for one already
// resolved Installation/Profile and restrict lifecycle availability. Mutating
// capabilities remain compile-time wiring and cannot be enabled by a read.
type InstanceManagementResolver interface {
	ResolveInstanceManagement(context.Context, Installation, string) (InstanceManagementFeatures, error)
}

func (b Bundle) ResolveInstanceManagement(ctx context.Context, installation Installation, nativeID string) Bundle {
	if b.InstanceManagement == nil {
		return b
	}
	features, err := b.InstanceManagement.ResolveInstanceManagement(ctx, installation, nativeID)
	if err != nil {
		return b
	}
	resolved := b
	if features.DisableLifecycle {
		resolved.Lifecycle = nil
	}
	resolved.Health = features.Health
	resolved.Logs = features.Logs
	resolved.Security = features.Security
	resolved.SkillRead = features.SkillRead
	resolved.MCPRead = features.MCPRead
	return resolved
}

type ManagementCapabilities struct {
	HealthRead    bool
	LogsRead      bool
	SecurityAudit bool
	SkillRead     bool
	SkillMutate   bool
	MCPRead       bool
	MCPMutate     bool
	MCPTest       bool
	BackupRead    bool
	BackupMutate  bool
	Restore       bool
	UpgradePlan   bool
	Upgrade       bool
	Rollback      bool
}

// ManagementCapabilities derives availability from compile-time wiring. An
// absent feature contract is capability-false; registration alone never
// advertises an unqualified management surface.
func (b Bundle) ManagementCapabilities() ManagementCapabilities {
	return ManagementCapabilities{
		HealthRead:    b.Health != nil,
		LogsRead:      b.Logs != nil,
		SecurityAudit: b.Security != nil,
		SkillRead:     b.SkillRead != nil,
		SkillMutate:   b.SkillProjection != nil,
		MCPRead:       b.MCPRead != nil,
		MCPMutate:     b.MCPMutate != nil,
		MCPTest:       b.MCPMutate != nil,
		BackupRead:    b.BackupRead != nil,
		BackupMutate:  b.BackupMutate != nil,
		Restore:       b.Restore != nil,
		UpgradePlan:   b.UpgradePlan != nil,
		Upgrade:       b.Upgrade != nil,
		Rollback:      b.Rollback != nil,
	}
}

type Registry struct {
	mu      sync.RWMutex
	bundles map[Kind]Bundle
}

func NewRegistry() *Registry {
	return &Registry{bundles: make(map[Kind]Bundle)}
}

func (r *Registry) Register(kind Kind, bundle Bundle) error {
	if kind == "" || bundle.Descriptor.Kind != kind || bundle.Descriptor.Name == "" {
		return errors.New("invalid Runtime registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.bundles[kind]; exists {
		return errors.New("Runtime kind already registered")
	}
	r.bundles[kind] = bundle
	return nil
}

func (r *Registry) Get(kind Kind) (Bundle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bundle, ok := r.bundles[kind]
	return bundle, ok
}

func (r *Registry) Kinds() []Kind {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]Kind, 0, len(r.bundles))
	for kind := range r.bundles {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}
