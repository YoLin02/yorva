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
	Descriptor Descriptor
	Discoverer Discoverer
	Models     ModelConfigurator
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
