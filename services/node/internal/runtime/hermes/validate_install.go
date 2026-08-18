package hermes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var pyprojectVersionPattern = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)

// ValidateStaging checks a built staging tree before Seal.
// config-templates output is not required (D3).
func ValidateStaging(stagingDir, expectedVersion string) error {
	if stagingDir == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("staging directory is required"))
	}
	if err := rejectReparsePoint(stagingDir); err != nil {
		return err
	}
	required := []string{
		"LICENSE",
		"pyproject.toml",
		filepath.Join("scripts", "install.ps1"),
		filepath.Join("hermes_cli", "main.py"),
		filepath.Join("venv", "Scripts", "hermes.exe"),
		filepath.Join("venv", "Scripts", "hermes-acp.exe"),
		filepath.Join("bin", "hermes.exe"),
		filepath.Join("bin", "hermes-acp.exe"),
		".hermes-bootstrap-complete",
	}
	for _, name := range required {
		path := filepath.Join(stagingDir, name)
		if !isRegularFile(path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("staging missing %s", name))
		}
		if err := rejectReparsePoint(path); err != nil {
			return err
		}
	}
	version, err := readPyprojectVersion(filepath.Join(stagingDir, "pyproject.toml"))
	if err != nil {
		return err
	}
	if expectedVersion != "" && version != expectedVersion {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("staging version %s != %s", version, expectedVersion))
	}
	parsed, err := parseVersionBanner("Hermes Agent v" + version)
	if err != nil || !parsed.supported() {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("unsupported staging version %s", version))
	}
	return nil
}

func readPyprojectVersion(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
	}
	match := pyprojectVersionPattern.FindSubmatch(payload)
	if len(match) != 2 {
		return "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("pyproject.toml has no version"))
	}
	version := strings.TrimSpace(string(match[1]))
	if version == "" {
		return "", installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("pyproject.toml version is empty"))
	}
	return version, nil
}
