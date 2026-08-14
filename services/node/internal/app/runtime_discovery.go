package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var ErrRuntimeKindNotFound = errors.New("Runtime kind not found")

const runtimeDiscoveryTimeout = 10 * time.Second

type RuntimeDiscovery struct {
	registry *yorvaruntime.Registry
	timeout  time.Duration
	logger   *slog.Logger
}

func NewRuntimeDiscovery(registry *yorvaruntime.Registry, logger *slog.Logger) *RuntimeDiscovery {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &RuntimeDiscovery{registry: registry, timeout: runtimeDiscoveryTimeout, logger: logger}
}

func (d *RuntimeDiscovery) Detect(ctx context.Context, kind yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	started := time.Now()
	bundle, ok := d.registry.Get(kind)
	if !ok || bundle.Discoverer == nil {
		d.logFailure(kind, ErrRuntimeKindNotFound, started)
		return yorvaruntime.Discovery{}, ErrRuntimeKindNotFound
	}
	discoveryCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	discovery, err := bundle.Discoverer.Detect(discoveryCtx)
	if err != nil {
		d.logFailure(kind, err, started)
		return yorvaruntime.Discovery{}, err
	}
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
	return discovery, nil
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
