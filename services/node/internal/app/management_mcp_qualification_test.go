//go:build mcpqualification

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement"
	"github.com/YoLin02/yorva/services/node/internal/transport/httpapi"
)

func TestYORVAManagesProductionLocalTestMCPAcrossProfiles(t *testing.T) {
	ctx := context.Background()
	hermesHome, executable := qualificationHermesHome(t)
	t.Setenv("LOCALAPPDATA", filepath.Dir(hermesHome))
	manager, err := hermes.NewProductionProfileMCPManager()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	registry := yorvaruntime.NewRegistry()
	discoverer := qualificationDiscoverer{path: executable}
	if err := registry.Register(hermes.Kind, yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: hermes.Kind, Name: "Hermes Agent", Description: "qualification"},
		Discoverer: discoverer,
		Instances:  qualificationProfiles{},
		MCPRead:    manager,
		MCPMutate:  manager,
	}); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	localNode, err := db.LoadOrCreateNode(ctx, node.LocalMetadata{
		Name: "MCP qualification", Hostname: "localhost", Platform: "windows", Architecture: "amd64", NodeVersion: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	const installationID = "rtinst-mcp-qualification"
	if err := db.UpsertAcceptedInstallation(ctx, sqlite.AcceptedInstallation{
		ID: installationID, NodeID: localNode.ID, RuntimeKind: hermes.Kind, InstallPath: executable,
		Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported, Status: "ACCEPTED",
		LastDetectedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyInstanceSnapshot(ctx, installationID, []sqlite.InstanceSnapshotEntry{
		{NativeID: "default", Default: true},
		{NativeID: "work"},
	}, now); err != nil {
		t.Fatal(err)
	}
	instances, err := db.ListInstances(ctx, installationID)
	if err != nil || len(instances) != 2 {
		t.Fatalf("instances = %#v, %v", instances, err)
	}
	instanceID := make(map[string]string, len(instances))
	for _, instance := range instances {
		instanceID[instance.NativeID] = instance.ID
	}

	discovery := app.NewRuntimeDiscovery(registry, nil)
	inventory := app.NewInstanceInventory(discovery, db, localNode.ID)
	service, err := inventory.NewMCPManagement()
	if err != nil {
		t.Fatal(err)
	}
	const apiToken = "mcp-qualification-api-token"
	apiServer := httptest.NewServer(httpapi.NewHandler(apiToken, localNode, events.NewBroker(), discovery, nil, inventory, t.TempDir(), nil))
	t.Cleanup(apiServer.Close)
	definitionResponse := qualificationAPIRequest(t, apiServer.Client(), apiToken, http.MethodGet, apiServer.URL+"/api/v1/runtimes/hermes/mcp-definitions", "", "")
	var definitions httpapi.ManagementMCPPresetListResponse
	if err := json.Unmarshal(definitionResponse, &definitions); err != nil || len(definitions.Items) != 1 || definitions.Items[0].ID != mcpmanagement.YORVATestPresetID || definitions.Items[0].CredentialRequired {
		t.Fatalf("Runtime MCP definitions = %#v, %v", definitions, err)
	}

	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodPut,
		apiServer.URL+"/api/v1/instances/"+instanceID["default"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID,
		`{"enabledToolIds":["`+mcpmanagement.YORVATestToolID+`"]}`,
		"mcp-qualification-create-default")
	assertManagedReady(t, service, instanceID["default"])
	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodPatch,
		apiServer.URL+"/api/v1/instances/"+instanceID["default"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID,
		`{"enabledToolIds":["`+mcpmanagement.YORVATestToolID+`"]}`,
		"mcp-qualification-configure-default")
	assertManagedConfigured(t, service, instanceID["default"])
	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodPost,
		apiServer.URL+"/api/v1/instances/"+instanceID["default"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID+"/test", `{}`,
		"mcp-qualification-retest-default")
	assertManagedReady(t, service, instanceID["default"])
	defaultConfig := readQualificationConfig(t, filepath.Join(hermesHome, "config.yaml"))
	if !strings.Contains(defaultConfig, mcpmanagement.YORVATestEndpoint) || strings.Contains(defaultConfig, "authorization:") || strings.Contains(defaultConfig, "command:") || strings.Contains(defaultConfig, "args:") || strings.Contains(defaultConfig, "env:") {
		t.Fatalf("default Profile config is not a closed HTTPS definition: %q", defaultConfig)
	}

	// Modify the binding by assigning the same Runtime definition to a second
	// exact Hermes Profile and removing it from the first.
	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodPut,
		apiServer.URL+"/api/v1/instances/"+instanceID["work"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID,
		`{"enabledToolIds":["`+mcpmanagement.YORVATestToolID+`"]}`,
		"mcp-qualification-bind-work")
	assertManagedReady(t, service, instanceID["work"])
	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodDelete,
		apiServer.URL+"/api/v1/instances/"+instanceID["default"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID, `{}`,
		"mcp-qualification-unbind-default")
	assertNoMCP(t, service, instanceID["default"])

	startAndWaitMCPHTTP(t, db, apiServer.Client(), apiToken, http.MethodDelete,
		apiServer.URL+"/api/v1/instances/"+instanceID["work"]+"/mcp-bindings/"+mcpmanagement.YORVATestPresetID, `{}`,
		"mcp-qualification-delete-work")
	assertNoMCP(t, service, instanceID["work"])
	bindings, err := db.ListManagedMCPBindings(ctx, instanceID["default"])
	if err != nil || len(bindings) != 0 {
		t.Fatalf("default managed bindings after remove = %#v, %v", bindings, err)
	}
	bindings, err = db.ListManagedMCPBindings(ctx, instanceID["work"])
	if err != nil || len(bindings) != 0 {
		t.Fatalf("work managed bindings after remove = %#v, %v", bindings, err)
	}

}

type qualificationDiscoverer struct{ path string }

func (d qualificationDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	candidate := yorvaruntime.Candidate{Path: d.path, Version: "0.20.5", State: yorvaruntime.DiscoverySupported}
	return yorvaruntime.Discovery{
		RuntimeKind: hermes.Kind, State: yorvaruntime.DiscoverySupported, Selected: &candidate,
		Candidates: []yorvaruntime.Candidate{candidate}, DetectedAt: time.Now().UTC(), SupportedRange: ">=0.20.0 <0.21.0",
	}, nil
}

type qualificationProfiles struct{ hermes.InstanceManager }

func (qualificationProfiles) List(context.Context, string) ([]yorvaruntime.NativeInstance, error) {
	return []yorvaruntime.NativeInstance{{NativeID: "default", Default: true}, {NativeID: "work"}}, nil
}

func qualificationHermesHome(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "hermes")
	for _, path := range []string{root, filepath.Join(root, "profiles", "work"), filepath.Join(root, "bin")} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{filepath.Join(root, "config.yaml"), filepath.Join(root, "profiles", "work", "config.yaml")} {
		if err := os.WriteFile(path, []byte("model: qualification\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executable := filepath.Join(root, "bin", "hermes.exe")
	if err := os.WriteFile(executable, []byte("qualification launcher"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, executable
}

func startAndWaitMCPHTTP(t *testing.T, db *sqlite.Database, client *http.Client, token, method, url, body, key string) {
	t.Helper()
	payload := qualificationAPIRequest(t, client, token, method, url, body, key)
	var started httpapi.OperationResponse
	if err := json.Unmarshal(payload, &started); err != nil || started.ID == "" {
		t.Fatalf("decode MCP Operation: %#v, %v", started, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		stored, readErr := db.GetOperation(context.Background(), started.ID)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if operation.IsTerminal(stored.Status) {
			if stored.Status != operation.StatusSucceeded {
				t.Fatalf("MCP Operation = %#v", stored)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("MCP qualification Operation did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func qualificationAPIRequest(t *testing.T, client *http.Client, token, method, url, body, key string) []byte {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(payload) > 64*1024 {
		t.Fatalf("read qualification API response: %v", err)
	}
	want := http.StatusOK
	if method != http.MethodGet {
		want = http.StatusAccepted
	}
	if response.StatusCode != want {
		t.Fatalf("qualification API status = %d, want %d: %s", response.StatusCode, want, payload)
	}
	return payload
}

func qualificationAPIRequestStatus(t *testing.T, client *http.Client, token, method, url, body string, want int) []byte {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(payload) > 64*1024 {
		t.Fatal(err)
	}
	if response.StatusCode != want {
		t.Fatalf("definition API status = %d, want %d: %s", response.StatusCode, want, payload)
	}
	return payload
}

func assertManagedReady(t *testing.T, service *app.MCPManagement, instanceID string) {
	assertManagedReadyID(t, service, instanceID, "yorva-mcp-test")
}
func assertManagedReadyID(t *testing.T, service *app.MCPManagement, instanceID, serverID string) {
	t.Helper()
	servers, err := service.ListMCPServers(context.Background(), instanceID)
	if err != nil || len(servers) != 1 || servers[0].ID != serverID || servers[0].Ownership != "YORVA_MANAGED" || servers[0].State != yorvaruntime.MCPReady || servers[0].ReadyAt == nil {
		t.Fatalf("managed READY readback = %#v, %v", servers, err)
	}
}

func assertManagedConfigured(t *testing.T, service *app.MCPManagement, instanceID string) {
	t.Helper()
	servers, err := service.ListMCPServers(context.Background(), instanceID)
	if err != nil || len(servers) != 1 || servers[0].ID != mcpmanagement.YORVATestPresetID || servers[0].Ownership != "YORVA_MANAGED" || servers[0].State != yorvaruntime.MCPConfigured || len(servers[0].EnabledToolIDs) != 1 || servers[0].EnabledToolIDs[0] != mcpmanagement.YORVATestToolID {
		t.Fatalf("managed CONFIGURED readback = %#v, %v", servers, err)
	}
}

func assertNoMCP(t *testing.T, service *app.MCPManagement, instanceID string) {
	t.Helper()
	servers, err := service.ListMCPServers(context.Background(), instanceID)
	if err != nil || len(servers) != 0 {
		t.Fatalf("post-remove authoritative readback = %#v, %v", servers, err)
	}
}

func readQualificationConfig(t *testing.T, path string) string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}
