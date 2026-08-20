package hermes

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestExactNodeZipMaterialization(t *testing.T) {
	archive := writeZip(t, map[string]string{
		officialNodeZipRoot + "/node.exe": "mz",
		officialNodeZipRoot + "/README":   "ok",
		"outside.txt":                     "skip",
	})
	dest := filepath.Join(t.TempDir(), "node")
	if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(dest, "node.exe")) || !isRegularFile(filepath.Join(dest, "README")) {
		t.Fatal("expected prefixed node files")
	}
	if _, err := os.Stat(filepath.Join(dest, "outside.txt")); err == nil {
		t.Fatal("unprefixed member was extracted")
	}
}

func TestExactNpmTarMaterialization(t *testing.T) {
	path := writeTarGZ(t, map[string]string{"package/bin/npm-cli.js": "cli", "package/LICENSE": "mit"})
	dest := filepath.Join(t.TempDir(), "npm")
	if err := extractNpmTarball(context.Background(), path, dest); err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(dest, "bin", "npm-cli.js")) {
		t.Fatal("npm cli missing")
	}
}

func TestExtractPrefixedZipRejectsLimitsAndCancel(t *testing.T) {
	t.Run("root prefix", func(t *testing.T) {
		archive := writeZip(t, map[string]string{"other-root/node.exe": "x"})
		dest := filepath.Join(t.TempDir(), "out")
		if err := extractPrefixedZip(context.Background(), archive, dest, officialNodeZipRoot); err == nil {
			t.Fatal("wrong prefix returned no error")
		}
		if _, err := os.Stat(filepath.Join(dest, "node.exe")); err == nil {
			t.Fatal("wrong prefix extracted")
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		archive := writeZip(t, map[string]string{officialNodeZipRoot + "/node.exe": "mz"})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := extractPrefixedZip(ctx, archive, t.TempDir(), officialNodeZipRoot); err == nil {
			t.Fatal("cancelled extraction succeeded")
		}
	})
}

func TestDependencyInstallArgvEnvironmentAndStamp(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "hermes-agent")
	nodeDir := filepath.Join(root, "node")
	if err := os.MkdirAll(filepath.Join(nodeDir, "node_modules", "npm", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node.exe"), []byte("mz"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedNpmCLI(nodeDir), []byte("cli"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, nodeDepsStampName), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	var saw installInvocation
	host := NewNodeHost(root, "", "")
	host.installDir = func() string { return installDir }
	host.nodeDir = func() string { return nodeDir }
	host.run = func(_ context.Context, invocation installInvocation, timeout time.Duration) commandResult {
		if timeout == nodeDepsTimeout {
			saw = invocation
			return commandResult{stdout: "ok"}
		}
		if len(invocation.Args) >= 1 && invocation.Args[0] == "--version" {
			return commandResult{stdout: "v22.23.1\n"}
		}
		return commandResult{stdout: "12.0.2\n"}
	}
	if err := host.installDependencies(context.Background()); err != nil {
		t.Fatal(err)
	}
	if saw.Executable != filepath.Join(nodeDir, "node.exe") || saw.Dir != installDir {
		t.Fatalf("invocation = %#v", saw)
	}
	wantArgs := []string{managedNpmCLI(nodeDir), "ci", "--workspaces=false", "--omit=dev", "--ignore-scripts", "--no-audit", "--no-fund", "--progress=false"}
	if strings.Join(saw.Args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("args = %#v", saw.Args)
	}
	if _, err := os.Stat(filepath.Join(installDir, nodeDepsStampName)); err != nil {
		t.Fatal("stamp missing after success")
	}

	host.run = func(context.Context, installInvocation, time.Duration) commandResult {
		return commandResult{exitCode: 1}
	}
	if err := host.installDependencies(context.Background()); installErrorCode(err) != yorvaruntime.ErrorHermesNodeDepsFailed {
		t.Fatalf("failure = %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, nodeDepsStampName)); err == nil {
		t.Fatal("stamp survived dependency failure")
	}

	host.run = func(context.Context, installInvocation, time.Duration) commandResult {
		return commandResult{timedOut: true}
	}
	if err := os.WriteFile(filepath.Join(installDir, nodeDepsStampName), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := host.installDependencies(context.Background()); installErrorCode(err) != yorvaruntime.ErrorHermesNodeDepsTimeout {
		t.Fatalf("timeout = %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, nodeDepsStampName)); err == nil {
		t.Fatal("stamp survived timeout")
	}

	host.run = func(context.Context, installInvocation, time.Duration) commandResult {
		return commandResult{err: context.Canceled}
	}
	if err := os.WriteFile(filepath.Join(installDir, nodeDepsStampName), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := host.installDependencies(context.Background()); !errorsIsCanceled(err) {
		t.Fatalf("cancel = %v", err)
	}
	if _, err := os.Stat(filepath.Join(installDir, nodeDepsStampName)); err == nil {
		t.Fatal("stamp survived cancellation")
	}
}

func TestInstallerEnvironmentPinsNpmRegistry(t *testing.T) {
	env := installerEnvironment(`C:\Users\a\AppData\Local\hermes`)
	found := false
	for _, entry := range env {
		if entry == "NPM_CONFIG_REGISTRY="+hermesNPMRegistry {
			found = true
		}
		if strings.Contains(strings.ToUpper(entry), "TOKEN") || strings.Contains(entry, "secret") {
			t.Fatalf("environment leaked %q", entry)
		}
	}
	if !found {
		t.Fatal("npm registry pin missing")
	}
}

func TestManagedDependencyTimeoutIsFifteenMinutes(t *testing.T) {
	if nodeDepsTimeout != 15*time.Minute {
		t.Fatalf("node deps timeout = %s", nodeDepsTimeout)
	}
}

func TestManagedPinRejectsCompatibleReplacement(t *testing.T) {
	root := t.TempDir()
	nodeDir := filepath.Join(root, "node")
	if err := os.MkdirAll(filepath.Join(nodeDir, "node_modules", "npm", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node.exe"), []byte("mz"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedNpmCLI(nodeDir), []byte("cli"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewNodeHost(root, "", "")
	host.nodeDir = func() string { return nodeDir }
	host.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		if len(invocation.Args) >= 1 && invocation.Args[0] == "--version" {
			return commandResult{stdout: "v22.24.0\n"}
		}
		return commandResult{stdout: "12.9.0\n"}
	}
	got := host.Inspect()
	if got.Node.State != PrereqUnsupported || got.Node.ErrorCode != yorvaruntime.ErrorHermesNodeUnsupported {
		t.Fatalf("higher node version = %#v", got.Node)
	}
}

func errorsIsCanceled(err error) bool {
	return err != nil && (err == context.Canceled || strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "cancelled"))
}

func writeTarGZ(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "npm.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		header := &tar.Header{Name: name, Mode: 0600, Size: int64(len(body))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	return path
}

func TestCompiledExtractionLimits(t *testing.T) {
	if nodeArchiveMaxEntries != 8000 || nodeArchiveMaxMember != 96<<20 || nodeArchiveMaxUncompressed != 256<<20 {
		t.Fatalf("node zip limits changed unexpectedly")
	}
	const officialNodeExecutableSize = 86_989_128
	if nodeArchiveMaxMember < officialNodeExecutableSize {
		t.Fatalf("node zip member limit %d rejects the pinned node.exe size %d", nodeArchiveMaxMember, officialNodeExecutableSize)
	}
	if npmArchiveMaxEntries != 8000 || npmArchiveMaxUncompressed != 64<<20 {
		t.Fatalf("npm tar limits changed unexpectedly")
	}
}
