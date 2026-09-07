package app

import (
	"context"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestNodeRecoveryRequiresFreshAuthoritativeInventory(t *testing.T) {
	ctx := context.Background()
	inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
	ready, err := inventory.CheckNodeRecovery(ctx)
	if err != nil || !ready.Ready || ready.ErrorCode != "" {
		t.Fatalf("initial recovery = %#v, %v", ready, err)
	}
	source.setErr(ErrInstanceOutputUnrecognized)
	failed, err := inventory.CheckNodeRecovery(ctx)
	if err != nil || failed.Ready || failed.ErrorCode != yorvaruntime.ErrorInstanceOutputUnrecognized {
		t.Fatalf("failed readback must not pass recovery: %#v, %v", failed, err)
	}
	listed, err := inventory.ListInstances(ctx, "hermes")
	if err != nil || listed.Freshness != "UNKNOWN" || len(listed.Instances) != 1 || listed.Instances[0].Availability != instance.Unknown {
		t.Fatalf("management inventory must remain queryable: %#v, %v", listed, err)
	}
	source.setErr(nil)
	recovered, err := inventory.CheckNodeRecovery(ctx)
	if err != nil || !recovered.Ready || recovered.ErrorCode != "" {
		t.Fatalf("fresh retry did not recover: %#v, %v", recovered, err)
	}
}

func TestNodeRecoveryDoesNotRequireRuntimeForFreshYorva(t *testing.T) {
	for _, previouslyAccepted := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "missing-accepted-runtime"}[previouslyAccepted], func(t *testing.T) {
			ctx := context.Background()
			inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil)
			if previouslyAccepted {
				if _, err := inventory.ListInstances(ctx, "hermes"); err != nil {
					t.Fatal(err)
				}
			}
			setRecoveryDiscovery(t, inventory, yorvaruntime.DiscoveryNotInstalled)
			result, err := inventory.CheckNodeRecovery(ctx)
			if err != nil || result.Ready == previouslyAccepted {
				t.Fatalf("recovery = %#v, %v", result, err)
			}
			if previouslyAccepted && result.ErrorCode != yorvaruntime.ErrorRuntimeNotInstalled {
				t.Fatalf("missing accepted Runtime code = %s", result.ErrorCode)
			}
		})
	}
}

func TestNodeRecoveryRejectsUnsupportedAndFailedDiscovery(t *testing.T) {
	for _, state := range []yorvaruntime.DiscoveryState{
		yorvaruntime.DiscoveryUnsupported, yorvaruntime.DiscoveryBrokenExecutable,
		yorvaruntime.DiscoveryMalformedVersion, yorvaruntime.DiscoveryTimedOut, yorvaruntime.DiscoveryAmbiguous,
	} {
		t.Run(string(state), func(t *testing.T) {
			inventory, _ := newTestInventory(t, nil, nil)
			setRecoveryDiscovery(t, inventory, state)
			result, err := inventory.CheckNodeRecovery(context.Background())
			if err != nil || result.Ready || result.ErrorCode == "" {
				t.Fatalf("failed discovery passed recovery: %#v, %v", result, err)
			}
		})
	}
}

func setRecoveryDiscovery(t *testing.T, inventory *InstanceInventory, state yorvaruntime.DiscoveryState) {
	t.Helper()
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{RuntimeKind: "hermes", State: state}},
	}); err != nil {
		t.Fatal(err)
	}
	inventory.discovery = NewRuntimeDiscovery(registry, nil)
}
