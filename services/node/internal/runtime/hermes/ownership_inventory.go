package hermes

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fileInventory map[string]string

type inventoryDelta struct {
	Added   []string
	Changed []string
	Removed []string
}

func walkInventory(root string) (fileInventory, string, error) {
	inv := fileInventory{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := rejectReparsePoint(path); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == yorvaPartialMarker || strings.HasPrefix(entry.Name(), ".yorva-atom-") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		inv[filepath.ToSlash(relative)] = sum
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return inv, digestInventory(inv), nil
}

func digestInventory(inv fileInventory) string {
	lines := make([]string, 0, len(inv))
	for path, sum := range inv {
		lines = append(lines, path+"\x00"+sum)
	}
	sort.Strings(lines)
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(digest[:])
}

func diffInventory(before, after fileInventory) inventoryDelta {
	var delta inventoryDelta
	for path, sum := range after {
		prev, ok := before[path]
		switch {
		case !ok:
			delta.Added = append(delta.Added, path)
		case prev != sum:
			delta.Changed = append(delta.Changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			delta.Removed = append(delta.Removed, path)
		}
	}
	sort.Strings(delta.Added)
	sort.Strings(delta.Changed)
	sort.Strings(delta.Removed)
	return delta
}

func (d inventoryDelta) empty() bool {
	return len(d.Added) == 0 && len(d.Changed) == 0 && len(d.Removed) == 0
}

var allowedVenvExecutables = map[string]struct{}{
	"venv/scripts/python.exe":     {},
	"venv/scripts/pythonw.exe":    {},
	"venv/scripts/python3.exe":    {},
	"venv/scripts/pip.exe":        {},
	"venv/scripts/pip3.exe":       {},
	"venv/scripts/uv.exe":         {},
	"venv/scripts/hermes.exe":     {},
	"venv/scripts/hermes-acp.exe": {},
	"venv/scripts/activate.bat":   {},
	"venv/scripts/activate.ps1":   {},
	"venv/scripts/deactivate.bat": {},
	"bin/hermes.exe":              {},
	"bin/hermes-acp.exe":          {},
}

func isSensitiveExecutable(rel string) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".exe", ".dll", ".cmd", ".bat", ".ps1", ".com", ".scr", ".msi":
		return true
	default:
		return false
	}
}

func stageAllowsPath(stage, rel string) bool {
	slash := strings.ToLower(filepath.ToSlash(rel))
	switch stage {
	case "venv":
		if slash == "venv.pending-backup" {
			return true
		}
		if !strings.HasPrefix(slash, "venv/") {
			return false
		}
		if isSensitiveExecutable(slash) {
			_, ok := allowedVenvExecutables[slash]
			return ok
		}
		return true
	case "dependencies":
		if strings.HasPrefix(slash, ".venv/") {
			if isSensitiveExecutable(slash) {
				return false
			}
			return true
		}
		if !strings.HasPrefix(slash, "venv/") {
			return false
		}
		if isSensitiveExecutable(slash) {
			_, ok := allowedVenvExecutables[slash]
			return ok
		}
		return true
	case "path":
		return slash == "bin/hermes.exe" || slash == "bin/hermes-acp.exe"
	case "bootstrap-marker":
		return slash == ".hermes-bootstrap-complete"
	case "config-templates", "uv", "python", "git", "system-packages":
		return false
	default:
		return false
	}
}

func acceptStageDelta(stage string, delta inventoryDelta) bool {
	for _, path := range append(append(append([]string{}, delta.Added...), delta.Changed...), delta.Removed...) {
		if !stageAllowsPath(stage, path) {
			return false
		}
	}
	return true
}
