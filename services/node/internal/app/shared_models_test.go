package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeModelSecretStore struct {
	mu     sync.Mutex
	values map[string][]byte
}

func newFakeModelSecretStore() *fakeModelSecretStore {
	return &fakeModelSecretStore{values: make(map[string][]byte)}
}

func (s *fakeModelSecretStore) Put(_ context.Context, reference string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.values[reference]; exists {
		return errors.New("duplicate")
	}
	s.values[reference] = append([]byte(nil), value...)
	return nil
}

func (s *fakeModelSecretStore) Use(_ context.Context, reference string, use func([]byte)) error {
	s.mu.Lock()
	value, ok := s.values[reference]
	copy := append([]byte(nil), value...)
	s.mu.Unlock()
	if !ok {
		return errors.New("missing")
	}
	use(copy)
	for index := range copy {
		copy[index] = 0
	}
	return nil
}

func (s *fakeModelSecretStore) Delete(_ context.Context, reference string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.values[reference]; !ok {
		return errors.New("missing")
	}
	delete(s.values, reference)
	return nil
}

func (s *fakeModelSecretStore) Configured(_ context.Context, reference string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.values[reference]
	return ok, nil
}

func TestSharedModelResourcesApplyOneCredentialToMultipleProfilesWithReadback(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}, {NativeID: "review"}}, nil)
	models := &fakeModelConfigurator{config: yorvaruntime.ModelConfiguration{
		ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true,
	}}
	registerTestModels(t, inventory, models)
	secrets := newFakeModelSecretStore()
	inventory.WithModelSecrets(secrets)

	connection, err := inventory.CreateModelProviderConnection(context.Background(), hermesRuntimeID, "deepseek", "Production DeepSeek", []byte("shared-secret"))
	if err != nil || !connection.CredentialSet || connection.Status != "CONFIGURED" {
		t.Fatalf("connection = %#v, err=%v", connection, err)
	}
	connections, err := inventory.ListModelProviderConnections(context.Background(), hermesRuntimeID)
	if err != nil || len(connections) != 1 || !connections[0].CredentialSet {
		t.Fatalf("connections = %#v, err=%v", connections, err)
	}
	profile, err := inventory.CreateModelProfile(context.Background(), hermesRuntimeID, connection.ID, "Coding models", "deepseek-v4-pro", []string{"deepseek-v4-pro"})
	if err != nil || profile.ProviderConnectionID != connection.ID {
		t.Fatalf("profile = %#v, err=%v", profile, err)
	}
	defaultValue, err := inventory.SetRuntimeModelDefault(context.Background(), hermesRuntimeID, profile.ID)
	if err != nil || defaultValue.ModelProfileID != profile.ID {
		t.Fatalf("default = %#v, err=%v", defaultValue, err)
	}
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil || len(listed.Instances) != 2 {
		t.Fatalf("instances = %#v, err=%v", listed, err)
	}
	targetIDs := []string{listed.Instances[0].InstanceID, listed.Instances[1].InstanceID}
	started, err := inventory.StartModelProfileApplication(context.Background(), hermesRuntimeID, profile.ID, targetIDs, "INHERIT", "apply-shared-models")
	if err != nil || started.Operation.Type != operation.TypeModelProfileApply {
		t.Fatalf("start = %#v, err=%v", started, err)
	}
	completed := waitForTestOperation(t, inventory.db, started.Operation.ID)
	if completed.Status != operation.StatusSucceeded {
		t.Fatalf("operation = %#v", completed)
	}
	bindings, err := inventory.ListInstanceModelBindings(context.Background(), hermesRuntimeID)
	if err != nil || len(bindings) != 2 {
		t.Fatalf("bindings = %#v, err=%v", bindings, err)
	}
	for _, binding := range bindings {
		if binding.ModelProfileID != profile.ID || binding.Mode != "INHERIT" || binding.State != "SUCCEEDED" || binding.AppliedRevision != profile.Revision {
			t.Fatalf("binding = %#v", binding)
		}
	}
	if models.setCalls != 2 || string(models.secret) != "shared-secret" {
		t.Fatalf("adapter calls=%d secret=%q", models.setCalls, models.secret)
	}
	if err := inventory.DeleteModelProviderConnection(context.Background(), hermesRuntimeID, connection.ID); !errors.Is(err, ErrModelResourceConflict) {
		t.Fatalf("in-use connection delete = %v", err)
	}
}

func TestSharedModelInheritApplicationRequiresMatchingRuntimeDefault(t *testing.T) {
	inventory, _ := newTestInventory(t, []ProfileSnapshot{{NativeID: "coder"}}, nil)
	models := &fakeModelConfigurator{config: yorvaruntime.ModelConfiguration{
		ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true,
	}}
	registerTestModels(t, inventory, models)
	inventory.WithModelSecrets(newFakeModelSecretStore())

	connection, err := inventory.CreateModelProviderConnection(context.Background(), hermesRuntimeID, "deepseek", "Production DeepSeek", []byte("shared-secret"))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := inventory.CreateModelProfile(context.Background(), hermesRuntimeID, connection.ID, "Coding models", "deepseek-v4-pro", []string{"deepseek-v4-pro"})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := inventory.ListInstances(context.Background(), hermesRuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = inventory.StartModelProfileApplication(context.Background(), hermesRuntimeID, profile.ID, []string{listed.Instances[0].InstanceID}, "INHERIT", "inherit-without-default")
	if !errors.Is(err, ErrModelResourceInvalid) {
		t.Fatalf("inherit without matching default = %v", err)
	}
	started, err := inventory.StartModelProfileApplication(context.Background(), hermesRuntimeID, profile.ID, []string{listed.Instances[0].InstanceID}, "OVERRIDE", "override-without-default")
	if err != nil {
		t.Fatalf("override without default = %v", err)
	}
	if completed := waitForTestOperation(t, inventory.db, started.Operation.ID); completed.Status != operation.StatusSucceeded {
		t.Fatalf("override operation = %#v", completed)
	}
}

func waitForTestOperation(t *testing.T, db interface {
	GetOperation(context.Context, string) (operation.Operation, error)
}, operationID string) operation.Operation {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		value, err := db.GetOperation(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if operation.IsTerminal(value.Status) {
			return value
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("operation did not complete")
	return operation.Operation{}
}
