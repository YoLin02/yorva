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
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeManagementUpgradePlanService struct {
	plan      yorvaruntime.UpgradePlan
	err       error
	runtimeID string
}

func (f *fakeManagementUpgradePlanService) PlanUpgrade(_ context.Context, runtimeID string) (yorvaruntime.UpgradePlan, error) {
	f.runtimeID = runtimeID
	return f.plan, f.err
}

func httpUpgradePlan() yorvaruntime.UpgradePlan {
	return yorvaruntime.UpgradePlan{
		State:                   yorvaruntime.UpgradeAvailable,
		Rollback:                yorvaruntime.RollbackEligible,
		CurrentVersion:          "0.20.2",
		TargetVersion:           "0.20.5",
		Managed:                 true,
		InventoryComplete:       true,
		ProtectionPointRequired: true,
		ProtectionPointReady:    true,
		PlanEvidenceComplete:    true,
		ObservedAt:              time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	}
}

func TestManagementUpgradeHTTPReturnsClosedSafePlan(t *testing.T) {
	service := &fakeManagementUpgradePlanService{plan: httpUpgradePlan()}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/upgrade-plan", nil)
	request.SetPathValue("runtimeId", "hermes")
	response := httptest.NewRecorder()
	getRuntimeUpgradePlan(service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.runtimeID != "hermes" {
		t.Fatalf("response = %d %s, runtime=%q", response.Code, response.Body.String(), service.runtimeID)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"state", "rollbackEligibility", "currentVersion", "targetVersion", "managed",
		"inventoryComplete", "protectionPointRequired", "protectionPointReady", "planEvidenceComplete",
		"upgradeMutationQualified", "rollbackMutationQualified", "upgradeExecutable", "rollbackExecutable", "observedAt",
	}
	if len(body) != len(want) {
		t.Fatalf("fields = %v", body)
	}
	for _, field := range want {
		if _, ok := body[field]; !ok {
			t.Fatalf("missing field %q in %s", field, response.Body.String())
		}
	}
	for _, field := range []string{"upgradeMutationQualified", "rollbackMutationQualified", "upgradeExecutable", "rollbackExecutable"} {
		if string(body[field]) != "false" {
			t.Fatalf("unqualified plan reported %s=%s", field, body[field])
		}
	}
	if string(body["planEvidenceComplete"]) != "true" {
		t.Fatalf("planning evidence truth lost: %s", body["planEvidenceComplete"])
	}
	if _, ambiguous := body["executable"]; ambiguous {
		t.Fatalf("ambiguous executable field remained: %s", response.Body.String())
	}
	for _, forbidden := range []string{
		"path", "command", "argv", "environment", "url", "secret", "credential", "token",
		"seal", "sha256", "installationId", "protectionPointId",
	} {
		if strings.Contains(strings.ToLower(response.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("response exposed forbidden field %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestManagementUpgradeHTTPCapabilityFalse(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/upgrade-plan", nil)
	request.SetPathValue("runtimeId", "hermes")
	for name, service := range map[string]ManagementUpgradePlanService{
		"nil service": nil,
		"nil planner": &fakeManagementUpgradePlanService{err: app.ErrManagementCapabilityUnsupported},
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			getRuntimeUpgradePlan(service).ServeHTTP(response, request)
			if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), string(yorvaruntime.ErrorCapabilityNotSupported)) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestManagementUpgradeHTTPUsesSafeStableErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"query failure", app.ErrManagementQueryFailed, http.StatusServiceUnavailable, "MANAGEMENT_QUERY_FAILED"},
		{"deadline", context.DeadlineExceeded, http.StatusServiceUnavailable, "MANAGEMENT_QUERY_FAILED"},
		{"unowned cancellation", context.Canceled, http.StatusServiceUnavailable, "MANAGEMENT_QUERY_FAILED"},
		{"unsupported Runtime", app.ErrRuntimeNotSupported, http.StatusConflict, string(yorvaruntime.ErrorRuntimeNotSupported)},
		{"raw internal", errors.New("C:/secret generation seal token-super-secret"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeManagementUpgradePlanService{err: test.err}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/upgrade-plan", nil)
			request.SetPathValue("runtimeId", "hermes")
			response := httptest.NewRecorder()
			getRuntimeUpgradePlan(service).ServeHTTP(response, request)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "C:/secret") || strings.Contains(response.Body.String(), "token-super-secret") {
				t.Fatalf("response leaked internal error: %s", response.Body.String())
			}
		})
	}
}
