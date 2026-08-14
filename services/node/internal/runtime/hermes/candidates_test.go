package hermes

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestCandidateFinderPreservesPathOrderAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	firstDir := filepath.Join(root, "first")
	secondDir := filepath.Join(root, "second")
	first := writeCandidate(t, firstDir)
	second := writeCandidate(t, secondDir)

	finder := testCandidateFinder([]string{firstDir, secondDir}, []string{first})
	got := finder.find()
	want := []string{canonicalPath(t, first), canonicalPath(t, second)}
	if !slices.Equal(invocationPaths(got.commands), want) {
		t.Fatalf("find() paths = %#v, want %#v", invocationPaths(got.commands), want)
	}
	if got.truncated {
		t.Fatal("find() unexpectedly reported truncation")
	}
}

func TestCandidateFinderIgnoresMissingDirectoriesAndNonFiles(t *testing.T) {
	root := t.TempDir()
	directoryCandidate := filepath.Join(root, executableNameForTest())
	if err := os.Mkdir(directoryCandidate, 0o755); err != nil {
		t.Fatal(err)
	}

	finder := testCandidateFinder([]string{filepath.Join(root, "missing"), root}, nil)
	got := finder.find()
	if len(got.commands) != 0 || got.truncated {
		t.Fatalf("find() = %#v, want no candidates", got)
	}
}

func TestCandidateFinderEnforcesLimit(t *testing.T) {
	root := t.TempDir()
	directories := make([]string, 0, maxCandidates+2)
	for i := 0; i < maxCandidates+2; i++ {
		directory := filepath.Join(root, string(rune('a'+i)))
		writeCandidate(t, directory)
		directories = append(directories, directory)
	}

	finder := testCandidateFinder(directories, nil)
	got := finder.find()
	if len(got.commands) != maxCandidates || !got.truncated {
		t.Fatalf("find() = %#v, want %d candidates and truncation", got, maxCandidates)
	}
}

func TestCandidateFinderSeparatesInstallationEvidenceFromExecutableCandidates(t *testing.T) {
	root := t.TempDir()
	venvScripts := filepath.Join(root, "venv", "Scripts")
	if err := os.MkdirAll(venvScripts, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyAgent := filepath.Join(venvScripts, "hermes-agent.exe")
	if err := os.WriteFile(legacyAgent, []byte("must not execute"), 0o755); err != nil {
		t.Fatal(err)
	}

	finder := candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}
	got := finder.find()
	if !got.installationEvidence {
		t.Fatal("find() did not report trusted installation evidence")
	}
	if len(got.commands) != 0 {
		t.Fatalf("find() commands = %#v, want no executable candidate", got.commands)
	}
}

func TestCandidateFinderRecognizesOfficialWrapperEvidenceWithoutLauncher(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "venv"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hermes"), []byte("official wrapper marker"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want installation evidence without an executable candidate", got)
	}
}

func TestCandidateFinderBuildsTrustedOfficialPythonInvocation(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		caseInsensitive:   true,
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 1 {
		t.Fatalf("find() = %#v, want one trusted Python invocation", got)
	}
	command := got.commands[0]
	wantPython := canonicalPath(t, filepath.Join(root, "venv", "Scripts", "python.exe"))
	wantArgs := []string{"-I", "-m", hermesCLIModule, "--version"}
	if command.path != wantPython || command.executable != wantPython || !slices.Equal(command.args, wantArgs) {
		t.Fatalf("Python invocation = %#v, want executable/path %q args %#v", command, wantPython, wantArgs)
	}
	if command.workingDir != canonicalPath(t, root) {
		t.Fatalf("Python invocation workingDir = %q, want %q", command.workingDir, canonicalPath(t, root))
	}
}

func TestCandidateFinderRejectsUntrustedPythonEntrypointMetadata(t *testing.T) {
	root := writeOfficialPythonInstallation(t, "attacker.module:main")

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command for untrusted entry point", got)
	}
}

func TestCandidateFinderRejectsDuplicateHermesEntrypointDefinitions(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint+"\nhermes = attacker.module:main")

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command for duplicate entry points", got)
	}
}

func TestCandidateFinderRejectsMultipleHermesPackageMetadataDirectories(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	metadata := filepath.Join(root, "venv", "Lib", "site-packages", "hermes_agent-0.20.1.dist-info", "entry_points.txt")
	if err := os.MkdirAll(filepath.Dir(metadata), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, []byte("[console_scripts]\nhermes = "+hermesCLIEntryPoint+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command for ambiguous package metadata", got)
	}
}

func TestCandidateFinderRejectsAdditionalNonMatchingHermesPackageMetadata(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	metadata := filepath.Join(root, "venv", "Lib", "site-packages", "hermes_agent-stale.dist-info", "entry_points.txt")
	if err := os.MkdirAll(filepath.Dir(metadata), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, []byte("[console_scripts]\nhermes = attacker.module:main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command for multiple package metadata directories", got)
	}
}

func TestCandidateFinderRejectsOversizedEntrypointMetadata(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	metadata := filepath.Join(root, "venv", "Lib", "site-packages", "hermes_agent-0.20.0.dist-info", "entry_points.txt")
	if err := os.WriteFile(metadata, []byte(strings.Repeat("x", entryPointMetadataLimit+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command for oversized metadata", got)
	}
}

func TestCandidateFinderBoundsSitePackagesEnumeration(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	sitePackages := filepath.Join(root, "venv", "Lib", "site-packages")
	for index := 0; index < entryPointDirectoryLimit; index++ {
		path := filepath.Join(sitePackages, fmt.Sprintf("unrelated-%04d", index))
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		installationRoots: []string{root},
		limit:             maxCandidates,
	}).find()
	if !got.installationEvidence || len(got.commands) != 0 {
		t.Fatalf("find() = %#v, want evidence but no command after the directory bound", got)
	}
}

func TestCandidateFinderPrefersOfficialExecutableOverPythonFallback(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	launcher := filepath.Join(root, "venv", "Scripts", "hermes.exe")
	if err := os.WriteFile(launcher, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{
		executableName:    "hermes.exe",
		officialPaths:     []string{launcher},
		installationRoots: []string{root},
		caseInsensitive:   true,
		limit:             maxCandidates,
	}).find()
	if len(got.commands) != 1 || got.commands[0].path != canonicalPath(t, launcher) || !slices.Equal(got.commands[0].args, []string{"--version"}) {
		t.Fatalf("find() = %#v, want only the official executable", got)
	}
}

func TestCandidateFinderRejectsIncompleteEvidence(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hermes"), []byte("wrapper"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := (candidateFinder{installationRoots: []string{root}, limit: maxCandidates}).find()
	if got.installationEvidence {
		t.Fatal("find() trusted a root without the official venv structure")
	}
}

func testCandidateFinder(pathDirectories, officialPaths []string) candidateFinder {
	return candidateFinder{
		pathValue:       strings.Join(pathDirectories, string(os.PathListSeparator)),
		executableName:  executableNameForTest(),
		officialPaths:   officialPaths,
		caseInsensitive: runtime.GOOS == "windows",
		requireExecBit:  runtime.GOOS != "windows",
		limit:           maxCandidates,
	}
}

func invocationPaths(commands []commandInvocation) []string {
	paths := make([]string, len(commands))
	for index, command := range commands {
		paths[index] = command.path
	}
	return paths
}

func writeOfficialPythonInstallation(t *testing.T, entryPoint string) string {
	t.Helper()
	root := t.TempDir()
	paths := map[string]string{
		filepath.Join(root, "hermes"):                        "official wrapper marker",
		filepath.Join(root, "hermes_cli", "main.py"):         "# official module marker\n",
		filepath.Join(root, "venv", "Scripts", "python.exe"): "python fixture",
		filepath.Join(root, "venv", "Lib", "site-packages", "hermes_agent-0.20.0.dist-info", "entry_points.txt"): "[console_scripts]\nhermes = " + entryPoint + "\nhermes-agent = run_agent:main\n",
	}
	for path, contents := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func executableNameForTest() string {
	if runtime.GOOS == "windows" {
		return "hermes.exe"
	}
	return "hermes"
}

func writeCandidate(t *testing.T, directory string) string {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, executableNameForTest())
	if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(canonical)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(absolute)
}
