//go:build windows

package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestRealWindowsHermesInstallationSmoke(t *testing.T) {
	if os.Getenv("YORVA_REAL_HERMES_SMOKE") != "1" {
		t.Skip("set YORVA_REAL_HERMES_SMOKE=1 to inspect the host's documented Hermes installation")
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		t.Fatal("LOCALAPPDATA is required for the Windows Hermes smoke")
	}
	installRoot := filepath.Join(localAppData, "hermes", "hermes-agent")
	if !isDirectory(installRoot) {
		t.Fatalf("expected real Hermes install root at %s", installRoot)
	}

	got, err := NewDetector().Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got.State == yorvaruntime.DiscoveryNotInstalled {
		t.Fatalf("Detect() returned NOT_INSTALLED for existing official root %s", installRoot)
	}
	if got.State != yorvaruntime.DiscoverySupported || got.Selected == nil {
		t.Fatalf("Detect() state/selected = %s/%#v, want runnable supported Hermes", got.State, got.Selected)
	}
	for _, candidate := range got.Candidates {
		if strings.EqualFold(filepath.Base(candidate.Path), "hermes-agent.exe") {
			t.Fatalf("Detect() treated forbidden entrypoint as a CLI candidate: %s", candidate.Path)
		}
	}
	t.Logf(
		"real Hermes discovery state=%s version=%s command=%s candidates=%d",
		got.State,
		got.Selected.Version,
		got.Selected.Path,
		len(got.Candidates),
	)
}
