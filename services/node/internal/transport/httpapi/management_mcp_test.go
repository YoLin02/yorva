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

func TestDecodeMCPInstallAcceptsOnlyCredentialAndAllowlistedToolIDs(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"credential":"test-token","enabledToolIds":["tool-a"]}`))
	credential, _, tools, err := decodeMCPInstall(request, "preset-a")
	if err != nil || string(credential) != "test-token" || len(tools) != 1 || tools[0] != "tool-a" {
		t.Fatalf("decodeMCPInstall() = %q %#v, %v", credential, tools, err)
	}
	clear(credential)
	emptyScope := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"enabledToolIds":[]}`))
	if _, _, _, err := decodeMCPInstall(emptyScope, "preset-a"); err != nil {
		t.Fatalf("empty Tool Scope should allow discovery: %v", err)
	}

	for _, field := range []string{"command", "args", "env", "headers", "url", "path", "json"} {
		body := `{"enabledToolIds":["tool-a"],"` + field + `":"forbidden"}`
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if _, _, _, err := decodeMCPInstall(request, "preset-a"); err == nil {
			t.Fatalf("forbidden field %q was accepted", field)
		}
	}
}

type managementMCPReadServiceFake struct {
	servers    []app.MCPServerView
	presets    []app.MCPPresetView
	serversErr error
	presetsErr error
	instanceID string
}

type runtimeMCPDefinitionServiceFake struct {
	items     []app.MCPPresetView
	runtimeID string
}

func (f *runtimeMCPDefinitionServiceFake) ListRuntimeMCPDefinitions(_ context.Context, runtimeID string) ([]app.MCPPresetView, error) {
	f.runtimeID = runtimeID
	return f.items, nil
}

func (f *managementMCPReadServiceFake) ListMCPServers(_ context.Context, instanceID string) ([]app.MCPServerView, error) {
	f.instanceID = instanceID
	return f.servers, f.serversErr
}

func (f *managementMCPReadServiceFake) ListMCPPresets(_ context.Context, instanceID string) ([]app.MCPPresetView, error) {
	f.instanceID = instanceID
	return f.presets, f.presetsErr
}

func TestListMCPServersReturnsClosedSafeStateAndTimeSemantics(t *testing.T) {
	readyAt := time.Date(2026, 8, 25, 4, 5, 6, 0, time.FixedZone("offset", 8*60*60))
	observedAt := readyAt.Add(time.Second)
	service := &managementMCPReadServiceFake{servers: []app.MCPServerView{
		{ID: "configured-server", PresetID: "preset-a", Ownership: "EXTERNAL", State: yorvaruntime.MCPConfigured, ObservedAt: observedAt},
		{ID: "ready-server", PresetID: "preset-b", Ownership: "YORVA_MANAGED", State: yorvaruntime.MCPReady, ReadyAt: &readyAt, ObservedAt: observedAt},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/mcp-servers", nil)
	request.SetPathValue("instanceId", "inst-1")
	response := httptest.NewRecorder()

	listMCPServers(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response = %d, Content-Type %q", response.Code, response.Header().Get("Content-Type"))
	}
	if service.instanceID != "inst-1" {
		t.Fatalf("instance ID = %q", service.instanceID)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("items = %#v", body.Items)
	}
	wantKeys := map[string]bool{"id": true, "presetId": true, "ownership": true, "enabledToolIds": true, "state": true, "readyAt": true, "observedAt": true}
	for _, item := range body.Items {
		if len(item) != len(wantKeys) {
			t.Fatalf("MCP server response is not closed: %#v", item)
		}
		for key := range item {
			if !wantKeys[key] {
				t.Fatalf("unexpected MCP server field %q", key)
			}
		}
	}
	if body.Items[0]["state"] != string(yorvaruntime.MCPConfigured) || body.Items[0]["readyAt"] != nil {
		t.Fatalf("CONFIGURED response = %#v", body.Items[0])
	}
	if body.Items[0]["ownership"] != "EXTERNAL" || body.Items[1]["ownership"] != "YORVA_MANAGED" {
		t.Fatalf("ownership response = %#v", body.Items)
	}
	if body.Items[1]["state"] != string(yorvaruntime.MCPReady) || body.Items[1]["readyAt"] != readyAt.UTC().Format(time.RFC3339) || body.Items[1]["observedAt"] != observedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("READY response = %#v", body.Items[1])
	}

	lower := strings.ToLower(response.Body.String())
	for _, forbidden := range []string{"url", "header", "command", "environment", "path", "secret", "description"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("response exposed forbidden %q field or value: %s", forbidden, response.Body.String())
		}
	}
}

func TestListMCPPresetsReturnsOnlyReviewedCatalogProjection(t *testing.T) {
	service := &managementMCPReadServiceFake{presets: []app.MCPPresetView{{ID: "preset-a", DisplayName: "Approved A", Description: "Reviewed preset", HomepageURL: "https://example.invalid/", DocumentationURL: "https://example.invalid/docs"}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst-1/mcp-catalog", nil)
	request.SetPathValue("instanceId", "inst-1")
	response := httptest.NewRecorder()

	listMCPPresets(service).ServeHTTP(response, request)

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != http.StatusOK || len(body.Items) != 1 || body.Items[0]["id"] != "preset-a" || body.Items[0]["displayName"] != "Approved A" {
		t.Fatalf("catalog response = %d %#v", response.Code, body.Items)
	}
	for key := range body.Items[0] {
		if key != "id" && key != "displayName" && key != "description" && key != "homepageUrl" && key != "documentationUrl" && key != "allowedToolIds" && key != "credentialRequired" && key != "source" && key != "editable" {
			t.Fatalf("catalog exposed field %q", key)
		}
	}
}

func TestListRuntimeMCPDefinitionsUsesRuntimeScopeAndOmitsCustomExecutionMaterial(t *testing.T) {
	service := &runtimeMCPDefinitionServiceFake{items: []app.MCPPresetView{{ID: "preset-a", DisplayName: "Approved A", Transport: yorvaruntime.MCPTransportStdio, Command: "unsafe-command", Args: []string{"--unsafe"}, Environment: []yorvaruntime.MCPConfigValue{{Name: "TOKEN", Secret: true}}, Headers: []yorvaruntime.MCPConfigValue{{Name: "Authorization", Secret: true}}}}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/mcp-definitions", nil)
	request.SetPathValue("runtimeId", "hermes")
	response := httptest.NewRecorder()
	listRuntimeMCPDefinitions(service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.runtimeID != "hermes" || !strings.Contains(response.Body.String(), `"id":"preset-a"`) {
		t.Fatalf("runtime definitions = %d %q, runtime %q", response.Code, response.Body.String(), service.runtimeID)
	}
	for _, forbidden := range []string{"transport", "command", "args", "environment", "headers", "unsafe-command", "Authorization", "TOKEN"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("runtime definition exposed %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestMCPListHandlersReturnEmptyArrays(t *testing.T) {
	service := &managementMCPReadServiceFake{}
	for name, handler := range map[string]http.Handler{
		"servers": listMCPServers(service),
		"presets": listMCPPresets(service),
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.SetPathValue("instanceId", "inst-1")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
				t.Fatalf("empty response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestMCPListHandlersUseStableSafeErrors(t *testing.T) {
	tests := []struct {
		name    string
		service ManagementMCPReadService
		status  int
		code    string
	}{
		{name: "nil service", status: http.StatusConflict, code: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "capability false", service: &managementMCPReadServiceFake{serversErr: app.ErrManagementCapabilityUnsupported}, status: http.StatusConflict, code: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "instance missing", service: &managementMCPReadServiceFake{serversErr: app.ErrInstanceNotFound}, status: http.StatusNotFound, code: string(yorvaruntime.ErrorInstanceNotFound)},
		{name: "query failed", service: &managementMCPReadServiceFake{serversErr: app.ErrManagementQueryFailed}, status: http.StatusServiceUnavailable, code: "MANAGEMENT_QUERY_FAILED"},
		{name: "unowned cancellation", service: &managementMCPReadServiceFake{serversErr: context.Canceled}, status: http.StatusServiceUnavailable, code: "MANAGEMENT_QUERY_FAILED"},
		{name: "raw adapter error", service: &managementMCPReadServiceFake{serversErr: errors.New("secret-canary https://private.invalid/path")}, status: http.StatusServiceUnavailable, code: "MANAGEMENT_QUERY_FAILED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.SetPathValue("instanceId", "inst-1")
			response := httptest.NewRecorder()
			listMCPServers(test.service).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			var body ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Error.Code != test.code || body.Error.Details == nil {
				t.Fatalf("error response = %#v", body)
			}
			lower := strings.ToLower(response.Body.String())
			if strings.Contains(lower, "secret-canary") || strings.Contains(lower, "private.invalid") {
				t.Fatalf("error leaked internal detail: %s", response.Body.String())
			}
		})
	}
}
