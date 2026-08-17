package hermes

import (
	"errors"
	"os"
	"path/filepath"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var errPlatform = errors.New("Hermes installation is Windows user-scope only")

type installInvocation struct {
	Executable string
	Args       []string
	Dir        string
}

func probeInvocation(powershell, script string, probe string, home, installDir string) (installInvocation, error) {
	if probe != "ProtocolVersion" && probe != "Manifest" {
		return installInvocation{}, errors.New("unsupported installer probe")
	}
	args := baseInstallerArgs(script)
	args = append(args, "-"+probe)
	if probe == "Manifest" {
		args = append(args, fixedInstallArgs(home, installDir)...)
	}
	return installInvocation{Executable: powershell, Args: args, Dir: filepath.Dir(script)}, nil
}

func stageInvocation(powershell, script, stage, home, installDir string) (installInvocation, error) {
	if !isApprovedStage(stage) || isExcludedStage(stage) {
		return installInvocation{}, installError(yorvaruntime.ErrorRuntimeInstallManifestMismatch, errors.New("refusing unapproved stage"))
	}
	args := append(baseInstallerArgs(script), "-Stage", stage, "-NonInteractive", "-Json")
	args = append(args, fixedInstallArgs(home, installDir)...)
	return installInvocation{Executable: powershell, Args: args, Dir: filepath.Dir(script)}, nil
}

func baseInstallerArgs(script string) []string {
	return []string{
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", script,
	}
}

func fixedInstallArgs(home, installDir string) []string {
	return []string{
		"-Branch", "main",
		"-Commit", officialCommit,
		"-HermesHome", home,
		"-InstallDir", installDir,
	}
}

func officialHermesHome() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "hermes")
}

func officialInstallDir() string {
	return filepath.Join(officialHermesHome(), "hermes-agent")
}
