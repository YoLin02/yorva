package hermes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

func TestBuildStagingMaterializesSourceWithoutOfficialRepositoryStage(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Hermes staging build is Windows user-scope")
	}
	env := newStagingBuildEnv(t)
	if err := env.installer.BuildStaging(context.Background(), "op_embed", env.staging, env.home); err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(env.staging, "pyproject.toml")) {
		t.Fatal("source was not materialized")
	}
	for _, args := range env.spawned {
		if strings.Contains(args, "-Stage") && strings.Contains(args, "repository") {
			t.Fatalf("official repository stage spawned: %s", args)
		}
		for _, excluded := range excludedInstallStages() {
			if strings.Contains(args, excluded) {
				t.Fatalf("excluded stage spawned: %s", args)
			}
		}
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

func installDirFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "-InstallDir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
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
