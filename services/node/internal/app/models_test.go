package app

import (
	"context"
	"errors"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeModelConfigurator struct {
	readNativeID  string
	applyNativeID string
	applyPresetID string
	applyModelID  string
	config        yorvaruntime.ModelConfiguration
	err           error
	credential    yorvaruntime.ModelCredentialStatus
	secret        []byte
	deleted       string
}

func (f *fakeModelConfigurator) ModelCredentialStatus(_ context.Context, _ yorvaruntime.ModelInstallation, nativeID, presetID string) (yorvaruntime.ModelCredentialStatus, error) {
	f.readNativeID = nativeID
	if f.credential.ProviderPresetID == "" {
		f.credential.ProviderPresetID = presetID
	}
	return f.credential, f.err
}

func (f *fakeModelConfigurator) SetModelCredential(_ context.Context, _ yorvaruntime.ModelInstallation, nativeID, presetID string, secret []byte) (yorvaruntime.ModelCredentialStatus, error) {
	f.applyNativeID, f.applyPresetID = nativeID, presetID
	f.secret = append([]byte(nil), secret...)
	if f.err != nil {
		return yorvaruntime.ModelCredentialStatus{}, f.err
	}
	f.credential = yorvaruntime.ModelCredentialStatus{ProviderPresetID: presetID, Configured: true}
	return f.credential, nil
}

func (f *fakeModelConfigurator) DeleteModelCredential(_ context.Context, _ yorvaruntime.ModelInstallation, nativeID, presetID string) (yorvaruntime.ModelCredentialStatus, error) {
	f.applyNativeID, f.deleted = nativeID, presetID
	if f.err != nil {
		return yorvaruntime.ModelCredentialStatus{}, f.err
	}
	f.credential = yorvaruntime.ModelCredentialStatus{ProviderPresetID: presetID}
	return f.credential, nil
}

func (f *fakeModelConfigurator) ListProviderPresets() []yorvaruntime.ModelProviderPreset {
	return []yorvaruntime.ModelProviderPreset{{ID: "deepseek", DisplayName: "DeepSeek", Region: yorvaruntime.ModelRegionChina, RecommendedModels: []string{"deepseek-v4-pro"}}}
}

func (f *fakeModelConfigurator) ReadModelConfig(_ context.Context, _ yorvaruntime.ModelInstallation, nativeID string) (yorvaruntime.ModelConfiguration, error) {
	f.readNativeID = nativeID
	return f.config, f.err
}

func (f *fakeModelConfigurator) ApplyModelConfig(_ context.Context, _ yorvaruntime.ModelInstallation, nativeID, presetID, modelID string) (yorvaruntime.ModelConfiguration, error) {
	f.applyNativeID, f.applyPresetID, f.applyModelID = nativeID, presetID, modelID
	return f.config, f.err
}

func TestModelUseCasesResolveStableInstanceToNativeProfile(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := &fakeModelConfigurator{config: yorvaruntime.ModelConfiguration{
		ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true,
	}}
	registerTestModels(t, inventory, models)
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	instanceID := listed.Instances[0].InstanceID
	presets, err := inventory.ListModelProviderPresets(context.Background())
	if err != nil || len(presets) != 1 || presets[0].ID != "deepseek" {
		t.Fatalf("presets = %#v, %v", presets, err)
	}
	got, err := inventory.GetModelConfiguration(context.Background(), instanceID)
	if err != nil || got.State != yorvaruntime.ModelConfigurationConfigured || models.readNativeID != "coder" {
		t.Fatalf("get = %#v native=%q err=%v", got, models.readNativeID, err)
	}
	got, err = inventory.PatchModelConfiguration(context.Background(), instanceID, "deepseek", "deepseek-v4-pro")
	if err != nil || models.applyNativeID != "coder" || models.applyPresetID != "deepseek" || models.applyModelID != "deepseek-v4-pro" {
		t.Fatalf("patch = %#v fake=%#v err=%v", got, models, err)
	}
	if models.applyNativeID == instanceID {
		t.Fatal("public instanceId was passed as native profile id")
	}
}

func TestModelUseCasesBlockMissingUnknownAndActiveMutation(t *testing.T) {
	for _, availability := range []instance.Availability{instance.Missing, instance.Unknown} {
		t.Run(string(availability), func(t *testing.T) {
			inventory, source := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
			models := &fakeModelConfigurator{}
			registerTestModels(t, inventory, models)
			listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
			if err != nil {
				t.Fatal(err)
			}
			instanceID := listed.Instances[0].InstanceID
			if availability == instance.Missing {
				source.setProfiles(nil)
			} else {
				source.setErr(ErrInstanceQueryFailed)
			}
			if _, err := inventory.PatchModelConfiguration(context.Background(), instanceID, "deepseek", "model"); !errors.Is(err, ErrInstanceNotAvailable) {
				t.Fatalf("availability %s error = %v", availability, err)
			}
			if models.applyNativeID != "" {
				t.Fatalf("adapter called for %s", availability)
			}
		})
	}
}

func TestModelUseCasePreservesSafeObservedStateOnAdapterError(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := &fakeModelConfigurator{
		config: yorvaruntime.ModelConfiguration{ProviderPresetID: "deepseek", ModelID: "old-model", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true},
		err:    yorvaruntime.ErrModelConfigIncomplete,
	}
	registerTestModels(t, inventory, models)
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inventory.PatchModelConfiguration(context.Background(), listed.Instances[0].InstanceID, "deepseek", "new-model")
	if !errors.Is(err, yorvaruntime.ErrModelConfigIncomplete) || got.ModelID != "old-model" || got.ObservedAt.IsZero() {
		t.Fatalf("observed state = %#v, %v", got, err)
	}
}

func TestCredentialUseCasesCoordinateSaveStatusAndDelete(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := &fakeModelConfigurator{
		config:     yorvaruntime.ModelConfiguration{ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true},
		credential: yorvaruntime.ModelCredentialStatus{ProviderPresetID: "deepseek", Configured: true},
	}
	registerTestModels(t, inventory, models)
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	instanceID := listed.Instances[0].InstanceID
	metadata, err := inventory.GetModelCredential(context.Background(), instanceID)
	if err != nil || !metadata.Configured || metadata.ProviderPresetID != "deepseek" || metadata.ObservedAt.IsZero() {
		t.Fatalf("metadata = %#v %v", metadata, err)
	}
	secret := []byte("batch-three-secret")
	configuration, err := inventory.SaveModelCredentialConfiguration(context.Background(), instanceID, "deepseek", "deepseek-v4-pro", secret)
	if err != nil || configuration.State != yorvaruntime.ModelConfigurationConfigured || string(models.secret) != string(secret) || models.applyNativeID != "coder" {
		t.Fatalf("save = %#v fake=%#v %v", configuration, models, err)
	}
	metadata, err = inventory.DeleteModelCredential(context.Background(), instanceID)
	if err != nil || metadata.Configured || models.deleted != "deepseek" || models.applyNativeID != "coder" {
		t.Fatalf("delete = %#v fake=%#v %v", metadata, models, err)
	}
	if models.applyNativeID == instanceID {
		t.Fatal("public instanceId crossed the Runtime boundary")
	}
}

func registerTestModels(t *testing.T, inventory *InstanceInventory, models yorvaruntime.ModelConfigurator) {
	t.Helper()
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes"},
		Discoverer: inventoryDiscoverer{result: yorvaruntime.Discovery{
			RuntimeKind: "hermes", State: yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: `C:\hermes\bin\hermes.exe`, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		}},
		Models: models,
	}); err != nil {
		t.Fatal(err)
	}
	inventory.discovery = NewRuntimeDiscovery(registry, nil)
}
