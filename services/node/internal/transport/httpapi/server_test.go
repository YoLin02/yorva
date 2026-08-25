package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const testToken = "test-token"

var testNode = node.Node{
	ID:           "node_test",
	Name:         "TEST-NODE",
	Hostname:     "TEST-NODE",
	Platform:     "windows",
	Architecture: "amd64",
	NodeVersion:  "0.0.0-test",
	CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
}

type fakeRuntimeDiscovery struct {
	result yorvaruntime.Discovery
	err    error
	ctx    chan context.Context
}

type managementRoutingInventory struct {
	fakeInstanceInventory
	target     app.ManagementTarget
	instanceID string
}

func (f *managementRoutingInventory) ResolveManagementTarget(_ context.Context, instanceID string) (app.ManagementTarget, error) {
	f.instanceID = instanceID
	return f.target, nil
}

type routingSkillReader struct {
	skill yorvaruntime.Skill
}

func (f routingSkillReader) ListSkills(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.Skill, error) {
	return []yorvaruntime.Skill{f.skill}, nil
}

func (f routingSkillReader) InspectSkill(context.Context, yorvaruntime.Installation, string, string) (yorvaruntime.Skill, error) {
	return f.skill, nil
}

type routingMCPReader struct {
	server yorvaruntime.MCPServer
	preset yorvaruntime.MCPPreset
}

type routingHealthInspector struct {
	observation yorvaruntime.HealthObservation
}

func (f routingHealthInspector) InspectRuntimeHealth(context.Context, yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	return f.observation, nil
}

func (f routingHealthInspector) InspectInstanceHealth(context.Context, yorvaruntime.Installation, string) (yorvaruntime.HealthObservation, error) {
	return f.observation, nil
}

type routingLogReader struct {
	snapshot yorvaruntime.LogSnapshot
}

func (f routingLogReader) ReadLogSnapshot(context.Context, yorvaruntime.Installation, string, yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	return f.snapshot, nil
}

func (f routingMCPReader) ListMCPServers(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPServer, error) {
	return []yorvaruntime.MCPServer{f.server}, nil
}

func (f routingMCPReader) ListMCPPresets(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPPreset, error) {
	return []yorvaruntime.MCPPreset{f.preset}, nil
}

func (d fakeRuntimeDiscovery) Detect(ctx context.Context, _ yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	if d.ctx != nil {
		d.ctx <- ctx
	}
	return d.result, d.err
}

func newTestHandler(services ...RuntimeDiscoveryService) http.Handler {
	var service RuntimeDiscoveryService = fakeRuntimeDiscovery{result: yorvaruntime.Discovery{
		RuntimeKind:    "hermes",
		State:          yorvaruntime.DiscoverySupported,
		DetectedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SupportedRange: ">=0.20.2 <0.21.0",
	}}
	if len(services) == 1 {
		service = services[0]
	}
	return NewHandler(testToken, testNode, events.NewBroker(), service, nil, nil, "", nil)
}

func TestHealthIsMinimalAndUnauthenticated(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, forbidden := range []string{"token", "dataDir", "environment", "arguments"} {
		if _, exists := body[forbidden]; exists || strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("health response exposes %q: %s", forbidden, response.Body.String())
		}
	}
	if body["status"] != "ok" || body["service"] != "yorvad" {
		t.Fatalf("unexpected health response: %#v", body)
	}
}

func TestNodeRequiresValidBearerToken(t *testing.T) {
	for _, test := range []struct {
		name   string
		header string
		status int
	}{
		{name: "missing", status: http.StatusUnauthorized},
		{name: "invalid", header: "Bearer invalid", status: http.StatusUnauthorized},
		{name: "valid", header: "Bearer " + testToken, status: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/node", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()

			newTestHandler().ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusUnauthorized && test.header != "" && strings.Contains(response.Body.String(), test.header) {
				t.Fatalf("response leaks authorization value: %s", response.Body.String())
			}
			if test.status == http.StatusUnauthorized {
				assertProtocolError(t, response, "UNAUTHORIZED")
			}
		})
	}
}

func TestEventsRequiresAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestEventStreamCancellationReleasesSubscriber(t *testing.T) {
	broker := events.NewBroker()
	server := httptest.NewServer(NewHandler(testToken, testNode, broker, fakeRuntimeDiscovery{}, nil, nil, "", nil))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("create events request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+testToken)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || line != ": connected\n" {
		_ = response.Body.Close()
		t.Fatalf("initial stream line = %q, error = %v", line, err)
	}

	cancel()
	_ = response.Body.Close()
	deadline := time.Now().Add(2 * time.Second)
	for broker.SubscriberCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if broker.SubscriberCount() != 0 {
		t.Fatalf("subscriber count = %d after cancellation, want 0", broker.SubscriberCount())
	}
}

func TestOriginPolicyAllowsOnlyDesktopOrigins(t *testing.T) {
	t.Run("allowed preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/node", nil)
		request.Header.Set("Origin", "http://127.0.0.1:1420")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:1420" {
			t.Fatalf("unexpected allow origin header: %q", response.Header().Get("Access-Control-Allow-Origin"))
		}
		if response.Header().Get("Allow") != "GET, OPTIONS" {
			t.Fatalf("unexpected Allow header: %q", response.Header().Get("Allow"))
		}
	})

	t.Run("rejected origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		request.Header.Set("Origin", "https://example.com")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("rejected origin received an allow-origin header")
		}
		assertProtocolError(t, response, "ORIGIN_NOT_ALLOWED")
	})

	t.Run("unknown preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/missing", nil)
		request.Header.Set("Origin", "http://tauri.localhost")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
		assertProtocolError(t, response, "NOT_FOUND")
	})
}

func TestRoutingErrorsUseStableProtocolEnvelope(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		path   string
		status int
		code   string
	}{
		{name: "not found", method: http.MethodGet, path: "/api/v1/missing", status: http.StatusNotFound, code: "NOT_FOUND"},
		{name: "method not allowed", method: http.MethodPost, path: "/api/v1/node", status: http.StatusMethodNotAllowed, code: "METHOD_NOT_ALLOWED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			newTestHandler().ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			assertProtocolError(t, response, test.code)
			if test.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, OPTIONS" {
				t.Fatalf("unexpected Allow header: %q", response.Header().Get("Allow"))
			}
		})
	}
}

func TestPhase7ReadOnlyManagementRoutesAreAuthenticatedAndTargetExactInstance(t *testing.T) {
	now := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	inventory := &managementRoutingInventory{target: app.ManagementTarget{
		Installation: yorvaruntime.Installation{RuntimeKind: "hermes", Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported},
		NativeID:     "profile-a",
		Bundle: yorvaruntime.Bundle{
			Health: routingHealthInspector{observation: yorvaruntime.HealthObservation{
				State: yorvaruntime.HealthHealthy, Findings: []yorvaruntime.HealthFinding{{Code: "gateway", State: yorvaruntime.HealthHealthy}}, ObservedAt: now,
			}},
			Logs: routingLogReader{snapshot: yorvaruntime.LogSnapshot{
				Category: yorvaruntime.LogCategoryErrors, Entries: []yorvaruntime.LogEntry{{Timestamp: now, Message: "bounded safe log"}}, ObservedAt: now,
			}},
			SkillRead: routingSkillReader{skill: yorvaruntime.Skill{
				ID: "skill-a", SourceID: "source-a", Version: "1.0.0", InstallationState: yorvaruntime.SkillInstalled,
				EnabledState: yorvaruntime.SkillEnabled, ScanState: yorvaruntime.SkillScanClean,
			}},
			MCPRead: routingMCPReader{
				server: yorvaruntime.MCPServer{ID: "server-a", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: now},
				preset: yorvaruntime.MCPPreset{ID: "preset-a", DisplayName: "Approved A"},
			},
		},
	}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "", nil)

	tests := []struct {
		path       string
		bodyMarker string
	}{
		{path: "/api/v1/instances/inst-1/health", bodyMarker: `"state":"HEALTHY"`},
		{path: "/api/v1/instances/inst-1/logs?category=ERRORS", bodyMarker: `"message":"bounded safe log"`},
		{path: "/api/v1/instances/inst-1/skills", bodyMarker: `"id":"skill-a"`},
		{path: "/api/v1/instances/inst-1/skills/skill-a", bodyMarker: `"id":"skill-a"`},
		{path: "/api/v1/instances/inst-1/mcp-servers", bodyMarker: `"id":"server-a"`},
		{path: "/api/v1/instances/inst-1/mcp-catalog", bodyMarker: `"id":"preset-a"`},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			unauthorized := httptest.NewRequest(http.MethodGet, test.path, nil)
			unauthorizedResponse := httptest.NewRecorder()
			handler.ServeHTTP(unauthorizedResponse, unauthorized)
			if unauthorizedResponse.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated status = %d", unauthorizedResponse.Code)
			}

			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+testToken)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.bodyMarker) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if inventory.instanceID != "inst-1" {
				t.Fatalf("resolved instance ID = %q", inventory.instanceID)
			}
		})
	}
}

func TestPhase7ReadOnlyManagementRoutesKeepCapabilityFalseStable(t *testing.T) {
	inventory := &managementRoutingInventory{target: app.ManagementTarget{Bundle: yorvaruntime.Bundle{}}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "", nil)
	for _, path := range []string{
		"/api/v1/instances/inst-1/health",
		"/api/v1/instances/inst-1/logs?category=ERRORS",
		"/api/v1/instances/inst-1/skills",
		"/api/v1/instances/inst-1/skills/skill-a",
		"/api/v1/instances/inst-1/mcp-servers",
		"/api/v1/instances/inst-1/mcp-catalog",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+testToken)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			assertProtocolError(t, response, string(yorvaruntime.ErrorCapabilityNotSupported))
		})
	}
}

func TestPhase7ReadOnlyManagementRouteContractIsGetOnly(t *testing.T) {
	inventory := &managementRoutingInventory{}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "", nil)
	for _, path := range []string{
		"/api/v1/instances/inst-1/health",
		"/api/v1/instances/inst-1/logs?category=ERRORS",
		"/api/v1/instances/inst-1/skills",
		"/api/v1/instances/inst-1/skills/skill-a",
		"/api/v1/instances/inst-1/mcp-servers",
		"/api/v1/instances/inst-1/mcp-catalog",
	} {
		t.Run(path, func(t *testing.T) {
			preflight := httptest.NewRequest(http.MethodOptions, path, nil)
			preflight.Header.Set("Origin", "http://tauri.localhost")
			preflightResponse := httptest.NewRecorder()
			handler.ServeHTTP(preflightResponse, preflight)
			if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Allow") != "GET, OPTIONS" {
				t.Fatalf("preflight = %d Allow %q", preflightResponse.Code, preflightResponse.Header().Get("Allow"))
			}

			post := httptest.NewRequest(http.MethodPost, path, nil)
			post.Header.Set("Authorization", "Bearer "+testToken)
			postResponse := httptest.NewRecorder()
			handler.ServeHTTP(postResponse, post)
			if postResponse.Code != http.StatusMethodNotAllowed || postResponse.Header().Get("Allow") != "GET, OPTIONS" {
				t.Fatalf("POST = %d Allow %q", postResponse.Code, postResponse.Header().Get("Allow"))
			}
		})
	}
}

func TestPhase7InstanceLogRouteRejectsNonClosedQuery(t *testing.T) {
	inventory := &managementRoutingInventory{target: app.ManagementTarget{Bundle: yorvaruntime.Bundle{Logs: routingLogReader{}}}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "", nil)
	for _, path := range []string{
		"/api/v1/instances/inst-1/logs",
		"/api/v1/instances/inst-1/logs?category=UNKNOWN",
		"/api/v1/instances/inst-1/logs?category=ERRORS&category=MCP",
		"/api/v1/instances/inst-1/logs?category=ERRORS&filter=secret",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+testToken)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			assertProtocolError(t, response, "INVALID_REQUEST")
		})
	}
}

func TestPhase7SecurityAuditReadRemainsUnregistered(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/security-audit", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("security audit status = %d, want 404", response.Code)
	}
	assertProtocolError(t, response, "NOT_FOUND")
}

func TestPhase7RuntimeHealthRemainsUnregisteredWithoutRuntimeTarget(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/health", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("runtime health status = %d, want 404", response.Code)
	}
	assertProtocolError(t, response, "NOT_FOUND")
}

func assertProtocolError(t *testing.T, response *httptest.ResponseRecorder, code string) {
	t.Helper()
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
	var body ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != code || body.Error.Message == "" || body.Error.Details == nil {
		t.Fatalf("invalid error response: %#v", body)
	}
}
