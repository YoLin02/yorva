package hermes

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestProfileDiagnosticsProjectsPartialHealthAndRedactedExactProfileLogs(t *testing.T) {
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles", "work")
	writeProfileResourceFixture(t, filepath.Join(profileRoot, "gateway_state.json"), `{"gateway_state":"running","updated_at":"2026-08-26T03:00:00Z","argv":["must-not-project"]}`)
	writeProfileResourceFixture(t, filepath.Join(profileRoot, "logs", "errors.log"), "2026-08-26 11:00:00 ERROR Authorization: Bearer abcdefghijklmnopqrstuvwxyz\n2026-08-26 11:01:00 ERROR failed path C:\\Users\\Alice\\secret\n")
	reader := newProfileResourceReaderAt(root)
	installation := profileResourceInstallation(root)

	health, err := reader.InspectInstanceHealth(context.Background(), installation, "work")
	if err != nil || health.State != yorvaruntime.HealthHealthy || !health.Partial || len(health.Findings) != 1 || health.Findings[0].Code != "gateway_state" || !health.ObservedAt.Equal(time.Date(2026, 8, 26, 3, 0, 0, 0, time.UTC)) {
		t.Fatalf("health = %#v, %v", health, err)
	}

	logs, err := reader.ReadLogSnapshot(context.Background(), installation, "work", yorvaruntime.LogCategoryErrors)
	if err != nil || logs.Category != yorvaruntime.LogCategoryErrors || len(logs.Entries) != 2 {
		t.Fatalf("logs = %#v, %v", logs, err)
	}
	joined := logs.Entries[0].Message + logs.Entries[1].Message
	for _, prohibited := range []string{"abcdefghijklmnopqrstuvwxyz", `C:\Users\Alice`} {
		if strings.Contains(joined, prohibited) {
			t.Fatalf("log output contains %q: %q", prohibited, joined)
		}
	}
	if !strings.Contains(joined, "[REDACTED]") || !strings.Contains(joined, "[PATH]") {
		t.Fatalf("log output was not redacted: %q", joined)
	}
	if logs.Entries[0].Timestamp.IsZero() || logs.ObservedAt.IsZero() {
		t.Fatalf("log timestamps are missing: %#v", logs)
	}
}

func TestProfileDiagnosticsReturnsEmptySnapshotWhenFixedLogIsAbsent(t *testing.T) {
	root := t.TempDir()
	writeProfileResourceFixture(t, filepath.Join(root, "config.yaml"), "model: test\n")
	reader := newProfileResourceReaderAt(root)
	logs, err := reader.ReadLogSnapshot(context.Background(), profileResourceInstallation(root), "default", yorvaruntime.LogCategoryGateway)
	if err != nil || len(logs.Entries) != 0 || logs.ObservedAt.IsZero() {
		t.Fatalf("logs = %#v, %v", logs, err)
	}
}
