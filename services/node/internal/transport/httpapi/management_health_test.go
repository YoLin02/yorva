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

type b3FakeManagementHealthService struct {
	health        yorvaruntime.HealthObservation
	logs          yorvaruntime.LogSnapshot
	security      yorvaruntime.SecurityAuditResult
	err           error
	gotInstanceID string
	gotCategory   yorvaruntime.LogCategory
	calls         int
}

func (f *b3FakeManagementHealthService) GetInstanceHealth(_ context.Context, instanceID string) (yorvaruntime.HealthObservation, error) {
	f.capture(instanceID)
	return f.health, f.err
}

func (f *b3FakeManagementHealthService) GetInstanceLogSnapshot(_ context.Context, instanceID string, category yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	f.capture(instanceID)
	f.gotCategory = category
	return f.logs, f.err
}

func (f *b3FakeManagementHealthService) GetInstanceSecurityAudit(_ context.Context, instanceID string) (yorvaruntime.SecurityAuditResult, error) {
	f.capture(instanceID)
	return f.security, f.err
}

func (f *b3FakeManagementHealthService) capture(instanceID string) {
	f.calls++
	f.gotInstanceID = instanceID
}

func TestManagementHealthHandlersUsePathTargetAndReturnTypedSafeResponses(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	service := &b3FakeManagementHealthService{
		health: yorvaruntime.HealthObservation{
			State: yorvaruntime.HealthHealthy, Findings: []yorvaruntime.HealthFinding{{Code: "gateway", State: yorvaruntime.HealthHealthy}}, ObservedAt: now,
		},
		logs: yorvaruntime.LogSnapshot{
			Category: yorvaruntime.LogCategoryErrors, Entries: []yorvaruntime.LogEntry{{Timestamp: now, Message: "token=[REDACTED]"}}, Truncated: true, ObservedAt: now,
		},
		security: yorvaruntime.SecurityAuditResult{
			State:    yorvaruntime.SecurityAuditWarning,
			Findings: []yorvaruntime.SecurityFinding{{ID: "dependency-1", Component: "dependency", Severity: yorvaruntime.SecuritySeverityHigh}},
			Partial:  true, ObservedAt: now,
		},
	}

	tests := []struct {
		name     string
		handler  http.Handler
		target   string
		contains []string
	}{
		{name: "health", handler: getInstanceHealth(service), target: "/health", contains: []string{`"state":"HEALTHY"`, `"code":"gateway"`, `"findings":[`}},
		{name: "logs", handler: getInstanceLogSnapshot(service), target: "/logs?category=ERRORS", contains: []string{`"category":"ERRORS"`, `"message":"token=[REDACTED]"`, `"truncated":true`}},
		{name: "security", handler: getInstanceSecurityAudit(service), target: "/security", contains: []string{`"state":"WARNING"`, `"severity":"HIGH"`, `"partial":true`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.SetPathValue("instanceId", "inst_safe")
			res := httptest.NewRecorder()
			tt.handler.ServeHTTP(res, req)
			if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response = %d %q %s", res.Code, res.Header().Get("Content-Type"), res.Body.String())
			}
			for _, value := range tt.contains {
				if !strings.Contains(res.Body.String(), value) {
					t.Fatalf("body %s does not contain %s", res.Body.String(), value)
				}
			}
			for _, forbidden := range []string{"nativeId", "installation", `C:\\hermes`, "command", "environment"} {
				if strings.Contains(res.Body.String(), forbidden) {
					t.Fatalf("body exposed forbidden field/value %q: %s", forbidden, res.Body.String())
				}
			}
			if service.gotInstanceID != "inst_safe" {
				t.Fatalf("instance ID = %q", service.gotInstanceID)
			}
		})
	}
	if service.gotCategory != yorvaruntime.LogCategoryErrors {
		t.Fatalf("log category = %q", service.gotCategory)
	}
}

func TestManagementLogHandlerRequiresOneAllowlistedCategory(t *testing.T) {
	service := &b3FakeManagementHealthService{}
	for _, target := range []string{
		"/logs",
		"/logs?category=arbitrary/path",
		"/logs?category=ERRORS&category=MCP",
		"/logs?category=ERRORS&filter=secret",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.SetPathValue("instanceId", "inst_safe")
		res := httptest.NewRecorder()
		getInstanceLogSnapshot(service).ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), `"code":"INVALID_REQUEST"`) {
			t.Fatalf("%s response = %d %s", target, res.Code, res.Body.String())
		}
	}
	if service.calls != 0 {
		t.Fatalf("service calls = %d, want zero", service.calls)
	}
}

func TestManagementHealthHandlerMapsStableSafeErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "not found", err: app.ErrInstanceNotFound, wantStatus: http.StatusNotFound, wantCode: string(yorvaruntime.ErrorInstanceNotFound)},
		{name: "unavailable", err: app.ErrInstanceNotAvailable, wantStatus: http.StatusConflict, wantCode: string(yorvaruntime.ErrorInstanceNotAvailable)},
		{name: "capability", err: app.ErrManagementCapabilityUnsupported, wantStatus: http.StatusConflict, wantCode: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "runtime unsupported", err: app.ErrRuntimeNotSupported, wantStatus: http.StatusConflict, wantCode: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "query", err: app.ErrManagementQueryFailed, wantStatus: http.StatusServiceUnavailable, wantCode: "MANAGEMENT_QUERY_FAILED"},
		{name: "deadline", err: context.DeadlineExceeded, wantStatus: http.StatusServiceUnavailable, wantCode: "MANAGEMENT_QUERY_FAILED"},
		{name: "unknown", err: errors.New("secret raw adapter output"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &b3FakeManagementHealthService{err: tt.err}
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.SetPathValue("instanceId", "inst_safe")
			res := httptest.NewRecorder()
			getInstanceHealth(service).ServeHTTP(res, req)
			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", res.Code, tt.wantStatus, res.Body.String())
			}
			var body ErrorResponse
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != tt.wantCode || strings.Contains(res.Body.String(), "secret raw adapter output") {
				t.Fatalf("error body = %#v", body)
			}
		})
	}
}

func TestManagementHealthHandlerHonorsCancelledRequest(t *testing.T) {
	service := &b3FakeManagementHealthService{err: context.Canceled}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx)
	req.SetPathValue("instanceId", "inst_safe")
	res := httptest.NewRecorder()
	getInstanceHealth(service).ServeHTTP(res, req)
	if res.Body.Len() != 0 {
		t.Fatalf("cancelled response body = %s", res.Body.String())
	}
}

func TestManagementHealthHandlersNilServiceReturnSafeError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	getInstanceHealth(nil).ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError || !strings.Contains(res.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}
