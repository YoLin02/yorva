package hermes

import (
	"os"
	"path/filepath"
	"runtime"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func trustedPowerShell() (string, error) {
	if runtime.GOOS != "windows" {
		return "", installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = os.Getenv("WINDIR")
	}
	if root == "" {
		return "", installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	candidate := filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	info, err := os.Lstat(candidate)
	if err != nil || !info.Mode().IsRegular() {
		return "", installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	return filepath.Clean(candidate), nil
}
