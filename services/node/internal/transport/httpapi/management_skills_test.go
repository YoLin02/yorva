package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeManagementSkillsService struct {
	listed    []yorvaruntime.Skill
	inspected yorvaruntime.Skill
	err       error
	instance  string
	skillID   string
}

func (f *fakeManagementSkillsService) ListSkills(_ context.Context, instanceID string) ([]yorvaruntime.Skill, error) {
	f.instance = instanceID
	return f.listed, f.err
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
		InstallationState: yorvaruntime.SkillInstalled,
		EnabledState:      yorvaruntime.SkillEnabled,
		ScanState:         yorvaruntime.SkillScanClean,
		UpdateAvailable:   true,
	}
}

func TestManagementSkillsHTTPListAndInspectClosedResponses(t *testing.T) {
	service := &fakeManagementSkillsService{listed: []yorvaruntime.Skill{httpSkill("writer")}, inspected: httpSkill("writer")}
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
	wantFields := []string{"id", "sourceId", "version", "installationState", "enabledState", "scanState", "updateAvailable"}
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
