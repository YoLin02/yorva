package runtime

import (
	"context"
	"errors"
	"testing"
)

type dynamicHealthFixture struct{}

func (dynamicHealthFixture) InspectRuntimeHealth(context.Context, Installation) (HealthObservation, error) {
	return HealthObservation{}, nil
}

func (dynamicHealthFixture) InspectInstanceHealth(context.Context, Installation, string) (HealthObservation, error) {
	return HealthObservation{}, nil
}

type dynamicManagementFixture struct {
	err error
}

func (f dynamicManagementFixture) ResolveInstanceManagement(context.Context, Installation, string) (InstanceManagementFeatures, error) {
	if f.err != nil {
		return InstanceManagementFeatures{}, f.err
	}
	return InstanceManagementFeatures{Health: dynamicHealthFixture{}}, nil
}

func TestBundleDynamicManagementDoesNotOverclaimStaticCapability(t *testing.T) {
	bundle := Bundle{
		Descriptor:         Descriptor{Kind: "fixture", Name: "Fixture"},
		InstanceManagement: dynamicManagementFixture{},
	}
	if bundle.ManagementCapabilities().HealthRead {
		t.Fatal("unresolved Bundle advertised a dynamic Health capability")
	}
	resolved := bundle.ResolveInstanceManagement(context.Background(), Installation{RuntimeKind: "fixture"}, "profile")
	if !resolved.ManagementCapabilities().HealthRead {
		t.Fatal("qualified target did not receive Health capability")
	}

	bundle.InstanceManagement = dynamicManagementFixture{err: errors.New("unqualified")}
	if failed := bundle.ResolveInstanceManagement(context.Background(), Installation{}, "profile"); failed.ManagementCapabilities().HealthRead {
		t.Fatal("failed dynamic qualification advertised Health capability")
	}
}
