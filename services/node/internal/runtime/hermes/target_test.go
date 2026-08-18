package hermes

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestCopyOwnedTreeStaysContained(t *testing.T) {
	staging := t.TempDir()
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "pyproject.toml"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyOwnedTree(context.Background(), staging, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestCopyOwnedTreeRejectsReparseDestination(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("reparse check is Windows-specific")
	}
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	realDest := filepath.Join(parent, "real")
	if err := os.Mkdir(realDest, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(realDest, link); err != nil {
		t.Skip(err)
	}
	if installErrorCode(copyOwnedTree(context.Background(), staging, link)) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed &&
		installErrorCode(copyOwnedTree(context.Background(), staging, link)) != yorvaruntime.ErrorRuntimeInstallStageFailed {
		err := copyOwnedTree(context.Background(), staging, link)
		if err == nil {
			t.Fatal("reparse destination accepted")
		}
	}
}

func TestValidateGenerationHomeRejectsEmpty(t *testing.T) {
	if err := validateGenerationHome(""); err == nil {
		t.Fatal("empty home accepted")
	}
	if err := validateGenerationHome(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
