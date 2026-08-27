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

type fakeManagementSkillsService struct {
	listed    []yorvaruntime.Skill
	inspected yorvaruntime.Skill
	sources   []app.SkillSourceView
	started   operation.Operation
	err       error
	instance  string
	skillID   string
	sourceID  string
	key       string
	action    string
}

func (f *fakeManagementSkillsService) ListSkills(_ context.Context, instanceID string) ([]yorvaruntime.Skill, error) {
	f.instance = instanceID
	return f.listed, f.err
}

func (f *fakeManagementSkillsService) ListSources(_ context.Context, instanceID string) ([]app.SkillSourceView, error) {
	f.instance = instanceID
	return f.sources, f.err
}

func (f *fakeManagementSkillsService) start(instanceID, skillID, sourceID, key, action string) (app.InstallStartResult, error) {
	f.instance, f.skillID, f.sourceID, f.key, f.action = instanceID, skillID, sourceID, key, action
	return app.InstallStartResult{Operation: f.started, Created: true}, f.err
}

func (f *fakeManagementSkillsService) StartInstall(_ context.Context, instanceID, skillID, sourceID, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, sourceID, key, "install")
}

func (f *fakeManagementSkillsService) StartImport(_ context.Context, instanceID, skillID, sourceRef, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, sourceRef, key, "import")
}

func (f *fakeManagementSkillsService) StartUpdate(_ context.Context, instanceID, skillID, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, "", key, "update")
}

func (f *fakeManagementSkillsService) StartEnable(_ context.Context, instanceID, skillID, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, "", key, "enable")
}

func (f *fakeManagementSkillsService) StartDisable(_ context.Context, instanceID, skillID, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, "", key, "disable")
}

func (f *fakeManagementSkillsService) StartRemove(_ context.Context, instanceID, skillID, key string) (app.InstallStartResult, error) {
	return f.start(instanceID, skillID, "", key, "remove")
}

func (f *fakeManagementSkillsService) InspectSkill(_ context.Context, instanceID, skillID string) (yorvaruntime.Skill, error) {
	f.instance = instanceID
	f.skillID = skillID
	return f.inspected, f.err
}

func httpSkill(id string) yorvaruntime.Skill {
	return yorvaruntime.Skill{
		ID:                id,
		SourceID:          "approved.source",
		Version:           "1.2.3",
		Ownership:         yorvaruntime.SkillOwnershipYORVAManaged,
		ProjectionState:   yorvaruntime.SkillProjectionProjected,
		InstallationState: yorvaruntime.SkillInstalled,
		EnabledState:      yorvaruntime.SkillEnabled,
		ScanState:         yorvaruntime.SkillScanClean,
		UpdateAvailable:   true,
	}
}

func TestManagementSkillsHTTPListAndInspectClosedResponses(t *testing.T) {
	listed := httpSkill("writer")
	listed.Description = "Draft and refine documents."
	inspected := listed
	inspected.Preview = "# Writer\n\nSafe preview."
	service := &fakeManagementSkillsService{listed: []yorvaruntime.Skill{listed}, inspected: inspected}
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills", nil)
	listRequest.SetPathValue("instanceId", "inst_1")
	listResponse := httptest.NewRecorder()
	listInstanceSkills(service).ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || service.instance != "inst_1" {
		t.Fatalf("list response = %d %s, instance=%q", listResponse.Code, listResponse.Body.String(), service.instance)
	}
	var listBody map[string]json.RawMessage
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listBody); err != nil || len(listBody) != 1 || listBody["items"] == nil {
		t.Fatalf("list JSON = %s, %v", listResponse.Body.String(), err)
	}

	inspectRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills/writer", nil)
	inspectRequest.SetPathValue("instanceId", "inst_1")
	inspectRequest.SetPathValue("skillId", "writer")
	inspectResponse := httptest.NewRecorder()
	inspectInstanceSkill(service).ServeHTTP(inspectResponse, inspectRequest)
	if inspectResponse.Code != http.StatusOK || service.instance != "inst_1" || service.skillID != "writer" {
		t.Fatalf("inspect response = %d %s, target=%q/%q", inspectResponse.Code, inspectResponse.Body.String(), service.instance, service.skillID)
	}
	var inspectBody map[string]json.RawMessage
	if err := json.Unmarshal(inspectResponse.Body.Bytes(), &inspectBody); err != nil {
		t.Fatal(err)
	}
	wantFields := []string{"id", "sourceId", "version", "description", "preview", "ownership", "projectionState", "installationState", "enabledState", "scanState", "updateAvailable"}
	if len(inspectBody) != len(wantFields) {
		t.Fatalf("inspect fields = %v", inspectBody)
	}
	for _, field := range wantFields {
		if _, ok := inspectBody[field]; !ok {
			t.Fatalf("missing field %q in %s", field, inspectResponse.Body.String())
		}
	}
}

func TestManagementSkillsHTTPAlwaysEmitsEmptyArray(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills", nil)
	request.SetPathValue("instanceId", "inst_1")
	listInstanceSkills(&fakeManagementSkillsService{}).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestManagementSkillsHTTPCatalogAndMutationsAreClosed(t *testing.T) {
	now := time.Now().UTC()
	service := &fakeManagementSkillsService{
		sources: []app.SkillSourceView{{SourceID: "yorva-demo", SkillID: "yorva-managed-demo", DisplayName: "Demo", Version: "1.0.0"}},
		started: operation.Operation{ID: "op_skill", Type: operation.TypeSkillInstall, TargetType: operation.TargetInstance, TargetID: "inst_1", Status: operation.StatusPending, Stage: operation.StageSkillProject, CorrelationID: "cor_skill", CreatedAt: now, UpdatedAt: now},
	}
	catalogRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skill-sources", nil)
	catalogRequest.SetPathValue("instanceId", "inst_1")
	catalogResponse := httptest.NewRecorder()
	listInstanceSkillSources(service).ServeHTTP(catalogResponse, catalogRequest)
	if catalogResponse.Code != http.StatusOK || !strings.Contains(catalogResponse.Body.String(), `"sourceId":"yorva-demo"`) {
		t.Fatalf("catalog response = %d %s", catalogResponse.Code, catalogResponse.Body.String())
	}
	for _, prohibited := range []string{"absolutePath", "relativePath", "contentSHA256", "deploymentId", `C:\\`} {
		if strings.Contains(catalogResponse.Body.String(), prohibited) {
			t.Fatalf("catalog leaked %q: %s", prohibited, catalogResponse.Body.String())
		}
	}

	install := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/skills/yorva-managed-demo/install", strings.NewReader(`{"sourceId":"yorva-demo"}`))
	install.Header.Set("Idempotency-Key", "skill-install-1")
	install.SetPathValue("instanceId", "inst_1")
	install.SetPathValue("skillId", "yorva-managed-demo")
	installResponse := httptest.NewRecorder()
	startManagedSkillMutation(service, skillMutationInstall).ServeHTTP(installResponse, install)
	if installResponse.Code != http.StatusAccepted || service.action != "install" || service.sourceID != "yorva-demo" || service.key != "skill-install-1" {
		t.Fatalf("install response = %d %s calls=%#v", installResponse.Code, installResponse.Body.String(), service)
	}

	importRequest := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/skills/local-skill/import", strings.NewReader(`{"sourceRef":"sssssssssssssssssssssssssssssssssssssssssss"}`))
	importRequest.Header.Set("Idempotency-Key", "skill-import-1")
	importRequest.SetPathValue("instanceId", "inst_1")
	importRequest.SetPathValue("skillId", "local-skill")
	importResponse := httptest.NewRecorder()
	startManagedSkillMutation(service, skillMutationImport).ServeHTTP(importResponse, importRequest)
	if importResponse.Code != http.StatusAccepted || service.action != "import" || service.sourceID != strings.Repeat("s", 43) {
		t.Fatalf("import response = %d %s calls=%#v", importResponse.Code, importResponse.Body.String(), service)
	}
	unsafeImport := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"sourceRef":"C:\\\\secret"}`))
	unsafeImport.Header.Set("Idempotency-Key", "skill-import-unsafe")
	unsafeImport.SetPathValue("instanceId", "inst_1")
	unsafeImport.SetPathValue("skillId", "local-skill")
	unsafeResponse := httptest.NewRecorder()
	startManagedSkillMutation(service, skillMutationImport).ServeHTTP(unsafeResponse, unsafeImport)
	if unsafeResponse.Code != http.StatusBadRequest {
		t.Fatalf("unsafe import response = %d %s", unsafeResponse.Code, unsafeResponse.Body.String())
	}

	for _, body := range []string{`{"sourceId":"yorva-demo","path":"C:\\secret"}`, `{"sourceId":"yorva-demo"} {}`, `{}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/skills/yorva-managed-demo/install", strings.NewReader(body))
		request.Header.Set("Idempotency-Key", "skill-invalid")
		request.SetPathValue("instanceId", "inst_1")
		request.SetPathValue("skillId", "yorva-managed-demo")
		response := httptest.NewRecorder()
		startManagedSkillMutation(service, skillMutationInstall).ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("body %q response = %d %s", body, response.Code, response.Body.String())
		}
	}

	update := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/skills/yorva-managed-demo/update", strings.NewReader(`{}`))
	update.Header.Set("Idempotency-Key", "skill-update-1")
	update.SetPathValue("instanceId", "inst_1")
	update.SetPathValue("skillId", "yorva-managed-demo")
	updateResponse := httptest.NewRecorder()
	startManagedSkillMutation(service, skillMutationUpdate).ServeHTTP(updateResponse, update)
	if updateResponse.Code != http.StatusAccepted || service.action != "update" {
		t.Fatalf("update response = %d %s", updateResponse.Code, updateResponse.Body.String())
	}
}

func TestManagementSkillsHTTPMutationRequiresBearerAndIdempotency(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, nil, "", nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/skills/writer/update", strings.NewReader(`{}`))
	request.Header.Set("Idempotency-Key", "skill-auth")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated mutation = %d %s", response.Code, response.Body.String())
	}

	service := &fakeManagementSkillsService{}
	missingKey := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	missingKey.SetPathValue("instanceId", "inst_1")
	missingKey.SetPathValue("skillId", "writer")
	missingKeyResponse := httptest.NewRecorder()
	startManagedSkillMutation(service, skillMutationUpdate).ServeHTTP(missingKeyResponse, missingKey)
	if missingKeyResponse.Code != http.StatusBadRequest || service.action != "" {
		t.Fatalf("missing idempotency key = %d %s action=%q", missingKeyResponse.Code, missingKeyResponse.Body.String(), service.action)
	}
}

func TestManagementSkillsHTTPCapabilityFalse(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills", nil)
	request.SetPathValue("instanceId", "inst_1")
	for name, service := range map[string]ManagementSkillsService{
		"nil service": nil,
		"nil reader":  &fakeManagementSkillsService{err: app.ErrManagementCapabilityUnsupported},
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			listInstanceSkills(service).ServeHTTP(response, request)
			if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), string(yorvaruntime.ErrorCapabilityNotSupported)) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestManagementSkillsHTTPDoesNotLeakAdapterErrors(t *testing.T) {
	const secret = "token-super-secret"
	service := &fakeManagementSkillsService{err: errors.New(secret)}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills/writer", nil)
	request.SetPathValue("instanceId", "inst_1")
	request.SetPathValue("skillId", "writer")
	response := httptest.NewRecorder()
	inspectInstanceSkill(service).ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), secret) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestManagementSkillsHTTPMapsNormalizedQueryFailure(t *testing.T) {
	service := &fakeManagementSkillsService{err: app.ErrManagementQueryFailed}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/skills", nil)
	request.SetPathValue("instanceId", "inst_1")
	response := httptest.NewRecorder()
	listInstanceSkills(service).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"MANAGEMENT_QUERY_FAILED"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
