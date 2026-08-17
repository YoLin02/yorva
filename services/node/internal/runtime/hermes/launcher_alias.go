package hermes

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const maxLauncherAliasBytes = 64 << 20

func officialLauncherAliasSelection(candidates []yorvaruntime.Candidate, runnable []int, officialRoots []string) (yorvaruntime.Candidate, bool) {
	if len(runnable) != 2 || len(officialRoots) == 0 {
		return yorvaruntime.Candidate{}, false
	}
	first := candidates[runnable[0]]
	second := candidates[runnable[1]]
	if first.Version == "" || first.Version != second.Version {
		return yorvaruntime.Candidate{}, false
	}
	if first.State != second.State {
		return yorvaruntime.Candidate{}, false
	}
	for _, root := range officialRoots {
		canonicalRoot, ok := canonicalDirectory(root)
		if !ok {
			continue
		}
		bin, venv, ok := officialLauncherPair(canonicalRoot, first.Path, second.Path)
		if !ok {
			continue
		}
		if !sameRegularDigest(bin, venv) {
			return yorvaruntime.Candidate{}, false
		}
		selected := first
		if officialLauncherRelative(canonicalRoot, first.Path) != filepath.Join("bin", "hermes.exe") {
			selected = second
		}
		selected.Path = bin
		return selected, true
	}
	return yorvaruntime.Candidate{}, false
}

func officialLauncherPair(root, first, second string) (bin, venv string, ok bool) {
	left, leftOK := canonicalRegularWithin(root, first)
	right, rightOK := canonicalRegularWithin(root, second)
	if !leftOK || !rightOK {
		return "", "", false
	}
	rels := map[string]string{
		officialLauncherRelative(root, left):  left,
		officialLauncherRelative(root, right): right,
	}
	bin, binOK := rels[filepath.Join("bin", "hermes.exe")]
	venv, venvOK := rels[filepath.Join("venv", "Scripts", "hermes.exe")]
	return bin, venv, binOK && venvOK && bin != venv
}

func officialLauncherRelative(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	return filepath.Clean(relative)
}

func sameRegularDigest(first, second string) bool {
	sumFirst, okFirst := regularFileSHA256(first)
	sumSecond, okSecond := regularFileSHA256(second)
	return okFirst && okSecond && strings.EqualFold(sumFirst, sumSecond)
}

func regularFileSHA256(path string) (string, bool) {
	if err := rejectReparsePoint(path); err != nil {
		return "", false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxLauncherAliasBytes {
		return "", false
	}
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, io.LimitReader(file, maxLauncherAliasBytes+1)); err != nil {
		return "", false
	}
	return hex.EncodeToString(sum.Sum(nil)), true
}
