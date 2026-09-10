package daemon

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type slowStartupDiscoverer struct{ ended chan struct{} }

func (d slowStartupDiscoverer) Detect(ctx context.Context) (yorvaruntime.Discovery, error) {
	<-ctx.Done()
	close(d.ended)
	return yorvaruntime.Discovery{}, ctx.Err()
}

func TestStartupInventoryBudgetCancelsSlowRuntimeWithoutCancellingServer(t *testing.T) {
	serverCtx, stopServer := context.WithCancel(context.Background())
	defer stopServer()
	registry := yorvaruntime.NewRegistry()
	ended := make(chan struct{})
	if err := registry.Register("openclaw", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "openclaw", Name: "OpenClaw"},
		Discoverer: slowStartupDiscoverer{ended: ended},
	}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	discovery := app.NewRuntimeDiscovery(registry, logger)
	budget, cancelBudget := context.WithTimeout(serverCtx, 25*time.Millisecond)
	defer cancelBudget()
	started := time.Now()
	reconcileStartupInstances(budget, registry, discovery, nil, logger)
	if time.Since(started) > time.Second || serverCtx.Err() != nil {
		t.Fatal("startup Runtime wait exhausted the server lifetime")
	}
	select {
	case <-ended:
	case <-time.After(time.Second):
		t.Fatal("timed-out discovery outlived its startup owner")
	}
}
