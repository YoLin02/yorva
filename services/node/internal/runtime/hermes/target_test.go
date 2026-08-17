package hermes

import (
	"os"
	"path/filepath"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestValidateInstallTarget(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := validateInstallTarget(home, installDir, false); err != nil {
		t.Fatalf("absent target: %v", err)
	}
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "foreign.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, installDir, false)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("foreign occupied target was accepted")
	}
	if installErrorCode(validateInstallTarget(home, installDir, true)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("unrecognized retry target was accepted")
	}
	if err := writeYorvaPartialMarker(installDir, officialCommit); err != nil {
		t.Fatal(err)
	}
	if err := validateInstallTarget(home, installDir, true); err != nil {
		t.Fatalf("yorva partial retry: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, outside, true)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("path outside hermes home was accepted")
	}
}
