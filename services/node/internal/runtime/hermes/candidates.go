package hermes

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	maxCandidates            = 8
	entryPointDirectoryLimit = 1024
	entryPointMetadataLimit  = 16 * 1024
	hermesCLIModule          = "hermes_cli.main"
	hermesCLIEntryPoint      = hermesCLIModule + ":main"
)

type commandInvocation struct {
	path       string
	executable string
	args       []string
	workingDir string
}

func directInvocation(path string) commandInvocation {
	return commandInvocation{
		path:       path,
		executable: path,
		args:       []string{"--version"},
	}
}

type candidateSet struct {
	commands             []commandInvocation
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
	ordered := make([]commandInvocation, 0, f.limit)
	for _, directory := range filepath.SplitList(f.pathValue) {
		if directory == "" {
			continue
		}
		ordered = append(ordered, directInvocation(filepath.Join(directory, f.executableName)))
	}
	for _, path := range f.officialPaths {
		ordered = append(ordered, directInvocation(path))
	}
	for _, root := range f.installationRoots {
		if command, ok := f.officialPythonInvocation(root); ok {
			ordered = append(ordered, command)
		}
	}

	result := candidateSet{
		commands:             make([]commandInvocation, 0, min(f.limit, len(ordered))),
		installationEvidence: f.hasInstallationEvidence(),
	}
	seen := make(map[string]struct{}, len(ordered))
	for _, command := range ordered {
		if command.executable == command.path && len(command.args) == 1 {
			canonical, ok := f.canonicalExecutable(command.executable)
			if !ok {
				continue
			}
			command = directInvocation(canonical)
		}
		key := command.executable + "\x00" + strings.Join(command.args, "\x00")
		if f.caseInsensitive {
			key = strings.ToLower(key)
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if len(result.commands) == f.limit {
			result.truncated = true
			continue
		}
		result.commands = append(result.commands, command)
	}
	return result
}

func (f candidateFinder) officialPythonInvocation(root string) (commandInvocation, bool) {
	canonicalRoot, ok := canonicalDirectory(root)
	if !ok {
		return commandInvocation{}, false
	}

	for _, launcher := range []string{
		filepath.Join(canonicalRoot, "bin", "hermes.exe"),
		filepath.Join(canonicalRoot, "venv", "Scripts", "hermes.exe"),
	} {
		if _, ok := canonicalRegularWithin(canonicalRoot, launcher); ok {
			return commandInvocation{}, false
		}
	}

	python, ok := canonicalRegularWithin(canonicalRoot, filepath.Join(
		canonicalRoot, "venv", "Scripts", "python.exe",
	))
	if !ok {
		return commandInvocation{}, false
	}
	if _, ok := canonicalRegularWithin(canonicalRoot, filepath.Join(canonicalRoot, "hermes")); !ok {
		return commandInvocation{}, false
	}
	if _, ok := canonicalRegularWithin(canonicalRoot, filepath.Join(
		canonicalRoot, "hermes_cli", "main.py",
	)); !ok {
		return commandInvocation{}, false
	}

	sitePackages, ok := canonicalDirectoryWithin(canonicalRoot, filepath.Join(
		canonicalRoot, "venv", "Lib", "site-packages",
	))
	if !ok || !hasUniqueHermesConsoleEntryPoint(canonicalRoot, sitePackages) {
		return commandInvocation{}, false
	}

	return commandInvocation{
		path:       python,
		executable: python,
		args:       []string{"-I", "-m", hermesCLIModule, "--version"},
		workingDir: canonicalRoot,
	}, true
}

func hasUniqueHermesConsoleEntryPoint(root, sitePackages string) bool {
	directory, err := os.Open(sitePackages)
	if err != nil {
		return false
	}
	defer directory.Close()
	entries, err := directory.ReadDir(entryPointDirectoryLimit + 1)
	if (err != nil && !errors.Is(err, io.EOF)) || len(entries) > entryPointDirectoryLimit {
		return false
	}
	packages := 0
	valid := false
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if !entry.IsDir() || !strings.HasPrefix(name, "hermes_agent-") || !strings.HasSuffix(name, ".dist-info") {
			continue
		}
		packages++
		metadata, ok := canonicalRegularWithin(root, filepath.Join(sitePackages, entry.Name(), "entry_points.txt"))
		if !ok {
			continue
		}
		contents, ok := readRegularFileBounded(metadata, entryPointMetadataLimit)
		if ok && hasConsoleEntryPoint(contents, "hermes", hermesCLIEntryPoint) {
			valid = true
		}
	}
	return packages == 1 && valid
}

func hasConsoleEntryPoint(contents []byte, name, target string) bool {
	section := ""
	definitions := 0
	matched := false
	scanner := bufio.NewScanner(bytes.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		if section != "console_scripts" || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == name {
			definitions++
			matched = strings.TrimSpace(value) == target
		}
	}
	return scanner.Err() == nil && definitions == 1 && matched
}

func readRegularFileBounded(path string, limit int64) ([]byte, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, false
	}
	contents, err := io.ReadAll(io.LimitReader(file, limit+1))
	return contents, err == nil && int64(len(contents)) <= limit
}

func canonicalDirectory(path string) (string, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return filepath.Clean(canonical), true
}

func canonicalDirectoryWithin(root, path string) (string, bool) {
	canonical, ok := canonicalDirectory(path)
	return canonical, ok && pathWithin(root, canonical)
}

func canonicalRegularWithin(root, path string) (string, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", false
	}
	if !pathWithin(root, canonical) || !isRegularFile(canonical) {
		return "", false
	}
	return filepath.Clean(canonical), true
}

func pathWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
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
