// Package managedskills owns YORVA-managed Skill source bundles.
package managedskills

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	SkillFileName       = "SKILL.md"
	ManagedMarkerName   = ".yorva-managed.json"
	MaxBundleFiles      = 128
	MaxBundleDepth      = 8
	MaxBundleFileBytes  = 512 << 10
	MaxBundleTotalBytes = 2 << 20
)

var ErrBundleInvalid = errors.New("managed Skill bundle is invalid")

// Bundle is a validated immutable view of a Skill source directory.
type Bundle struct {
	Files         []string
	ContentSHA256 string
	TotalBytes    int64
}

// InspectSourceDirectory validates and hashes an on-disk source. The reserved
// YORVA ownership marker is never accepted as source content.
func InspectSourceDirectory(root string) (Bundle, error) {
	return inspectDirectory(root, false)
}

// InspectProjectedDirectory validates and hashes projected content while
// excluding the YORVA ownership marker from the content digest.
func InspectProjectedDirectory(root string) (Bundle, error) {
	return inspectDirectory(root, true)
}

// CopySourceDirectory validates source, copies it to a new destination, and
// revalidates the copy. Destination must not exist.
func CopySourceDirectory(ctx context.Context, source, destination string) (Bundle, error) {
	bundle, err := InspectSourceDirectory(source)
	if err != nil {
		return Bundle{}, err
	}
	if _, err := os.Lstat(destination); err == nil {
		return Bundle{}, fmt.Errorf("%w: destination already exists", ErrBundleInvalid)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Bundle{}, fmt.Errorf("%w: inspect destination: %v", ErrBundleInvalid, err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return Bundle{}, fmt.Errorf("%w: create destination: %v", ErrBundleInvalid, err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(destination)
		}
	}()

	for _, relative := range bundle.Files {
		if err := ctx.Err(); err != nil {
			return Bundle{}, err
		}
		from := filepath.Join(source, filepath.FromSlash(relative))
		to := filepath.Join(destination, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return Bundle{}, fmt.Errorf("%w: create copied directory: %v", ErrBundleInvalid, err)
		}
		if err := copyRegularFile(from, to); err != nil {
			return Bundle{}, err
		}
	}
	copied, err := InspectSourceDirectory(destination)
	if err != nil {
		return Bundle{}, err
	}
	if copied.ContentSHA256 != bundle.ContentSHA256 {
		return Bundle{}, fmt.Errorf("%w: source changed while copying", ErrBundleInvalid)
	}
	cleanup = false
	return copied, nil
}

func inspectDirectory(root string, excludeMarker bool) (Bundle, error) {
	absolute, err := filepath.Abs(root)
	if err != nil || filepath.Clean(root) != absolute {
		return Bundle{}, fmt.Errorf("%w: source path must be absolute and clean", ErrBundleInvalid)
	}
	rootInfo, err := os.Lstat(absolute)
	if err != nil || !rootInfo.IsDir() || isReparsePoint(rootInfo) {
		return Bundle{}, fmt.Errorf("%w: source root must be a regular directory", ErrBundleInvalid)
	}
	if alternate, streamErr := hasAlternateDataStream(absolute); streamErr != nil || alternate {
		return Bundle{}, fmt.Errorf("%w: source root has an alternate data stream", ErrBundleInvalid)
	}

	type entry struct {
		relative string
		size     int64
	}
	entries := make([]entry, 0, 8)
	caseFolded := make(map[string]string)
	var total int64
	err = filepath.WalkDir(absolute, func(path string, directoryEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == absolute {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
			return fmt.Errorf("symlink or reparse point at %q", path)
		}
		if alternate, streamErr := hasAlternateDataStream(path); streamErr != nil || alternate {
			return fmt.Errorf("alternate data stream at %q", path)
		}
		relative, err := filepath.Rel(absolute, path)
		if err != nil || !safeRelativePath(relative) {
			return fmt.Errorf("unsafe relative path %q", relative)
		}
		slashRelative := filepath.ToSlash(relative)
		if pathDepth(slashRelative) > MaxBundleDepth {
			return fmt.Errorf("path exceeds maximum depth: %q", slashRelative)
		}
		folded := strings.ToLower(slashRelative)
		if previous, exists := caseFolded[folded]; exists && previous != slashRelative {
			return fmt.Errorf("case-fold collision between %q and %q", previous, slashRelative)
		}
		caseFolded[folded] = slashRelative
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file at %q", slashRelative)
		}
		if slashRelative == ManagedMarkerName {
			if excludeMarker {
				return nil
			}
			return fmt.Errorf("reserved marker in source")
		}
		if info.Size() < 0 || info.Size() > MaxBundleFileBytes {
			return fmt.Errorf("file exceeds size limit: %q", slashRelative)
		}
		total += info.Size()
		if total > MaxBundleTotalBytes {
			return fmt.Errorf("bundle exceeds total size limit")
		}
		entries = append(entries, entry{relative: slashRelative, size: info.Size()})
		if len(entries) > MaxBundleFiles {
			return fmt.Errorf("bundle exceeds file count limit")
		}
		return nil
	})
	if err != nil {
		return Bundle{}, fmt.Errorf("%w: %v", ErrBundleInvalid, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].relative < entries[j].relative })
	if len(entries) == 0 || entries[0].relative != SkillFileName {
		found := false
		for _, item := range entries {
			if item.relative == SkillFileName {
				found = true
				break
			}
		}
		if !found {
			return Bundle{}, fmt.Errorf("%w: root SKILL.md is required", ErrBundleInvalid)
		}
	}

	hash := sha256.New()
	files := make([]string, 0, len(entries))
	for _, item := range entries {
		if err := hashFile(hash, absolute, item.relative, item.size); err != nil {
			return Bundle{}, fmt.Errorf("%w: %v", ErrBundleInvalid, err)
		}
		files = append(files, item.relative)
	}
	return Bundle{Files: files, ContentSHA256: hex.EncodeToString(hash.Sum(nil)), TotalBytes: total}, nil
}

func hashFile(hash io.Writer, root, relative string, expectedSize int64) error {
	if err := binary.Write(hash, binary.BigEndian, uint32(len(relative))); err != nil {
		return err
	}
	if _, err := io.WriteString(hash, relative); err != nil {
		return err
	}
	if err := binary.Write(hash, binary.BigEndian, uint64(expectedSize)); err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	written, copyErr := io.CopyN(hash, file, expectedSize)
	closeErr := file.Close()
	if copyErr != nil || written != expectedSize {
		return fmt.Errorf("read %q: %v", relative, copyErr)
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func copyRegularFile(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() || isReparsePoint(info) {
		return fmt.Errorf("%w: source file changed type", ErrBundleInvalid)
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("%w: open source: %v", ErrBundleInvalid, err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("%w: create destination file: %v", ErrBundleInvalid, err)
	}
	_, copyErr := io.Copy(output, io.LimitReader(input, MaxBundleFileBytes+1))
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("%w: copy file: %v", ErrBundleInvalid, copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("%w: sync file: %v", ErrBundleInvalid, syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%w: close file: %v", ErrBundleInvalid, closeErr)
	}
	return nil
}

func safeRelativePath(path string) bool {
	if path == "" || path == "." || filepath.IsAbs(path) || filepath.VolumeName(path) != "" {
		return false
	}
	clean := filepath.Clean(path)
	if clean != path || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false
	}
	for _, component := range strings.Split(filepath.ToSlash(clean), "/") {
		if component == "" || component == "." || component == ".." || strings.Contains(component, ":") || !utf8.ValidString(component) {
			return false
		}
	}
	return true
}

func pathDepth(path string) int { return strings.Count(path, "/") + 1 }
