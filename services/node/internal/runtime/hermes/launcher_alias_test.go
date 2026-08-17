package hermes

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestOfficialLauncherAliasesSelectBinWhenDigestAndVersionMatch(t *testing.T) {
	root, bin, venv := writeOfficialLauncherPair(t, []byte("same-launcher-bytes"))
	detector := aliasDetector(t, root, bin, venv, "Hermes Agent v0.20.2\n", "Hermes Agent v0.20.2\n")
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoverySupported || got.Selected == nil {
		t.Fatalf("state = %#v", got)
	}
	if !sameFile(t, got.Selected.Path, bin) {
		t.Fatalf("selected = %s, want bin %s", got.Selected.Path, bin)
	}
	if warningCode(got, "HERMES_LAUNCHER_ALIAS") == "" {
		t.Fatal("expected alias warning")
	}
}

func TestOfficialLauncherAliasesAmbiguousWhenDigestDiffers(t *testing.T) {
	root, bin, venv := writeOfficialLauncherPair(t, []byte("bin-bytes"))
	if err := os.WriteFile(venv, []byte("venv-different-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	detector := aliasDetector(t, root, bin, venv, "Hermes Agent v0.20.2\n", "Hermes Agent v0.20.2\n")
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoveryAmbiguous || got.Selected != nil {
		t.Fatalf("state = %#v", got)
	}
}

func TestOfficialLauncherAliasesAmbiguousWhenVersionDiffers(t *testing.T) {
	root, bin, venv := writeOfficialLauncherPair(t, []byte("same-launcher-bytes"))
	detector := aliasDetector(t, root, bin, venv, "Hermes Agent v0.20.2\n", "Hermes Agent v0.19.4\n")
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoveryAmbiguous {
		t.Fatalf("state = %s", got.State)
	}
}

func TestOfficialLauncherAliasesAmbiguousAcrossDifferentRoots(t *testing.T) {
	rootA, binA, _ := writeOfficialLauncherPair(t, []byte("same-launcher-bytes"))
	rootB := t.TempDir()
	other := filepath.Join(rootB, "bin", "hermes.exe")
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("same-launcher-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	detector := &Detector{
		finder: candidateFinder{
			officialPaths:     []string{binA, other},
			installationRoots: []string{rootA, rootB},
			limit:             maxCandidates,
		},
		run: func(_ context.Context, command commandInvocation) commandResult {
			return commandResult{stdout: "Hermes Agent v0.20.2\n", exitCode: 0}
		},
		now:            time.Now,
		overallTimeout: overallDiscoveryTimeout,
	}
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoveryAmbiguous {
		t.Fatalf("state = %s", got.State)
	}
}

func TestVenvOnlyOfficialLauncherRemainsSupported(t *testing.T) {
	root := t.TempDir()
	venv := filepath.Join(root, "venv", "Scripts", "hermes.exe")
	if err := os.MkdirAll(filepath.Dir(venv), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(venv, []byte("venv-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	detector := &Detector{
		finder: candidateFinder{
			officialPaths:     []string{venv},
			installationRoots: []string{root},
			limit:             maxCandidates,
		},
		run:            func(context.Context, commandInvocation) commandResult { return commandResult{stdout: "Hermes Agent v0.20.2\n", exitCode: 0} },
		now:            time.Now,
		overallTimeout: overallDiscoveryTimeout,
	}
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoverySupported || got.Selected == nil || !sameFile(t, got.Selected.Path, venv) {
		t.Fatalf("got = %#v", got)
	}
}

func TestPathAndOfficialLauncherDedupExecuteOnce(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path alias")
	}
	root, bin, _ := writeOfficialLauncherPair(t, []byte("same-launcher-bytes"))
	pathDir := filepath.Dir(bin)
	finder := candidateFinder{
		pathValue:         pathDir,
		executableName:    "hermes.exe",
		officialPaths:     []string{bin},
		installationRoots: []string{root},
		caseInsensitive:   true,
		limit:             maxCandidates,
	}
	got := finder.find()
	if len(got.commands) != 1 {
		t.Fatalf("commands = %#v", got.commands)
	}
}

func writeOfficialLauncherPair(t *testing.T, payload []byte) (root, bin, venv string) {
	t.Helper()
	root = t.TempDir()
	bin = filepath.Join(root, "bin", "hermes.exe")
	venv = filepath.Join(root, "venv", "Scripts", "hermes.exe")
	for _, path := range []string{bin, venv} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, payload, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, bin, venv
}

func aliasDetector(t *testing.T, root, bin, venv, binBanner, venvBanner string) *Detector {
	t.Helper()
	return &Detector{
		finder: candidateFinder{
			officialPaths:     []string{bin, venv},
			installationRoots: []string{root},
			limit:             maxCandidates,
		},
		run: func(_ context.Context, command commandInvocation) commandResult {
			if sameFile(t, command.executable, venv) {
				return commandResult{stdout: venvBanner, exitCode: 0}
			}
			return commandResult{stdout: binBanner, exitCode: 0}
		},
		now:            time.Now,
		overallTimeout: overallDiscoveryTimeout,
	}
}

func sameFile(t *testing.T, left, right string) bool {
	t.Helper()
	a, err := filepath.EvalSymlinks(left)
	if err != nil {
		a = filepath.Clean(left)
	}
	b, err := filepath.EvalSymlinks(right)
	if err != nil {
		b = filepath.Clean(right)
	}
	return strings.EqualFold(a, b)
}

func warningCode(result yorvaruntime.Discovery, code string) string {
	for _, warning := range result.Warnings {
		if warning.Code == code {
			return warning.Code
		}
	}
	return ""
}
