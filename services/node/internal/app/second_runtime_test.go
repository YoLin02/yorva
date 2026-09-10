package app

import (
	"context"
	"errors"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func addSecondRuntime(t *testing.T, inventory *InstanceInventory, profiles []ProfileSnapshot, lifecycle yorvaruntime.LifecycleManager) *fakeProfileSource {
	t.Helper()
	source := &fakeProfileSource{profiles: profiles}
	source.mutator = &fakeMutator{source: source}
	if err := inventory.discovery.registry.Register("openclaw", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "openclaw", Name: "OpenClaw"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{
			RuntimeKind: "openclaw", State: yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: `C:\openclaw\openclaw.mjs`, Version: "2026.9.3", State: yorvaruntime.DiscoverySupported},
		}},
		Instances: source, Lifecycle: lifecycle,
	}); err != nil {
		t.Fatal(err)
	}
	return source
}

func TestTwoRuntimesKeepSameNameIdentityAndMutationOwnership(t *testing.T) {
	ctx := context.Background()
	hermesLifecycle := &fakeLifecycleManager{state: yorvaruntime.LifecycleStopped}
	clawLifecycle := &fakeLifecycleManager{state: yorvaruntime.LifecycleStopped}
	inventory, hermesSource := newTestInventoryWithLifecycle(t, []ProfileSnapshot{{NativeID: "coder"}}, nil, hermesLifecycle)
	hermesMutator := &fakeMutator{source: hermesSource}
	inventory.WithMutator(hermesMutator)
	claw := addSecondRuntime(t, inventory, []ProfileSnapshot{{NativeID: "coder"}}, clawLifecycle)
	hermesList, err := inventory.ListInstances(ctx, "hermes")
	if err != nil {
		t.Fatal(err)
	}
	clawList, err := inventory.ListInstances(ctx, "openclaw")
	if err != nil {
		t.Fatal(err)
	}
	if hermesList.RuntimeInstallationID == clawList.RuntimeInstallationID || hermesList.Instances[0].InstanceID == clawList.Instances[0].InstanceID {
		t.Fatal("native names collided across Runtimes")
	}
	started, err := inventory.StartLifecycle(ctx, clawList.Instances[0].InstanceID, LifecycleStart, "claw-start")
	if err != nil {
		t.Fatal(err)
	}
	waitInstanceOperation(t, inventory, started.Operation.ID, operation.StatusSucceeded)
	clawStatus, _ := clawLifecycle.Status(ctx, yorvaruntime.LifecycleInstallation{}, "coder")
	hermesStatus, _ := hermesLifecycle.Status(ctx, yorvaruntime.LifecycleInstallation{}, "coder")
	if clawStatus.State != yorvaruntime.LifecycleRunning || hermesStatus.State != yorvaruntime.LifecycleStopped {
		t.Fatal("lifecycle reached the wrong Runtime")
	}
	created, err := inventory.StartCreate(ctx, "openclaw", "second", "same-request-key")
	if err != nil {
		t.Fatal(err)
	}
	waitInstanceOperation(t, inventory, created.Operation.ID, operation.StatusSucceeded)
	if _, err := inventory.StartCreate(ctx, "hermes", "second", "same-request-key"); !errors.Is(err, ErrInstanceConflict) {
		t.Fatalf("cross-Runtime idempotency = %v", err)
	}
	if calls, _ := hermesMutator.snapshot(); calls != 0 {
		t.Fatal("Hermes was mutated")
	}
	if calls, _ := claw.mutator.(*fakeMutator).snapshot(); calls != 1 {
		t.Fatalf("OpenClaw create calls = %d", calls)
	}
}

func TestSecondRuntimeAbsentCapabilitiesCannotFallBackToHermes(t *testing.T) {
	ctx := context.Background()
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := &fakeModelConfigurator{}
	registerTestModels(t, inventory, models)
	addSecondRuntime(t, inventory, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	listed, err := inventory.ListInstances(ctx, "openclaw")
	if err != nil {
		t.Fatal(err)
	}
	id := listed.Instances[0].InstanceID
	if listed.Capabilities.Models || listed.Capabilities.Channels || listed.Capabilities.Lifecycle {
		t.Fatal("unsupported capabilities advertised")
	}
	if _, err := inventory.GetModelConfiguration(ctx, id); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("models = %v", err)
	}
	if _, err := inventory.ListModelProviderPresets(ctx, "openclaw"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("presets = %v", err)
	}
	if _, err := inventory.GetLifecycle(ctx, id); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("lifecycle = %v", err)
	}
	if _, _, _, err := inventory.resolveChannelTarget(ctx, id); !errors.Is(err, ErrChannelNotSupported) {
		t.Fatalf("channels = %v", err)
	}
	if _, _, _, err := inventory.runtimeModelContext(ctx, "openclaw"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("shared models = %v", err)
	}
}

func TestSecondRuntimeFailurePreservesUnknownAndDoesNotMaskRecovery(t *testing.T) {
	ctx := context.Background()
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	claw := addSecondRuntime(t, inventory, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	if recovery, err := inventory.CheckNodeRecovery(ctx); err != nil || !recovery.Ready {
		t.Fatalf("initial recovery = %#v %v", recovery, err)
	}
	claw.setErr(ErrInstanceOutputUnrecognized)
	if recovery, err := inventory.CheckNodeRecovery(ctx); err != nil || recovery.Ready || recovery.ErrorCode != yorvaruntime.ErrorInstanceOutputUnrecognized {
		t.Fatalf("failed second Runtime = %#v %v", recovery, err)
	}
	listed, err := inventory.ListInstances(ctx, "openclaw")
	if err != nil || listed.Freshness != "UNKNOWN" || listed.Instances[0].Availability != instance.Unknown {
		t.Fatalf("unknown = %#v %v", listed, err)
	}
	if _, err := inventory.StartCreate(ctx, "openclaw", "second", "unknown-create"); !errors.Is(err, ErrInstanceOutputUnrecognized) {
		t.Fatalf("unknown mutation = %v", err)
	}
	if calls, _ := claw.mutator.(*fakeMutator).snapshot(); calls != 0 {
		t.Fatal("mutation proceeded with unknown inventory")
	}
	if listed, err := inventory.ListInstances(ctx, "hermes"); err != nil || listed.Freshness != "FRESH" || listed.Instances[0].Availability != instance.Available {
		t.Fatalf("healthy Runtime unavailable = %#v %v", listed, err)
	}
	claw.setErr(nil)
	if recovery, err := inventory.CheckNodeRecovery(ctx); err != nil || !recovery.Ready {
		t.Fatalf("retry = %#v %v", recovery, err)
	}
}

func TestSecondRuntimeExistingProfileRemainsProtected(t *testing.T) {
	ctx := context.Background()
	inventory, _ := newTestInventory(t, nil, nil)
	addSecondRuntime(t, inventory, []ProfileSnapshot{{NativeID: "personal", Protected: true}}, nil)
	listed, err := inventory.ListInstances(ctx, "openclaw")
	if err != nil {
		t.Fatal(err)
	}
	item := listed.Instances[0]
	if item.Default || !item.Protected {
		t.Fatalf("external protection = %#v", item)
	}
	if _, err := inventory.StartDelete(ctx, item.InstanceID, "personal", "protected-delete"); !errors.Is(err, ErrInstanceProtected) {
		t.Fatalf("delete = %v", err)
	}
}

type protectedLifecycleResolver struct{}

func (protectedLifecycleResolver) ResolveInstanceManagement(context.Context, yorvaruntime.Installation, string) (yorvaruntime.InstanceManagementFeatures, error) {
	return yorvaruntime.InstanceManagementFeatures{DisableLifecycle: true}, nil
}

func TestInstanceLifecycleRestrictionAppliesToDirectMutationRequests(t *testing.T) {
	ctx := context.Background()
	inventory, _ := newTestInventory(t, nil, nil)
	lifecycle := &fakeLifecycleManager{state: yorvaruntime.LifecycleStopped}
	source := &fakeProfileSource{profiles: []ProfileSnapshot{{NativeID: "personal", Protected: true}}}
	if err := inventory.discovery.registry.Register("openclaw", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "openclaw", Name: "OpenClaw"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{
			RuntimeKind: "openclaw", State: yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: `C:\openclaw\openclaw.mjs`, Version: "2026.9.3", State: yorvaruntime.DiscoverySupported},
		}},
		Instances: source, Lifecycle: lifecycle, InstanceManagement: protectedLifecycleResolver{},
	}); err != nil {
		t.Fatal(err)
	}
	listed, err := inventory.ListInstances(ctx, "openclaw")
	if err != nil || len(listed.Instances) != 1 {
		t.Fatalf("list = %#v, %v", listed, err)
	}
	item := listed.Instances[0]
	if item.Capabilities.Lifecycle {
		t.Fatal("protected target advertises lifecycle")
	}
	for _, action := range []LifecycleAction{LifecycleStart, LifecycleStop, LifecycleRestart} {
		if _, err := inventory.StartLifecycle(ctx, item.InstanceID, action, "protected-"+string(action)); !errors.Is(err, ErrManagementCapabilityUnsupported) {
			t.Fatalf("action %s = %v", action, err)
		}
	}
	state, _ := lifecycle.Status(ctx, yorvaruntime.LifecycleInstallation{}, "personal")
	if state.State != yorvaruntime.LifecycleStopped {
		t.Fatal("protected target was mutated")
	}
}
