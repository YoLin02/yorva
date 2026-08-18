package hermes

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/applog"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestArchiveFailureLogsExcludeSentinelSecrets(t *testing.T) {
	const (
		secretURL    = "https://files.example.invalid/hermes.zip?token=super-secret-token"
		secretUser   = `C:\Users\alice.secret`
		secretReason = "upstream said password=hunter2"
	)
	dataDir := t.TempDir()
	var stderr bytes.Buffer
	logger, closeLog := applog.New(&stderr, dataDir)
	defer closeLog()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, secretReason+" "+secretURL+" "+secretUser, http.StatusBadGateway)
	}))
	defer server.Close()

	installer := NewHostInstaller(dataDir).WithLogger(logger)
	installer.operationID = "op_secret"
	installer.archive.url = server.URL + "/hermes.zip?token=super-secret-token"
	installer.embeddedSourcePath = ""

	_, _, err := installer.resolveArchive(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected archive failure")
	}

	fileLog, readErr := os.ReadFile(applog.InstallLogPath(dataDir))
	if readErr != nil {
		t.Fatal(readErr)
	}
	httpLog := applog.ReadMatching(dataDir, "op_secret", 96*1024)
	for _, payload := range []string{stderr.String(), string(fileLog), httpLog} {
		for _, forbidden := range []string{secretURL, "super-secret-token", secretUser, "alice.secret", secretReason, "hunter2", server.URL} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("persisted/log output leaked %q: %s", forbidden, payload)
			}
		}
		if !strings.Contains(payload, "source.archive.") {
			t.Fatalf("missing structured archive event: %s", payload)
		}
		if !strings.Contains(payload, string(yorvaruntime.ErrorRuntimeInstallSourceUnavailable)) &&
			!strings.Contains(payload, string(yorvaruntime.ErrorRuntimeInstallIntegrityFailed)) &&
			!strings.Contains(payload, string(yorvaruntime.ErrorRuntimeInstallTimeout)) {
			t.Fatalf("missing stable error code: %s", payload)
		}
	}
}

func TestArchiveLogFieldsNeverIncludeRawError(t *testing.T) {
	var output bytes.Buffer
	installer := NewHostInstaller(t.TempDir())
	installer.logger = slog.New(slog.NewJSONHandler(&output, nil))
	installer.operationID = "op_fields"
	raw := errors.New("GET https://evil.example/file?token=abc C:\\Users\\bob failed: raw stderr boom")
	installer.debug("source.archive.integrity", archiveLogFields(installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, raw), sourceOriginOfficial)...)
	text := output.String()
	for _, forbidden := range []string{"https://evil.example", "token=abc", `C:\Users\bob`, "raw stderr boom"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("structured log leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, `"integrityMismatch":true`) || !strings.Contains(text, string(yorvaruntime.ErrorRuntimeInstallIntegrityFailed)) {
		t.Fatalf("structured fields missing: %s", text)
	}
}
