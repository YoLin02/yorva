package hermes

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxCandidates = 8

type candidateSet struct {
	paths                []string
	truncated            bool
	installationEvidence bool
}

type candidateFinder struct {
	pathValue         string
	executableName    string
	officialPaths     []string
	installationRoots []string
	caseInsensitive   bool
	requireExecBit    bool
	limit             int
}

func newCandidateFinder() candidateFinder {
	executableName := "hermes"
	officialPaths := make([]string, 0, 2)
	installationRoots := make([]string, 0, 1)
	if runtime.GOOS == "windows" {
		executableName = "hermes.exe"
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			installRoot := filepath.Join(localAppData, "hermes", "hermes-agent")
			installationRoots = append(installationRoots, installRoot)
			officialPaths = append(officialPaths, filepath.Join(
				installRoot,
				"bin",
				"hermes.exe",
			))
			officialPaths = append(officialPaths, filepath.Join(
				installRoot,
				"venv",
				"Scripts",
				"hermes.exe",
			))
		}
	} else {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			officialPaths = append(officialPaths, filepath.Join(home, ".local", "bin", "hermes"))
		}
		officialPaths = append(officialPaths, "/usr/local/bin/hermes")
	}
	return candidateFinder{
		pathValue:         os.Getenv("PATH"),
		executableName:    executableName,
		officialPaths:     officialPaths,
		installationRoots: installationRoots,
		caseInsensitive:   runtime.GOOS == "windows",
		requireExecBit:    runtime.GOOS != "windows",
		limit:             maxCandidates,
	}
}

func (f candidateFinder) find() candidateSet {
	ordered := make([]string, 0, f.limit)
	for _, directory := range filepath.SplitList(f.pathValue) {
		if directory == "" {
			continue
		}
		ordered = append(ordered, filepath.Join(directory, f.executableName))
	}
	ordered = append(ordered, f.officialPaths...)

	result := candidateSet{
		paths:                make([]string, 0, min(f.limit, len(ordered))),
		installationEvidence: f.hasInstallationEvidence(),
	}
	seen := make(map[string]struct{}, len(ordered))
	for _, path := range ordered {
		canonical, ok := f.canonicalExecutable(path)
		if !ok {
			continue
		}
		key := canonical
		if f.caseInsensitive {
			key = strings.ToLower(key)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if len(result.paths) == f.limit {
			result.truncated = true
			continue
		}
		result.paths = append(result.paths, canonical)
	}
	return result
}

func (f candidateFinder) hasInstallationEvidence() bool {
	for _, root := range f.installationRoots {
		if !isDirectory(root) || !isDirectory(filepath.Join(root, "venv")) {
			continue
		}
		markers := []string{
			filepath.Join(root, "hermes"),
			filepath.Join(root, "pyproject.toml"),
			filepath.Join(root, "venv", "Scripts", "hermes-agent.exe"),
		}
		for _, marker := range markers {
			if isRegularFile(marker) {
				return true
			}
		}
	}
	return false
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func (f candidateFinder) canonicalExecutable(path string) (string, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	if f.requireExecBit && info.Mode().Perm()&0o111 == 0 {
		return "", false
	}
	return filepath.Clean(canonical), true
}
