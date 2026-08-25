package runtime

import (
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
	Descriptor   Descriptor
	Discoverer   Discoverer
	Models       ModelConfigurator
	Lifecycle    LifecycleManager
	Channels     ChannelManager
	Health       HealthInspector
	Logs         LogReader
	Security     SecurityAuditor
	SkillRead    SkillReader
	SkillMutate  SkillManager
	MCPRead      MCPReader
	MCPMutate    MCPManager
	BackupRead   BackupReader
	BackupMutate BackupManager
	Restore      RestoreManager
	UpgradePlan  UpgradePlanner
	Upgrade      RuntimeUpgrader
	Rollback     RuntimeRollbacker
}

type ManagementCapabilities struct {
	HealthRead    bool
	LogsRead      bool
	SecurityAudit bool
	SkillRead     bool
	SkillMutate   bool
	MCPRead       bool
	MCPMutate     bool
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
		SkillMutate:   b.SkillMutate != nil,
		MCPRead:       b.MCPRead != nil,
		MCPMutate:     b.MCPMutate != nil,
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
