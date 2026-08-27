package backupmanagement

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	// QualifiedSnapshotRuntimeVersion is the only Hermes source contract that
	// has been reviewed for this policy. A newer compatible Runtime can still be
	// detected, but it must be re-qualified before this snapshot policy is used.
	QualifiedSnapshotRuntimeVersion = "0.20.5"

	ErrorSourceUnsafe       ErrorCode = "BACKUP_SOURCE_UNSAFE"
	ErrorSourceChanged      ErrorCode = "BACKUP_SOURCE_CHANGED"
	ErrorSourceIncomplete   ErrorCode = "BACKUP_SOURCE_INCOMPLETE"
	ErrorSourceRuntimeLive  ErrorCode = "BACKUP_SOURCE_RUNTIME_NOT_STOPPED"
	ErrorSourceSQLiteActive ErrorCode = "BACKUP_SOURCE_SQLITE_ACTIVE"
)

// These exclusions mirror the exact 0.20.5 full-backup policy for
// regenerable/runtime-only material, plus YORVA's explicit exclusion of
// installation bytes. They are matched as path components. The canonical
// Hermes root itself remains the sole allowlisted source boundary.
var snapshotExcludedDirectoryNames = map[string]struct{}{
	"__pycache__":      {},
	".cache":           {},
	".git":             {},
	".mypy_cache":      {},
	".nox":             {},
	".pytest_cache":    {},
	".ruff_cache":      {},
	".tox":             {},
	".venv":            {},
	"backups":          {},
	"browser-profiles": {},
	"checkpoints":      {},
	"node_modules":     {},
	"site-packages":    {},
	"state-snapshots":  {},
	"venv":             {},
}

var snapshotRootExcludedDirectoryNames = map[string]struct{}{
	".hermes-runtime": {},
	"bin":             {},
	"control":         {},
	"generations":     {},
	"hermes-agent":    {},
	"lsp":             {},
	"logs":            {},
	"node":            {},
	"operations":      {},
	"transactions":    {},
	"yorva":           {},
}

var snapshotExcludedFileNames = map[string]struct{}{
	".backup.lock": {},
	"cron.pid":     {},
	"gateway.pid":  {},
}

var snapshotRootExcludedFileNames = map[string]struct{}{
	"yorva.db":      {},
	"yorva.sqlite":  {},
	"yorva.sqlite3": {},
}

var snapshotExcludedFileSuffixes = []string{".db-shm", ".pyc", ".pyo"}

type snapshotSourceEntry struct {
	relPath string
	absPath string
	info    fs.FileInfo
	isDir   bool
}

type snapshotDirectoryState struct {
	absPath  string
	children []snapshotChildState
}

type snapshotChildState struct {
	name    string
	mode    fs.FileMode
	size    int64
	modTime time.Time
}

type snapshotInventory struct {
	entries     []snapshotSourceEntry
	directories []snapshotDirectoryState
	fileCount   int
	totalBytes  int64
}

func canonicalHermesRuntimeRoot() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" || !filepath.IsAbs(localAppData) {
		return "", verificationError(ErrorSourceUnsafe)
	}
	root := filepath.Clean(filepath.Join(localAppData, "hermes"))
	if !filepath.IsAbs(root) || filepath.Base(root) != "hermes" {
		return "", verificationError(ErrorSourceUnsafe)
	}
	return root, nil
}

func inventorySnapshotSource(ctx context.Context, root string, limits Limits) (snapshotInventory, error) {
	if ctx == nil || !limits.valid() || root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return snapshotInventory{}, verificationError(ErrorInputInvalid)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || unsafeSnapshotInfo(rootInfo) || validateSnapshotSourcePathChain(root) != nil {
		return snapshotInventory{}, verificationError(ErrorSourceUnsafe)
	}

	inventory := snapshotInventory{
		entries: []snapshotSourceEntry{{relPath: "", absPath: root, info: rootInfo, isDir: true}},
	}
	seenFolded := map[string]string{strings.ToLower(PayloadArchiveRoot): PayloadArchiveRoot}
	if err := inventorySnapshotDirectory(ctx, root, "", limits, &inventory, seenFolded); err != nil {
		return snapshotInventory{}, err
	}
	if inventory.fileCount == 0 {
		return snapshotInventory{}, verificationError(ErrorSourceIncomplete)
	}
	return inventory, nil
}

func inventorySnapshotDirectory(ctx context.Context, root, relative string, limits Limits, inventory *snapshotInventory, seenFolded map[string]string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	directory := filepath.Join(root, filepath.FromSlash(relative))
	children, err := os.ReadDir(directory)
	if err != nil {
		return verificationError(ErrorSourceIncomplete)
	}
	state := snapshotDirectoryState{absPath: directory, children: make([]snapshotChildState, 0, len(children))}
	for _, child := range children {
		info, err := os.Lstat(filepath.Join(directory, child.Name()))
		if err != nil {
			return verificationError(ErrorSourceChanged)
		}
		state.children = append(state.children, childState(child.Name(), info))

		childRelative := child.Name()
		if relative != "" {
			childRelative = filepath.Join(relative, child.Name())
		}
		if excludedSnapshotSource(childRelative, info) {
			continue
		}
		if unsafeSnapshotInfo(info) {
			return verificationError(ErrorSourceUnsafe)
		}

		archiveName := PayloadArchiveRoot + "/" + filepath.ToSlash(childRelative)
		if info.IsDir() {
			archiveName += "/"
		}
		if !safeMemberName(archiveName) {
			return verificationError(ErrorSourceUnsafe)
		}
		folded := strings.ToLower(strings.TrimSuffix(archiveName, "/"))
		if previous, ok := seenFolded[folded]; ok && previous != strings.TrimSuffix(archiveName, "/") {
			return verificationError(ErrorArchiveCaseCollision)
		}
		seenFolded[folded] = strings.TrimSuffix(archiveName, "/")

		entry := snapshotSourceEntry{relPath: filepath.ToSlash(childRelative), absPath: filepath.Join(directory, child.Name()), info: info, isDir: info.IsDir()}
		inventory.entries = append(inventory.entries, entry)
		if len(inventory.entries) > limits.MaxPayloadMembers {
			return verificationError(ErrorArchiveMemberLimit)
		}
		if info.IsDir() {
			if err := inventorySnapshotDirectory(ctx, root, childRelative, limits, inventory, seenFolded); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limits.MaxMemberExpandedBytes || inventory.totalBytes > limits.MaxTotalExpandedBytes-info.Size() {
			return verificationError(ErrorArchiveMemberSizeLimit)
		}
		inventory.fileCount++
		inventory.totalBytes += info.Size()
	}
	inventory.directories = append(inventory.directories, state)
	return nil
}

func excludedSnapshotSource(relative string, info fs.FileInfo) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	foldedParts := make([]string, len(parts))
	for index, part := range parts {
		foldedParts[index] = strings.ToLower(part)
	}
	if len(parts) == 1 && info.IsDir() {
		if _, excluded := snapshotRootExcludedDirectoryNames[foldedParts[0]]; excluded {
			return true
		}
	}
	if len(parts) == 1 && !info.IsDir() {
		if _, excluded := snapshotRootExcludedFileNames[foldedParts[0]]; excluded {
			return true
		}
	}
	for _, part := range foldedParts {
		if _, excluded := snapshotExcludedDirectoryNames[part]; excluded && info.IsDir() {
			return true
		}
	}
	name := foldedParts[len(foldedParts)-1]
	if _, excluded := snapshotExcludedFileNames[name]; excluded && !info.IsDir() {
		return true
	}
	for _, suffix := range snapshotExcludedFileSuffixes {
		if !info.IsDir() && strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func validateSnapshotSourcePathChain(path string) error {
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || unsafeSnapshotInfo(info) {
			return verificationError(ErrorSourceUnsafe)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func childState(name string, info fs.FileInfo) snapshotChildState {
	return snapshotChildState{name: name, mode: info.Mode(), size: info.Size(), modTime: info.ModTime()}
}

func verifySnapshotDirectoriesUnchanged(ctx context.Context, directories []snapshotDirectoryState) error {
	for _, before := range directories {
		if err := contextError(ctx); err != nil {
			return err
		}
		children, err := os.ReadDir(before.absPath)
		if err != nil || len(children) != len(before.children) {
			return verificationError(ErrorSourceChanged)
		}
		after := make([]snapshotChildState, 0, len(children))
		for _, child := range children {
			info, err := os.Lstat(filepath.Join(before.absPath, child.Name()))
			if err != nil {
				return verificationError(ErrorSourceChanged)
			}
			after = append(after, childState(child.Name(), info))
		}
		if !slices.EqualFunc(before.children, after, func(left, right snapshotChildState) bool {
			return left.name == right.name && left.mode == right.mode && left.size == right.size && left.modTime.Equal(right.modTime)
		}) {
			return verificationError(ErrorSourceChanged)
		}
	}
	return nil
}

func sameSnapshotFileState(left, right fs.FileInfo) bool {
	return left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime()) && os.SameFile(left, right)
}
