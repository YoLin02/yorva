package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeModelsAPI struct {
	fakeInstanceInventory
	configuration app.ModelConfigurationView
	err           error
	patchedPreset string
	patchedModel  string
	credential    app.ModelCredentialView
	secret        []byte
	cancelCalled  bool
}

func (f *fakeModelsAPI) ListModelProviderPresets(context.Context) ([]yorvaruntime.ModelProviderPreset, error) {
	return []yorvaruntime.ModelProviderPreset{{ID: "deepseek", DisplayName: "DeepSeek", Region: yorvaruntime.ModelRegionChina, RecommendedModels: []string{"deepseek-v4-pro"}}}, f.err
}

func (f *fakeModelsAPI) GetModelConfiguration(context.Context, string) (app.ModelConfigurationView, error) {
	return f.configuration, f.err
}

func (f *fakeModelsAPI) PatchModelConfiguration(_ context.Context, _, presetID, modelID string) (app.ModelConfigurationView, error) {
	f.patchedPreset, f.patchedModel = presetID, modelID
	return f.configuration, f.err
}

func (f *fakeModelsAPI) GetModelCredential(context.Context, string) (app.ModelCredentialView, error) {
	return f.credential, f.err
}

func (f *fakeModelsAPI) SaveModelCredentialConfiguration(_ context.Context, _, presetID, modelID string, secret []byte) (app.ModelConfigurationView, error) {
	f.patchedPreset, f.patchedModel = presetID, modelID
	f.secret = append([]byte(nil), secret...)
	return f.configuration, f.err
}

func (f *fakeModelsAPI) DeleteModelCredential(context.Context, string) (app.ModelCredentialView, error) {
	return f.credential, f.err
}

func (f *fakeModelsAPI) StartModelValidation(_ context.Context, instanceID, key string) (app.InstallStartResult, error) {
	return app.InstallStartResult{Created: true, Operation: operation.Operation{
		ID: "op_validate", Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: instanceID,
		Status: operation.StatusPending, Stage: operation.StagePreflight, IdempotencyKey: key, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}, f.err
}

func (f *fakeModelsAPI) CancelModelValidation(context.Context, string) (operation.Operation, error) {
	f.cancelCalled = true
	return operation.Operation{ID: "op_validate", Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: "inst_public", Status: operation.StatusCancelled, Stage: operation.StageModelValidate, CreatedAt: time.Now(), UpdatedAt: time.Now()}, f.err
}

func TestModelHTTPContractIsAuthenticatedSafeAndClosed(t *testing.T) {
	now := time.Date(2026, 8, 19, 18, 0, 0, 0, time.UTC)
	models := &fakeModelsAPI{configuration: app.ModelConfigurationView{
		ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured,
		CredentialConfigured: true, ObservedAt: now,
	}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)

	unauthorized := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/model-provider-presets", nil)
	unauthorizedResult := httptest.NewRecorder()
	handler.ServeHTTP(unauthorizedResult, unauthorized)
	if unauthorizedResult.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorizedResult.Code)
	}

	presets := authorizedRequest(http.MethodGet, "/api/v1/runtimes/hermes/model-provider-presets", "")
	presetsResult := httptest.NewRecorder()
	handler.ServeHTTP(presetsResult, presets)
	if presetsResult.Code != http.StatusOK || strings.Contains(presetsResult.Body.String(), "API_KEY") || strings.Contains(presetsResult.Body.String(), "model.provider") {
		t.Fatalf("presets = %d %s", presetsResult.Code, presetsResult.Body.String())
	}

	get := authorizedRequest(http.MethodGet, "/api/v1/instances/inst_public/config", "")
	getResult := httptest.NewRecorder()
	handler.ServeHTTP(getResult, get)
	if getResult.Code != http.StatusOK || !strings.Contains(getResult.Body.String(), `"state":"CONFIGURED"`) || !strings.Contains(getResult.Body.String(), `"validation":{"state":"NOT_RUN"`) {
		t.Fatalf("get = %d %s", getResult.Code, getResult.Body.String())
	}

	patch := authorizedRequest(http.MethodPatch, "/api/v1/instances/inst_public/config", `{"providerPresetId":"deepseek","modelId":"deepseek-v4-pro"}`)
	patchResult := httptest.NewRecorder()
	handler.ServeHTTP(patchResult, patch)
	if patchResult.Code != http.StatusOK || models.patchedPreset != "deepseek" || models.patchedModel != "deepseek-v4-pro" {
		t.Fatalf("patch = %d %s fake=%#v", patchResult.Code, patchResult.Body.String(), models)
	}

	unknown := authorizedRequest(http.MethodPatch, "/api/v1/instances/inst_public/config", `{"providerPresetId":"deepseek","modelId":"m","extra":true}`)
	unknownResult := httptest.NewRecorder()
	handler.ServeHTTP(unknownResult, unknown)
	if unknownResult.Code != http.StatusBadRequest || !strings.Contains(unknownResult.Body.String(), string(yorvaruntime.ErrorModelConfigInvalid)) {
		t.Fatalf("unknown field = %d %s", unknownResult.Code, unknownResult.Body.String())
	}
}

func TestModelHTTPReturnsStableObservedIncompleteError(t *testing.T) {
	models := &fakeModelsAPI{
		configuration: app.ModelConfigurationView{ProviderPresetID: "deepseek", ModelID: "old-model", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true, ObservedAt: time.Now()},
		err:           yorvaruntime.ErrModelConfigIncomplete,
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)
	req := authorizedRequest(http.MethodPatch, "/api/v1/instances/inst_public/config", `{"providerPresetId":"deepseek","modelId":"new-model"}`)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), string(yorvaruntime.ErrorModelConfigIncomplete)) || !strings.Contains(res.Body.String(), "old-model") {
		t.Fatalf("incomplete = %d %s", res.Code, res.Body.String())
	}
	var body ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body.Error.Details == nil {
		t.Fatalf("error body = %#v %v", body, err)
	}

	models.err = errors.New("private native output")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, authorizedRequest(http.MethodPatch, "/api/v1/instances/inst_public/config", `{"providerPresetId":"deepseek","modelId":"new-model"}`))
	if strings.Contains(res.Body.String(), "private native output") {
		t.Fatalf("raw error leaked: %s", res.Body.String())
	}
}

func TestModelRoutesAdvertisePatchAndCorsPut(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, &fakeModelsAPI{}, "", nil)
	options := httptest.NewRequest(http.MethodOptions, "/api/v1/instances/inst_public/config", nil)
	options.Header.Set("Origin", "http://127.0.0.1:1420")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, options)
	if result.Code != http.StatusNoContent || result.Header().Get("Allow") != "GET, PATCH, OPTIONS" || !strings.Contains(result.Header().Get("Access-Control-Allow-Methods"), "PUT") {
		t.Fatalf("options = %d allow=%q cors=%q", result.Code, result.Header().Get("Allow"), result.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestModelCredentialHTTPIsMetadataOnlyWriteOnlyAndClosed(t *testing.T) {
	const secret = "sk-batch-three-http-sentinel"
	now := time.Date(2026, 8, 19, 19, 0, 0, 0, time.UTC)
	models := &fakeModelsAPI{
		configuration: app.ModelConfigurationView{ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured, CredentialConfigured: true, ObservedAt: now},
		credential:    app.ModelCredentialView{ProviderPresetID: "deepseek", Configured: true, ObservedAt: now},
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)

	get := authorizedRequest(http.MethodGet, "/api/v1/instances/inst_public/credentials/model-provider", "")
	getResult := httptest.NewRecorder()
	handler.ServeHTTP(getResult, get)
	if getResult.Code != http.StatusOK || !strings.Contains(getResult.Body.String(), `"configured":true`) || strings.Contains(getResult.Body.String(), secret) || strings.Contains(getResult.Body.String(), "value") {
		t.Fatalf("metadata = %d %s", getResult.Code, getResult.Body.String())
	}

	putBody := `{"providerPresetId":"deepseek","modelId":"deepseek-v4-pro","value":"` + secret + `"}`
	put := authorizedRequest(http.MethodPut, "/api/v1/instances/inst_public/credentials/model-provider", putBody)
	putResult := httptest.NewRecorder()
	handler.ServeHTTP(putResult, put)
	if putResult.Code != http.StatusOK || string(models.secret) != secret || strings.Contains(putResult.Body.String(), secret) || strings.Contains(putResult.Body.String(), "value") {
		t.Fatalf("put = %d %s fake=%#v", putResult.Code, putResult.Body.String(), models)
	}

	invalid := authorizedRequest(http.MethodPut, "/api/v1/instances/inst_public/credentials/model-provider", strings.TrimSuffix(putBody, "}")+`,"envName":"EVIL"}`)
	invalidResult := httptest.NewRecorder()
	handler.ServeHTTP(invalidResult, invalid)
	if invalidResult.Code != http.StatusBadRequest || strings.Contains(invalidResult.Body.String(), secret) {
		t.Fatalf("closed put = %d %s", invalidResult.Code, invalidResult.Body.String())
	}

	del := authorizedRequest(http.MethodDelete, "/api/v1/instances/inst_public/credentials/model-provider", "")
	delResult := httptest.NewRecorder()
	handler.ServeHTTP(delResult, del)
	if delResult.Code != http.StatusOK || strings.Contains(delResult.Body.String(), secret) {
		t.Fatalf("delete = %d %s", delResult.Code, delResult.Body.String())
	}

	options := httptest.NewRequest(http.MethodOptions, "/api/v1/instances/inst_public/credentials/model-provider", nil)
	optionsResult := httptest.NewRecorder()
	handler.ServeHTTP(optionsResult, options)
	if optionsResult.Code != http.StatusNoContent || optionsResult.Header().Get("Allow") != "GET, PUT, DELETE, OPTIONS" {
		t.Fatalf("credential options = %d %q", optionsResult.Code, optionsResult.Header().Get("Allow"))
	}
}

func TestModelCredentialHTTPAcceptsOpenAPIMaximumEscapedValue(t *testing.T) {
	models := &fakeModelsAPI{configuration: app.ModelConfigurationView{
		ProviderPresetID: "deepseek", ModelID: "deepseek-v4-pro", State: yorvaruntime.ModelConfigurationConfigured,
		CredentialConfigured: true, ObservedAt: time.Now(),
	}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)
	secret := strings.Repeat("\"", 16*1024)
	payload, err := json.Marshal(map[string]string{
		"providerPresetId": "deepseek",
		"modelId":          "deepseek-v4-pro",
		"value":            secret,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) <= 24*1024 {
		t.Fatalf("escaped payload did not exercise the former limit: %d", len(payload))
	}
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, authorizedRequest(http.MethodPut, "/api/v1/instances/inst_public/credentials/model-provider", string(payload)))
	if result.Code != http.StatusOK || len(models.secret) != len(secret) {
		t.Fatalf("maximum escaped credential = %d body=%s stored=%d", result.Code, result.Body.String(), len(models.secret))
	}
}

func TestModelCredentialHTTPReturnsSafeIncompleteAfterWrite(t *testing.T) {
	const secret = "sk-partial-save-must-not-echo"
	models := &fakeModelsAPI{
		configuration: app.ModelConfigurationView{State: yorvaruntime.ModelConfigurationUnconfigured, ObservedAt: time.Now()},
		err:           yorvaruntime.ErrModelConfigIncomplete,
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)
	payload := `{"providerPresetId":"deepseek","modelId":"deepseek-v4-pro","value":"` + secret + `"}`
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, authorizedRequest(http.MethodPut, "/api/v1/instances/inst_public/credentials/model-provider", payload))
	if result.Code != http.StatusConflict ||
		!strings.Contains(result.Body.String(), string(yorvaruntime.ErrorModelConfigIncomplete)) ||
		!strings.Contains(result.Body.String(), `"state":"UNCONFIGURED"`) ||
		strings.Contains(result.Body.String(), secret) || strings.Contains(result.Body.String(), `"value"`) {
		t.Fatalf("partial save = %d %s", result.Code, result.Body.String())
	}
}

func TestModelValidationHTTPStartsExplicitOperationWithClosedBody(t *testing.T) {
	models := &fakeModelsAPI{}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, models, "", nil)

	req := authorizedRequest(http.MethodPost, "/api/v1/instances/inst_public/model-validation", `{}`)
	req.Header.Set("Idempotency-Key", "validate-http")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted || !strings.Contains(res.Body.String(), `"type":"model.validate"`) || !strings.Contains(res.Body.String(), `"targetType":"instance"`) || !strings.Contains(res.Body.String(), `"targetId":"inst_public"`) {
		t.Fatalf("validation start = %d %s", res.Code, res.Body.String())
	}

	unknown := authorizedRequest(http.MethodPost, "/api/v1/instances/inst_public/model-validation", `{"prompt":"leak"}`)
	unknown.Header.Set("Idempotency-Key", "validate-bad-body")
	unknownResult := httptest.NewRecorder()
	handler.ServeHTTP(unknownResult, unknown)
	if unknownResult.Code != http.StatusBadRequest || strings.Contains(unknownResult.Body.String(), "leak") {
		t.Fatalf("closed validation = %d %s", unknownResult.Code, unknownResult.Body.String())
	}

	missingKey := authorizedRequest(http.MethodPost, "/api/v1/instances/inst_public/model-validation", `{}`)
	missingResult := httptest.NewRecorder()
	handler.ServeHTTP(missingResult, missingKey)
	if missingResult.Code != http.StatusBadRequest || !strings.Contains(missingResult.Body.String(), "INVALID_IDEMPOTENCY_KEY") {
		t.Fatalf("validation idempotency = %d %s", missingResult.Code, missingResult.Body.String())
	}

	options := httptest.NewRequest(http.MethodOptions, "/api/v1/instances/inst_public/model-validation", nil)
	optionsResult := httptest.NewRecorder()
	handler.ServeHTTP(optionsResult, options)
	if optionsResult.Code != http.StatusNoContent || optionsResult.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatalf("validation options = %d %q", optionsResult.Code, optionsResult.Header().Get("Allow"))
	}
}

func TestOperationCancelDispatchesModelValidationToInstanceModelService(t *testing.T) {
	models := &fakeModelsAPI{}
	operationValue := operation.Operation{ID: "op_validate", Type: operation.TypeModelValidate, TargetType: operation.TargetInstance, TargetID: "inst_public", Status: operation.StatusRunning, Stage: operation.StageModelValidate, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	installs := fakeInstallService{started: app.InstallStartResult{Operation: operationValue}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, installs, models, "", nil)
	req := authorizedRequest(http.MethodPost, "/api/v1/operations/op_validate/cancel", "")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !models.cancelCalled || !strings.Contains(res.Body.String(), `"status":"CANCELLED"`) {
		t.Fatalf("cancel = %d %s called=%v", res.Code, res.Body.String(), models.cancelCalled)
	}
}

func authorizedRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testToken)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}
