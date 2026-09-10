package daemon

import (
	"context"
	"log/slog"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func reconcileStartupInstances(ctx context.Context, registry *yorvaruntime.Registry, discovery *app.RuntimeDiscovery, instances *app.InstanceInventory, logger *slog.Logger) {
	for _, kind := range registry.Kinds() {
		if ctx.Err() != nil {
			logger.Warn("startup Runtime reconciliation budget ended")
			return
		}
		detected, err := discovery.Detect(ctx, kind)
		if err != nil {
			logger.Warn("startup Runtime discovery did not complete", "runtimeKind", kind)
			continue
		}
		if detected.State != yorvaruntime.DiscoverySupported || detected.Selected == nil {
			continue
		}
		listed, err := instances.ListInstances(ctx, string(kind))
		if err != nil {
			logger.Warn("startup Instance inventory requires recovery", "runtimeKind", kind)
			continue
		}
		if listed.Freshness != "FRESH" || listed.ErrorCode != "" {
			logger.Warn("startup Instance inventory requires recovery", "runtimeKind", kind, "errorCode", listed.ErrorCode)
		} else {
			logger.Info("startup Instance inventory reconciled", "runtimeKind", kind, "instanceCount", len(listed.Instances), "freshness", listed.Freshness)
		}
	}
}
