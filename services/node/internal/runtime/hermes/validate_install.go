package hermes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const finalGenerationProbeTimeout = 30 * time.Second

var pyprojectVersionPattern = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)

// ValidateGenerationFiles checks the final candidate tree before its functional
// probes and Seal. config-templates output is not required (D3).
func ValidateGenerationFiles(generationDir, expectedVersion string) error {
	if generationDir == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("generation directory is required"))
	}
	if err := rejectReparsePoint(generationDir); err != nil {
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
		path := filepath.Join(generationDir, name)
		if !isRegularFile(path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("generation missing %s", name))
		}
		if err := rejectReparsePoint(path); err != nil {
			return err
		}
	}
	version, err := readPyprojectVersion(filepath.Join(generationDir, "pyproject.toml"))
	if err != nil {
		return err
	}
	if expectedVersion != "" && version != expectedVersion {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("generation version %s != %s", version, expectedVersion))
	}
	parsed, err := parseVersionBanner("Hermes Agent v" + version)
	if err != nil || !parsed.supported() {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("unsupported generation version %s", version))
	}
	return nil
}

// ValidateGeneration raises the install gate from file existence to execution at
// the exact final path. Running both copied and venv launchers catches uv
// trampoline and editable-install paths that still reference a former directory.
func (h *HostInstaller) ValidateGeneration(ctx context.Context, generationDir, expectedVersion string) error {
	if err := ValidateGenerationFiles(generationDir, expectedVersion); err != nil {
		return err
	}
	canonicalRoot, ok := canonicalDirectory(generationDir)
	if !ok {
		return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("final generation path is not canonical"))
	}
	for _, rel := range []string{
		filepath.Join("bin", "hermes.exe"),
		filepath.Join("venv", "Scripts", "hermes.exe"),
	} {
		executable := filepath.Join(canonicalRoot, rel)
		canonical, ok := canonicalRegularWithin(canonicalRoot, executable)
		if !ok {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("final launcher is not contained"))
		}
		result := h.run(ctx, installInvocation{
			Executable: canonical,
			Args:       []string{"--version"},
			Dir:        canonicalRoot,
		}, finalGenerationProbeTimeout)
		h.logCommand("installer.final_path_probe", filepath.ToSlash(rel), "", result)
		if result.limited {
			return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
		}
		if result.timedOut {
			return installError(yorvaruntime.ErrorRuntimeInstallTimeout, result.err)
		}
		if result.err != nil || result.exitCode != 0 {
			if result.err == context.Canceled {
				return result.err
			}
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("final launcher failed"))
		}
		parsed, err := parseVersionBanner(result.stdout)
		if err != nil || !parsed.supported() || (expectedVersion != "" && parsed.String() != expectedVersion) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, fmt.Errorf("final launcher version invalid"))
		}
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
