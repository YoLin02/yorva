package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var ErrRuntimeKindNotFound = errors.New("Runtime kind not found")

const (
	runtimeDiscoveryTimeout  = 35 * time.Second
	runtimeDiscoveryCacheTTL = 5 * time.Second
)

type runtimeDiscoveryFlight struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	result  yorvaruntime.Discovery
	err     error
}

type cachedRuntimeDiscovery struct {
	result    yorvaruntime.Discovery
	expiresAt time.Time
}

type RuntimeDiscovery struct {
	registry *yorvaruntime.Registry
	timeout  time.Duration
	logger   *slog.Logger
	now      func() time.Time
	cacheTTL time.Duration
	mu       sync.Mutex
	flights  map[yorvaruntime.Kind]*runtimeDiscoveryFlight
	cache    map[yorvaruntime.Kind]cachedRuntimeDiscovery
}

func NewRuntimeDiscovery(registry *yorvaruntime.Registry, logger *slog.Logger) *RuntimeDiscovery {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &RuntimeDiscovery{
		registry: registry, timeout: runtimeDiscoveryTimeout, logger: logger,
		now: time.Now, cacheTTL: runtimeDiscoveryCacheTTL,
		flights: make(map[yorvaruntime.Kind]*runtimeDiscoveryFlight),
		cache:   make(map[yorvaruntime.Kind]cachedRuntimeDiscovery),
	}
}

func (d *RuntimeDiscovery) Detect(ctx context.Context, kind yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	bundle, ok := d.registry.Get(kind)
	if !ok || bundle.Discoverer == nil {
		d.logFailure(kind, ErrRuntimeKindNotFound, d.now())
		return yorvaruntime.Discovery{}, ErrRuntimeKindNotFound
	}
	if err := ctx.Err(); err != nil {
		return yorvaruntime.Discovery{}, err
	}

	d.mu.Lock()
	now := d.now()
	if cached, exists := d.cache[kind]; exists && now.Before(cached.expiresAt) {
		result := cloneRuntimeDiscovery(cached.result)
		d.mu.Unlock()
		return result, nil
	}
	flight, exists := d.flights[kind]
	if exists && flight.waiters > 0 {
		flight.waiters++
		d.mu.Unlock()
		return d.waitForFlight(ctx, flight)
	}
	flightCtx, cancel := context.WithTimeout(context.Background(), d.timeout)
	flight = &runtimeDiscoveryFlight{done: make(chan struct{}), cancel: cancel, waiters: 1}
	d.flights[kind] = flight
	d.mu.Unlock()

	go d.runFlight(flightCtx, kind, bundle.Discoverer, flight)
	return d.waitForFlight(ctx, flight)
}

func (d *RuntimeDiscovery) runFlight(ctx context.Context, kind yorvaruntime.Kind, discoverer yorvaruntime.Discoverer, flight *runtimeDiscoveryFlight) {
	started := d.now()
	discovery, err := discoverer.Detect(ctx)
	if err != nil {
		d.logFailure(kind, err, started)
	} else {
		d.logSuccess(kind, discovery, started)
	}

	d.mu.Lock()
	flight.result = cloneRuntimeDiscovery(discovery)
	flight.err = err
	current := d.flights[kind]
	if current == flight {
		delete(d.flights, kind)
	}
	if current == flight && flight.waiters > 0 && err == nil && discovery.State == yorvaruntime.DiscoverySupported {
		d.cache[kind] = cachedRuntimeDiscovery{result: cloneRuntimeDiscovery(discovery), expiresAt: d.now().Add(d.cacheTTL)}
	}
	close(flight.done)
	d.mu.Unlock()
	flight.cancel()
}

func (d *RuntimeDiscovery) waitForFlight(ctx context.Context, flight *runtimeDiscoveryFlight) (yorvaruntime.Discovery, error) {
	select {
	case <-flight.done:
		return cloneRuntimeDiscovery(flight.result), flight.err
	case <-ctx.Done():
		d.mu.Lock()
		select {
		case <-flight.done:
			d.mu.Unlock()
			return cloneRuntimeDiscovery(flight.result), flight.err
		default:
			flight.waiters--
			if flight.waiters == 0 {
				flight.cancel()
			}
			d.mu.Unlock()
			return yorvaruntime.Discovery{}, ctx.Err()
		}
	}
}

func cloneRuntimeDiscovery(discovery yorvaruntime.Discovery) yorvaruntime.Discovery {
	result := discovery
	result.Candidates = append([]yorvaruntime.Candidate(nil), discovery.Candidates...)
	result.Warnings = append([]yorvaruntime.Warning(nil), discovery.Warnings...)
	if discovery.Selected != nil {
		selected := *discovery.Selected
		result.Selected = &selected
	}
	return result
}

func (d *RuntimeDiscovery) logSuccess(kind yorvaruntime.Kind, discovery yorvaruntime.Discovery, started time.Time) {
	d.logger.Info("runtime discovery completed",
		"runtimeKind", kind,
		"state", discovery.State,
		"errorCode", discovery.ErrorCode,
		"candidateCount", len(discovery.Candidates),
		"warningCount", len(discovery.Warnings),
		"durationMs", time.Since(started).Milliseconds(),
		"timedOut", discovery.State == yorvaruntime.DiscoveryTimedOut,
		"cancelled", false,
	)
}

func (d *RuntimeDiscovery) logFailure(kind yorvaruntime.Kind, err error, started time.Time) {
	result := "FAILED"
	errorCode := "RUNTIME_DISCOVERY_FAILED"
	timedOut := errors.Is(err, context.DeadlineExceeded)
	cancelled := errors.Is(err, context.Canceled)
	if errors.Is(err, ErrRuntimeKindNotFound) {
		result = "KIND_NOT_FOUND"
		errorCode = "RUNTIME_KIND_NOT_FOUND"
	} else if timedOut {
		result = string(yorvaruntime.DiscoveryTimedOut)
		errorCode = string(yorvaruntime.ErrorRuntimeDiscoveryTimeout)
	} else if cancelled {
		result = "CANCELLED"
		errorCode = string(yorvaruntime.ErrorRuntimeDiscoveryCancelled)
	}
	d.logger.Warn("runtime discovery failed",
		"runtimeKind", kind,
		"state", result,
		"errorCode", errorCode,
		"candidateCount", 0,
		"warningCount", 0,
		"durationMs", time.Since(started).Milliseconds(),
		"timedOut", timedOut,
		"cancelled", cancelled,
	)
}
