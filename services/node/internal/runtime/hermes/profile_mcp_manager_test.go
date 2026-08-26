package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement"
	"github.com/YoLin02/yorva/services/node/internal/testsupport/mcptestserver"
)

func TestProfileMCPManagerQualificationPresetLifecycle(t *testing.T) {
	server := mcptestserver.Start(t)

	root := t.TempDir()
	writeProfileResourceFixture(t, filepath.Join(root, "config.yaml"), "model: test\n")
	writeProfileResourceFixture(t, filepath.Join(root, "profiles", "work", "config.yaml"), "model: test\n")
	reader := newProfileResourceReaderAt(root)
	manager := newProfileMCPManager(reader, mcpmanagement.NewQualificationRegistry(), server.Client())
	clock := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { clock = clock.Add(time.Millisecond); return clock }
	installation := profileResourceInstallation(root)

	presets, err := manager.ListMCPPresets(context.Background(), installation, "default")
	if err != nil || len(presets) != 1 || presets[0].ID != "yorva-mcp-test" || len(presets[0].AllowedToolIDs) != 1 || !presets[0].CredentialRequired {
		t.Fatalf("presets = %#v, %v", presets, err)
	}
	created, err := manager.InstallMCPPreset(context.Background(), installation, "default", yorvaruntime.MCPInstallRequest{PresetID: "yorva-mcp-test"}, nil)
	if err != nil || created.State != yorvaruntime.MCPAuthRequired {
		t.Fatalf("InstallMCPPreset() = %#v, %v", created, err)
	}
	config, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	if err != nil || !strings.Contains(string(config), "https://mcp-test.yorva.invalid/mcp") || !strings.Contains(string(config), "Bearer ${MCP_YORVA_TEST_API_KEY}") || strings.Contains(string(config), mcptestserver.Credential) || strings.Contains(string(config), "command:") || strings.Contains(string(config), "args:") || strings.Contains(string(config), "env:") {
		t.Fatalf("safe config = %q, %v", config, err)
	}
	authenticated, err := manager.AuthenticateMCP(context.Background(), installation, "default", yorvaruntime.MCPAuthenticateRequest{ServerID: "yorva-mcp-test", Credential: []byte(mcptestserver.Credential)}, nil)
	if err != nil || authenticated.State != yorvaruntime.MCPConfigured {
		t.Fatalf("AuthenticateMCP() = %#v, %v", authenticated, err)
	}
	result, err := manager.TestMCP(context.Background(), installation, "default", "yorva-mcp-test", nil)
	if err != nil || result.State != yorvaruntime.MCPReady || len(result.ToolIDs) != 1 || result.ToolIDs[0] != mcptestserver.ToolPing || server.RequestCount() != 3 {
		t.Fatalf("TestMCP() = %#v, requests=%d, %v", result, server.RequestCount(), err)
	}
	observed, err := manager.ListMCPServers(context.Background(), installation, "default")
	if err != nil || len(observed) != 1 || observed[0].State != yorvaruntime.MCPReady || observed[0].ReadyAt == nil {
		t.Fatalf("ready readback = %#v, %v", observed, err)
	}
	if _, err := manager.InstallMCPPreset(context.Background(), installation, "work", yorvaruntime.MCPInstallRequest{PresetID: "yorva-mcp-test"}, nil); err != nil {
		t.Fatalf("bind second Profile: %v", err)
	}
	if _, err := manager.AuthenticateMCP(context.Background(), installation, "work", yorvaruntime.MCPAuthenticateRequest{ServerID: "yorva-mcp-test", Credential: []byte(mcptestserver.Credential)}, nil); err != nil {
		t.Fatalf("authenticate second Profile: %v", err)
	}
	if _, err := manager.RemoveMCP(context.Background(), installation, "default", "yorva-mcp-test", nil); err != nil {
		t.Fatalf("RemoveMCP(): %v", err)
	}
	after, err := manager.ListMCPServers(context.Background(), installation, "default")
	if err != nil || len(after) != 0 {
		t.Fatalf("post-remove readback = %#v, %v", after, err)
	}
	work, err := manager.ListMCPServers(context.Background(), installation, "work")
	if err != nil || len(work) != 1 || work[0].State != yorvaruntime.MCPConfigured {
		t.Fatalf("second binding = %#v, %v", work, err)
	}
}
