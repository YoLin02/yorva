package app

import (
	"context"
	"errors"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var ErrRuntimeKindNotFound = errors.New("Runtime kind not found")

const runtimeDiscoveryTimeout = 10 * time.Second

type RuntimeDiscovery struct {
	registry *yorvaruntime.Registry
	timeout  time.Duration
}

func NewRuntimeDiscovery(registry *yorvaruntime.Registry) *RuntimeDiscovery {
	return &RuntimeDiscovery{registry: registry, timeout: runtimeDiscoveryTimeout}
}

func (d *RuntimeDiscovery) Detect(ctx context.Context, kind yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	bundle, ok := d.registry.Get(kind)
	if !ok || bundle.Discoverer == nil {
		return yorvaruntime.Discovery{}, ErrRuntimeKindNotFound
	}
	discoveryCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	return bundle.Discoverer.Detect(discoveryCtx)
}
