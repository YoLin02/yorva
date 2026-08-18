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

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestBuildStagingUsesStagingInstallDirAndOfficialHome(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Hermes staging build is Windows user-scope")
	}
	env := newStagingBuildEnv(t)
	if err := env.installer.BuildStaging(context.Background(), "op_stage", env.staging, env.home); err != nil {
		t.Fatal(err)
	}
	if env.sawPathStage {
		t.Fatal("official -Stage path must not run")
	}
	if !env.sawStagingInstallDir || !env.sawOfficialHome {
		t.Fatalf("install flags not used: staging=%v home=%v args=%q", env.sawStagingInstallDir, env.sawOfficialHome, env.spawned)
	}
	if isRegularFile(filepath.Join(env.home, "hermes-agent", "bin", "hermes.exe")) {
		t.Fatal("live hermes-agent was written")
	}
	if _, err := os.Stat(filepath.Join(env.staging, ".yorva-phase3-install")); err == nil {
		t.Fatal("ownership marker written into staging")
	}
	if !isRegularFile(filepath.Join(env.staging, "bin", "hermes.exe")) || !isRegularFile(filepath.Join(env.staging, "bin", "hermes-acp.exe")) {
		t.Fatal("required public launchers missing")
	}
	if err := ValidateStaging(env.staging, officialPackageVersion); err != nil {
		t.Fatal(err)
	}
}

func TestBuildStagingConfigTemplatesWarningDoesNotFail(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Hermes staging build is Windows user-scope")
	}
	env := newStagingBuildEnv(t)
	env.failStage = "config-templates"
	if err := env.installer.BuildStaging(context.Background(), "op_d3", env.staging, env.home); err != nil {
		t.Fatalf("D3 should warn only: %v", err)
	}
	if err := ValidateStaging(env.staging, officialPackageVersion); err != nil {
		t.Fatal(err)
	}
}

func TestBuildStagingCancelDoesNotPublish(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Hermes staging build is Windows user-scope")
	}
	env := newStagingBuildEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	env.installer.afterStage = func(stage, _ string) {
		if stage == "uv" {
			cancel()
		}
	}
	err := env.installer.BuildStaging(ctx, "op_cancel", env.staging, env.home)
	if err == nil {
		t.Fatal("expected cancel")
	}
	if _, err := os.Stat(filepath.Join(env.home, "generations")); !os.IsNotExist(err) {
		t.Fatal("cancel published generations")
	}
}

func TestBuildStagingTimeoutFailsRequiredStage(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Hermes staging build is Windows user-scope")
	}
	env := newStagingBuildEnv(t)
	env.timeoutStage = "python"
	err := env.installer.BuildStaging(context.Background(), "op_timeout", env.staging, env.home)
	if err == nil {
		t.Fatal("required timeout ignored")
	}
	if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallTimeout {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateStagingRejectsMissingLauncherAndWrongVersion(t *testing.T) {
	root := t.TempDir()
	writeStagingLayout(t, root, officialPackageVersion)
	if err := os.Remove(filepath.Join(root, "bin", "hermes-acp.exe")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStaging(root, officialPackageVersion); err == nil {
		t.Fatal("missing launcher accepted")
	}
	writeStagingLayout(t, root, officialPackageVersion)
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(`version = "0.18.0"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStaging(root, officialPackageVersion); err == nil {
		t.Fatal("wrong version accepted")
	}
}

type stagingBuildEnv struct {
	t                    *testing.T
	home                 string
	staging              string
	installer            *HostInstaller
	failStage            string
	timeoutStage         string
	sawPathStage         bool
	sawStagingInstallDir bool
	sawOfficialHome      bool
	spawned              []string
}

func newStagingBuildEnv(t *testing.T) *stagingBuildEnv {
	t.Helper()
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
	staging := filepath.Join(home, "staging", "txn_test")
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(reviewedManifest())
	if err != nil {
		t.Fatal(err)
	}
	env := &stagingBuildEnv{t: t, home: home, staging: staging}
	installer := NewHostInstaller(t.TempDir())
	installer.source = testSourceClient(server.URL, script)
	installer.home = func() string { return home }
	installer.shell = func() (string, error) { return `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, nil }
	installer.archive.diskFree = func(string) (uint64, error) { return archiveDiskBudget + archiveDiskMargin, nil }
	installer.acquireArchive = func(context.Context, string) (string, string, error) {
		return archive, sourceOriginBundled, nil
	}
	installer.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		joined := strings.Join(invocation.Args, " ")
		env.spawned = append(env.spawned, joined)
		if strings.Contains(joined, "-InstallDir") && strings.Contains(joined, env.staging) {
			env.sawStagingInstallDir = true
		}
		if strings.Contains(joined, "-HermesHome") && strings.Contains(joined, env.home) {
			env.sawOfficialHome = true
		}
		if strings.Contains(joined, "ProtocolVersion") {
			return commandResult{stdout: "1\n"}
		}
		if strings.Contains(joined, "Manifest") {
			return commandResult{stdout: string(manifest) + "\n"}
		}
		stage := stageFromArgs(invocation.Args)
		stageDir := installDirFromArgs(invocation.Args)
		if stage == "path" {
			env.sawPathStage = true
		}
		if env.timeoutStage != "" && stage == env.timeoutStage {
			return commandResult{timedOut: true, err: context.DeadlineExceeded}
		}
		if env.failStage != "" && stage == env.failStage {
			return commandResult{stdout: `{"stage":"` + stage + `","ok":false,"skipped":false}` + "\n", exitCode: 1, err: errPlatform}
		}
		if stage == "venv" {
			for _, name := range []string{"hermes.exe", "hermes-acp.exe"} {
				path := filepath.Join(stageDir, "venv", "Scripts", name)
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("mz-"+name), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		if stage == "bootstrap-marker" {
			if err := os.WriteFile(filepath.Join(stageDir, ".hermes-bootstrap-complete"), []byte("ok"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return commandResult{stdout: `{"stage":"` + stage + `","ok":true,"skipped":false}` + "\n"}
	}
	env.installer = installer
	return env
}

func writeStagingLayout(t *testing.T, root, version string) {
	t.Helper()
	files := map[string]string{
		"LICENSE":                     "license",
		"pyproject.toml":              `version = "` + version + `"`,
		"scripts/install.ps1":         "script",
		"hermes_cli/main.py":          "pass",
		"venv/Scripts/hermes.exe":     "mz",
		"venv/Scripts/hermes-acp.exe": "mz",
		"bin/hermes.exe":              "mz",
		"bin/hermes-acp.exe":          "mz",
		".hermes-bootstrap-complete":  "ok",
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
