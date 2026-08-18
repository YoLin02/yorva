package hermes

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestNpmVersionSupported(t *testing.T) {
	if npmVersionSupported("10.9.8") || npmVersionSupported("12.0.0") || npmVersionSupported("12.1.0") {
		t.Fatal("npm versions other than 12.0.2 must be unsupported")
	}
	if !npmVersionSupported("12.0.2") || !npmVersionSupported("v12.0.2") {
		t.Fatal("npm 12.0.2 must be supported")
	}
}

func TestNodeVersionSupported(t *testing.T) {
	if !nodeVersionSupported("22.23.1") || !nodeVersionSupported("v22.23.1") {
		t.Fatal("node 22.23.1 must be supported")
	}
	if nodeVersionSupported("20.0.0") || nodeVersionSupported("22.22.0") || nodeVersionSupported("22.24.0") {
		t.Fatal("node versions other than 22.23.1 must be unsupported")
	}
}

func TestExtractPrefixedZipRejectsTraversal(t *testing.T) {
	archive := writeZip(t, map[string]string{officialNodeZipRoot + "/../evil.exe": "x"})
	err := extractPrefixedZip(context.Background(), archive, t.TempDir(), officialNodeZipRoot)
	if err == nil {
		t.Fatal("traversal accepted")
	}
}

func TestExtractNpmTarballRejectsSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "npm.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	header := &tar.Header{Name: "package/link", Typeflag: tar.TypeSymlink, Linkname: "../outside", Mode: 0777}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	if err := extractNpmTarball(context.Background(), path, t.TempDir()); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestVerifySizedDigestRejectsWrongHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.bin")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifySizedDigest(path, 3, officialNodeArchiveSHA); err == nil {
		t.Fatal("wrong hash accepted")
	}
}

func TestApplySucceedsWithoutHermesTree(t *testing.T) {
	root := t.TempDir()
	nodeDir := filepath.Join(root, "node")
	if err := os.MkdirAll(nodeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node.exe"), []byte("mz"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nodeDir, "node_modules", "npm", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedNpmCLI(nodeDir), []byte("cli"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewNodeHost(root, "", "")
	host.nodeDir = func() string { return nodeDir }
	host.installDir = func() string { return filepath.Join(root, "missing-hermes") }
	host.home = func() string { return root }
	host.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		if len(invocation.Args) >= 1 && invocation.Args[0] == "--version" {
			return commandResult{stdout: "v22.23.1\n"}
		}
		return commandResult{stdout: "12.0.2\n"}
	}
	if err := host.Apply(context.Background(), "op_node_only", nil); err != nil {
		t.Fatal(err)
	}
	got := host.Inspect()
	if got.NodeDependencies.State != PrereqNotInstalled {
		t.Fatalf("deps before Hermes = %#v", got.NodeDependencies)
	}
}

func TestInspectDepsRequiresStampMatchingLock(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "hermes-agent")
	nodeDir := filepath.Join(root, "node")
	if err := os.MkdirAll(filepath.Join(installDir, "node_modules"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nodeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "package-lock.json"), []byte(`{"lockfileVersion":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node.exe"), []byte("mz"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(nodeDir, "node_modules", "npm", "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedNpmCLI(nodeDir), []byte("cli"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewNodeHost(root, "", "")
	host.installDir = func() string { return installDir }
	host.nodeDir = func() string { return nodeDir }
	host.run = func(_ context.Context, invocation installInvocation, _ time.Duration) commandResult {
		if len(invocation.Args) >= 1 && invocation.Args[0] == "--version" {
			return commandResult{stdout: "v22.23.1\n"}
		}
		return commandResult{stdout: "12.0.2\n"}
	}
	got := host.Inspect()
	if got.NodeDependencies.State != PrereqFailed {
		t.Fatalf("partial modules without stamp = %#v", got.NodeDependencies)
	}
	digest, err := fileSHA256(filepath.Join(installDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, nodeDepsStampName), []byte(digest+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got = host.Inspect()
	if got.NodeDependencies.State != PrereqReady {
		t.Fatalf("stamped deps = %#v", got.NodeDependencies)
	}
}

func TestInspectReportsMissingManagedNode(t *testing.T) {
	host := NewNodeHost(t.TempDir(), "", "")
	host.nodeDir = func() string { return filepath.Join(t.TempDir(), "missing-node") }
	got := host.Inspect()
	if got.Node.State != PrereqMissing || got.Node.ErrorCode != yorvaruntime.ErrorHermesNodeMissing {
		t.Fatalf("%#v", got.Node)
	}
}

func TestStageInvocationNeverSpawnsOfficialNodeStages(t *testing.T) {
	for _, stage := range []string{"node", "node-deps"} {
		if _, err := stageInvocation("powershell", "install.ps1", stage, "C:\\h", "C:\\h\\a"); err == nil {
			t.Fatalf("%s was spawnable", stage)
		}
	}
}

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "n.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	_ = writer.Close()
	_ = file.Close()
	return path
}

func TestCuaURLNotInYorvaSpawnArgs(t *testing.T) {
	joined := strings.Join([]string{"-Stage", "uv"}, " ")
	if strings.Contains(joined, "trycua") || bytes.Contains([]byte(joined), []byte("Invoke-Expression")) {
		t.Fatal("unexpected CUA content")
	}
}
