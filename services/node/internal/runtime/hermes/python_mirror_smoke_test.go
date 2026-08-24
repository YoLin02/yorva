package hermes

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

func TestRealBundledPythonMirrorSmoke(t *testing.T) {
	if os.Getenv("YORVA_REAL_PYTHON_MIRROR_SMOKE") != "1" {
		t.Skip("set YORVA_REAL_PYTHON_MIRROR_SMOKE=1 for the local bundled Python smoke")
	}
	if runtime.GOOS != "windows" {
		t.Skip("bundled Python artifact is Windows x64")
	}
	uv := filepath.Join(os.Getenv("LOCALAPPDATA"), "hermes", "bin", "uv.exe")
	if !isRegularFile(uv) {
		t.Skip("managed uv is unavailable")
	}
	_, here, _, _ := runtime.Caller(0)
	resource := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "..", "apps", "desktop", "src-tauri", "resources", "hermes", "source", officialPythonArchiveName))
	if !isRegularFile(resource) {
		t.Skip("bundled Python build input is unavailable")
	}
	workDir := t.TempDir()
	installer := NewHostInstaller(t.TempDir()).WithEmbeddedPython(resource)
	mirrorURL, metadataURL, err := installer.preparePythonMirror(context.Background(), workDir, downloadsources.Default())
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(t.TempDir(), "python")
	cacheDir := filepath.Join(t.TempDir(), "cache")
	cmd := exec.Command(uv, "python", "install", "3.11", "--reinstall")
	cmd.Env = append(os.Environ(),
		"UV_PYTHON_INSTALL_DIR="+installDir,
		"UV_CACHE_DIR="+cacheDir,
		"UV_PYTHON_INSTALL_MIRROR="+mirrorURL,
		"UV_PYTHON_DOWNLOADS_JSON_URL="+metadataURL,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("uv local Python install failed: %v: %s", err, output)
	}
	python := filepath.Join(installDir, "cpython-3.11.15-windows-x86_64-none", "python.exe")
	result, err := exec.Command(python, "--version").CombinedOutput()
	if err != nil || string(result) != "Python 3.11.15\r\n" {
		t.Fatalf("bundled Python is not runnable: %v: %q", err, result)
	}
}
