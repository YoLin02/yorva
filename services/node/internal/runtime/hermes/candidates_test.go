package hermes

import (
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
	if !slices.Equal(got.paths, want) {
		t.Fatalf("find() paths = %#v, want %#v", got.paths, want)
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
	if len(got.paths) != 0 || got.truncated {
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
	if len(got.paths) != maxCandidates || !got.truncated {
		t.Fatalf("find() = %#v, want %d candidates and truncation", got, maxCandidates)
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
