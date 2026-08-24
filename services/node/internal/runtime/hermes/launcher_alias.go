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
		if !sameCanonicalPath(first.Path, bin) {
			selected = second
		}
		selected.Path = bin
		return selected, true
	}
	return yorvaruntime.Candidate{}, false
}

func officialLauncherPair(root, first, second string) (bin, venv string, ok bool) {
	hermesHome := filepath.Dir(root)
	left, leftOK := canonicalRegularWithin(hermesHome, first)
	right, rightOK := canonicalRegularWithin(hermesHome, second)
	if !leftOK || !rightOK {
		return "", "", false
	}
	venv, venvOK := canonicalRegularWithin(root, filepath.Join(root, "venv", "Scripts", "hermes.exe"))
	if !venvOK {
		return "", "", false
	}
	for _, candidate := range []string{
		filepath.Join(root, "bin", "hermes.exe"),
		filepath.Join(hermesHome, "bin", "hermes.exe"),
	} {
		var binOK bool
		bin, binOK = canonicalRegularWithin(hermesHome, candidate)
		if !binOK || sameCanonicalPath(bin, venv) {
			continue
		}
		if (sameCanonicalPath(left, bin) && sameCanonicalPath(right, venv)) ||
			(sameCanonicalPath(right, bin) && sameCanonicalPath(left, venv)) {
			return bin, venv, true
		}
	}
	return "", "", false
}

func sameCanonicalPath(first, second string) bool {
	return strings.EqualFold(filepath.Clean(first), filepath.Clean(second))
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
