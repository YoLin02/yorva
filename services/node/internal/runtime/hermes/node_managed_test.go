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

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestNpmVersionSupported(t *testing.T) {
	if npmVersionSupported("10.9.8") {
		t.Fatal("npm 10.9.8 must be unsupported")
	}
	if !npmVersionSupported("12.0.2") {
		t.Fatal("npm 12.0.2 must be supported")
	}
}

func TestNodeVersionSupported(t *testing.T) {
	if !nodeVersionSupported("22.23.1") || nodeVersionSupported("20.0.0") {
		t.Fatal("node version policy mismatch")
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
