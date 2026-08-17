package hermes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestResolveArchiveFallsBackOnceOnTransportFailure(t *testing.T) {
	bundled := writeOfficialSizedStub(t, []byte("bundled-bytes"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	installer := NewHostInstaller(t.TempDir()).WithEmbeddedSource(bundled)
	installer.archive.url = server.URL
	installer.archive.http = server.Client()
	installer.verifyArchive = func(path string) error {
		if path != bundled {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errPlatform)
		}
		return nil
	}
	path, origin, err := installer.resolveArchive(context.Background(), t.TempDir())
	if err != nil || path != bundled || origin != sourceOriginBundled {
		t.Fatalf("path=%s origin=%s err=%v", path, origin, err)
	}
}

func TestResolveArchiveDoesNotFallBackOnIntegrityFailure(t *testing.T) {
	bundled := writeOfficialSizedStub(t, []byte("bundled-bytes"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("tampered-archive"))
	}))
	t.Cleanup(server.Close)
	installer := NewHostInstaller(t.TempDir()).WithEmbeddedSource(bundled)
	installer.archive.url = server.URL
	installer.archive.http = server.Client()
	_, _, err := installer.resolveArchive(context.Background(), t.TempDir())
	if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyMaterializesSourceWithoutOfficialRepositoryStage(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("installer Apply is Windows user-scope")
	}
	script := []byte("official-script\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(script)
	}))
	t.Cleanup(server.Close)

	archive := writeTestArchive(t, map[string]string{
		officialArchiveRoot + "/LICENSE":             "license",
		officialArchiveRoot + "/pyproject.toml":      `version = "0.20.2"`,
		officialArchiveRoot + "/scripts/install.ps1": "crlf-copy\r\n",
		officialArchiveRoot + "/hermes_cli/main.py":  "pass\n",
	})
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	var spawned []string
	manifest, err := json.Marshal(reviewedManifest())
	if err != nil {
		t.Fatal(err)
	}
	installer := NewHostInstaller(t.TempDir())
	installer.source = testSourceClient(server.URL, script)
	installer.home = func() string { return home }
	installer.installDir = func() string { return installDir }
	installer.shell = func() (string, error) { return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil }
	installer.archive.diskFree = func(string) (uint64, error) { return archiveDiskBudget + archiveDiskMargin, nil }
	installer.acquireArchive = func(context.Context, string) (string, string, error) {
		return archive, sourceOriginBundled, nil
	}
	installer.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		joined := strings.Join(invocation.Args, " ")
		spawned = append(spawned, joined)
		if strings.Contains(joined, "ProtocolVersion") {
			return commandResult{stdout: "1\n"}
		}
		if strings.Contains(joined, "Manifest") {
			return commandResult{stdout: string(manifest) + "\n"}
		}
		stage := stageFromArgs(invocation.Args)
		return commandResult{stdout: `{"stage":"` + stage + `","ok":true,"skipped":false}` + "\n"}
	}

	var notes []string
	if err := installer.Apply(context.Background(), "op_embed", func(_ operation.Stage, note string) {
		if note != "" {
			notes = append(notes, note)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(installDir, "pyproject.toml")) {
		t.Fatal("source was not materialized")
	}
	for _, args := range spawned {
		if strings.Contains(args, "-Stage") && strings.Contains(args, "repository") {
			t.Fatalf("official repository stage spawned: %s", args)
		}
		for _, excluded := range excludedInstallStages() {
			if strings.Contains(args, excluded) {
				t.Fatalf("excluded stage spawned: %s", args)
			}
		}
	}
	if !containsString(notes, warningBundledUsed) || !containsString(notes, warningSourcePrepared) {
		t.Fatalf("notes = %#v", notes)
	}
}

func writeOfficialSizedStub(t *testing.T, payload []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bundled.zip")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func stageFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "-Stage" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return "unknown"
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
