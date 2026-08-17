package hermes

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestExtractOfficialArchiveAcceptsBoundedTree(t *testing.T) {
	archive := writeTestArchive(t, map[string]string{
		officialArchiveRoot + "/LICENSE":                "license",
		officialArchiveRoot + "/pyproject.toml":         "name = \"hermes-agent\"",
		officialArchiveRoot + "/scripts/install.ps1":    "Write-Host official",
		officialArchiveRoot + "/hermes_cli/main.py":     "def main():\n    return 0\n",
	})
	dest := filepath.Join(t.TempDir(), "tree")
	if err := extractOfficialArchive(context.Background(), archive, dest); err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(dest, "hermes_cli", "main.py")) {
		t.Fatal("expected stripped official tree")
	}
}

func TestExtractOfficialArchiveRejectsAdversarialMembers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		files map[string]string
		mode  os.FileMode
	}{
		{name: "zip-slip", files: map[string]string{officialArchiveRoot + "/../../evil.txt": "x"}},
		{name: "absolute", files: map[string]string{"/tmp/evil.txt": "x"}},
		{name: "ads", files: map[string]string{officialArchiveRoot + "/file:stream": "x"}},
		{name: "wrong-prefix", files: map[string]string{"other-root/LICENSE": "x"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			archive := writeTestArchive(t, test.files)
			err := extractOfficialArchive(context.Background(), archive, filepath.Join(t.TempDir(), "out"))
			if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestExtractOfficialArchiveRejectsSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "link.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: officialArchiveRoot + "/link"}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("../outside")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if installErrorCode(extractOfficialArchive(context.Background(), path, filepath.Join(t.TempDir(), "out"))) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
		t.Fatal("symlink member was accepted")
	}
}

func TestExtractOfficialArchiveRejectsExpansionBomb(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bomb.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: officialArchiveRoot + "/bomb.bin", Method: zip.Store}
	header.UncompressedSize64 = uint64(archiveMaxMember + 1)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if installErrorCode(extractOfficialArchive(context.Background(), path, filepath.Join(t.TempDir(), "out"))) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
		t.Fatal("expansion bomb was accepted")
	}
}

func TestRequireExtractBudget(t *testing.T) {
	err := requireExtractBudget(t.TempDir(), func(string) (uint64, error) { return 1024, nil })
	if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallInsufficientDisk {
		t.Fatalf("error = %v", err)
	}
	if err := requireExtractBudget(t.TempDir(), func(string) (uint64, error) { return archiveDiskBudget + archiveDiskMargin, nil }); err != nil {
		t.Fatal(err)
	}
}

func TestOfficialScriptFromArchiveNormalizesCRLF(t *testing.T) {
	lf := bytes.Repeat([]byte("line\n"), 8)
	archive := writeTestArchive(t, map[string]string{
		officialArchiveRoot + "/scripts/install.ps1": strings.ReplaceAll(string(lf), "\n", "\r\n"),
	})
	// The helper verifies the official pin. This tiny fixture must fail closed.
	if installErrorCode(officialScriptFromArchive(archive, filepath.Join(t.TempDir(), "install.ps1"))) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
		t.Fatal("non-official script was accepted")
	}
}

func TestApprovedArchiveRedirect(t *testing.T) {
	ok, err := http.NewRequest(http.MethodGet, officialArchiveURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !approvedArchiveRedirect(ok) {
		t.Fatal("official archive URL should be allowed")
	}
	bad, err := http.NewRequest(http.MethodGet, "https://example.com/NousResearch/hermes-agent/"+officialCommit+".zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if approvedArchiveRedirect(bad) {
		t.Fatal("unapproved host was allowed")
	}
}

func writeTestArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.zip")
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
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
