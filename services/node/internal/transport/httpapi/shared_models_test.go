package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

type fakeSharedModelsInventory struct {
	fakeInstanceInventory
	secret      []byte
	profileID   string
	instanceIDs []string
	mode        string
	key         string
}

func (f *fakeSharedModelsInventory) ListModelProviderConnections(context.Context, string) ([]app.ModelProviderConnectionView, error) {
	return []app.ModelProviderConnectionView{{ID: "mpc-1", ProviderPresetID: "deepseek", DisplayName: "Shared DeepSeek", CredentialSet: true, Status: "CONFIGURED", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
}
func (f *fakeSharedModelsInventory) CreateModelProviderConnection(_ context.Context, _, _, _ string, secret []byte) (app.ModelProviderConnectionView, error) {
	f.secret = append([]byte(nil), secret...)
	return app.ModelProviderConnectionView{ID: "mpc-1", ProviderPresetID: "deepseek", DisplayName: "Shared DeepSeek", CredentialSet: true, Status: "CONFIGURED", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (f *fakeSharedModelsInventory) DeleteModelProviderConnection(context.Context, string, string) error {
	return nil
}
func (f *fakeSharedModelsInventory) ListModelProfiles(context.Context, string) ([]app.ModelProfileView, error) {
	return nil, nil
}
func (f *fakeSharedModelsInventory) CreateModelProfile(context.Context, string, string, string, string, []string) (app.ModelProfileView, error) {
	return app.ModelProfileView{}, nil
}
func (f *fakeSharedModelsInventory) DeleteModelProfile(context.Context, string, string) error {
	return nil
}
func (f *fakeSharedModelsInventory) GetRuntimeModelDefault(context.Context, string) (app.RuntimeModelDefaultView, error) {
	return app.RuntimeModelDefaultView{}, nil
}
func (f *fakeSharedModelsInventory) SetRuntimeModelDefault(_ context.Context, _ string, profileID string) (app.RuntimeModelDefaultView, error) {
	f.profileID = profileID
	return app.RuntimeModelDefaultView{ModelProfileID: profileID, AppliedRevision: 1, UpdatedAt: time.Now()}, nil
}
func (f *fakeSharedModelsInventory) ClearRuntimeModelDefault(context.Context, string) error {
	return nil
}
func (f *fakeSharedModelsInventory) ListInstanceModelBindings(context.Context, string) ([]app.InstanceModelBindingView, error) {
	return nil, nil
}
func (f *fakeSharedModelsInventory) StartModelProfileApplication(_ context.Context, _ string, profileID string, instanceIDs []string, mode, key string) (app.InstallStartResult, error) {
	f.profileID, f.instanceIDs, f.mode, f.key = profileID, append([]string(nil), instanceIDs...), mode, key
	now := time.Now()
	return app.InstallStartResult{Created: true, Operation: operation.Operation{ID: "op-model-apply", Type: operation.TypeModelProfileApply, Status: operation.StatusPending, Stage: operation.StagePreflight, CreatedAt: now, UpdatedAt: now}}, nil
}
func (f *fakeSharedModelsInventory) CancelModelProfileApplication(context.Context, string) (operation.Operation, error) {
	return operation.Operation{}, nil
}

func TestSharedModelHTTPKeepsCredentialWriteOnlyAndBodiesClosed(t *testing.T) {
	service := &fakeSharedModelsInventory{}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, service, "", nil)
	const secret = "shared-provider-secret"

	create := authorizedRequest(http.MethodPost, "/api/v1/runtimes/hermes/model-provider-connections", `{"providerPresetId":"deepseek","displayName":"Shared DeepSeek","credential":"`+secret+`"}`)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated || string(service.secret) != secret || strings.Contains(created.Body.String(), secret) || strings.Contains(created.Body.String(), "credential\"") {
		t.Fatalf("create = %d %s secret=%q", created.Code, created.Body.String(), service.secret)
	}

	unknown := authorizedRequest(http.MethodPost, "/api/v1/runtimes/hermes/model-provider-connections", `{"providerPresetId":"deepseek","displayName":"Shared DeepSeek","credential":"safe","env":{"EVIL":"value"}}`)
	rejected := httptest.NewRecorder()
	handler.ServeHTTP(rejected, unknown)
	if rejected.Code != http.StatusBadRequest || len(service.secret) != len(secret) {
		t.Fatalf("closed body = %d %s", rejected.Code, rejected.Body.String())
	}

	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, authorizedRequest(http.MethodGet, "/api/v1/runtimes/hermes/model-provider-connections", ""))
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), secret) || !strings.Contains(listed.Body.String(), `"credentialConfigured":true`) {
		t.Fatalf("list = %d %s", listed.Code, listed.Body.String())
	}
}

func TestSharedModelHTTPAppliesTypedProfileBinding(t *testing.T) {
	service := &fakeSharedModelsInventory{}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, service, "", nil)
	request := authorizedRequest(http.MethodPost, "/api/v1/runtimes/hermes/model-profile-applications", `{"modelProfileId":"mpr-1","instanceIds":["inst-1","inst-2"],"mode":"INHERIT"}`)
	request.Header.Set("Idempotency-Key", "apply-model-profile")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, request)
	if result.Code != http.StatusAccepted || service.profileID != "mpr-1" || service.mode != "INHERIT" || service.key != "apply-model-profile" || len(service.instanceIDs) != 2 {
		t.Fatalf("apply = %d %s fake=%#v", result.Code, result.Body.String(), service)
	}
}
