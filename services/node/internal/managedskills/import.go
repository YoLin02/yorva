package managedskills

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const LocalImportSourceID = "local-import"

var sourceRefPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func ValidateImportSourceRef(sourceRef string) error {
	if !sourceRefPattern.MatchString(sourceRef) {
		return fmt.Errorf("%w: invalid local import reference", ErrCatalogEntryInvalid)
	}
	return nil
}

// ImportToManaged consumes a source staged by the native Desktop picker. The
// caller supplies only an opaque reference and a closed Skill ID; a filesystem
// path never crosses the HTTP boundary.
func (s *Store) ImportToManaged(ctx context.Context, instanceID, skillID, sourceRef string) (Acquisition, error) {
	if s == nil || !instanceIDPattern.MatchString(instanceID) || !managedIDPattern.MatchString(skillID) || ValidateImportSourceRef(sourceRef) != nil {
		return Acquisition{}, fmt.Errorf("%w: invalid local import", ErrCatalogEntryInvalid)
	}
	importsRoot := filepath.Join(filepath.Dir(s.root), "skill-imports")
	importRoot := filepath.Join(importsRoot, sourceRef)
	if err := ensureContained(importsRoot, importRoot); err != nil {
		return Acquisition{}, ErrCatalogEntryInvalid
	}
	defer os.RemoveAll(importRoot)

	sourceRoot := filepath.Join(importRoot, "directory")
	if _, err := os.Lstat(sourceRoot); errors.Is(err, os.ErrNotExist) {
		extracted := filepath.Join(importRoot, "extracted")
		if err := extractSkillZIP(filepath.Join(importRoot, "source.zip"), extracted); err != nil {
			return Acquisition{}, err
		}
		sourceRoot = extracted
	} else if err != nil {
		return Acquisition{}, fmt.Errorf("%w: inspect staged source", ErrBundleInvalid)
	}

	parent, name := filepath.Dir(sourceRoot), filepath.Base(sourceRoot)
	metadataBytes, err := os.ReadFile(filepath.Join(sourceRoot, SkillFileName))
	if err != nil {
		return Acquisition{}, fmt.Errorf("%w: root SKILL.md is required", ErrCatalogEntryInvalid)
	}
	metadata, err := parseSkillFrontmatter(metadataBytes)
	if err != nil || metadata.Name != skillID {
		return Acquisition{}, fmt.Errorf("%w: Skill ID must match SKILL.md name", ErrCatalogEntryInvalid)
	}
	files, digest, err := validateCatalogSource(catalogSource{
		descriptor: CatalogEntry{SourceID: LocalImportSourceID, SkillID: skillID, Version: metadata.Version, Description: metadata.Description},
		filesystem: os.DirFS(parent), directory: name,
	})
	if err != nil {
		return Acquisition{}, err
	}
	return s.materializeImported(ctx, instanceID, skillID, metadata.Version, digest, files)
}

func (s *Store) materializeImported(ctx context.Context, instanceID, skillID, version, digest string, files []catalogFile) (Acquisition, error) {
	relative := filepath.Join(instanceID, skillID, digest)
	destination := filepath.Join(s.root, relative)
	if err := ensureContained(s.root, destination); err != nil {
		return Acquisition{}, err
	}
	result := Acquisition{SourceID: LocalImportSourceID, SkillID: skillID, Version: version, ContentSHA256: digest, RelativePath: relative, AbsolutePath: destination}
	if existing, err := InspectSourceDirectory(destination); err == nil && existing.ContentSHA256 == digest {
		return result, nil
	} else if _, statErr := os.Lstat(destination); statErr == nil {
		return Acquisition{}, ErrManagedPathConflict
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return Acquisition{}, err
	}
	stagingParent := filepath.Join(s.root, ".staging")
	if err := os.MkdirAll(stagingParent, 0o700); err != nil {
		return Acquisition{}, err
	}
	staging, err := os.MkdirTemp(stagingParent, "import-")
	if err != nil {
		return Acquisition{}, err
	}
	defer os.RemoveAll(staging)
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return Acquisition{}, err
		}
		to := filepath.Join(staging, filepath.FromSlash(file.relative))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return Acquisition{}, err
		}
		if err := os.WriteFile(to, file.content, 0o600); err != nil {
			return Acquisition{}, err
		}
	}
	verified, err := InspectSourceDirectory(staging)
	if err != nil || verified.ContentSHA256 != digest {
		return Acquisition{}, ErrBundleInvalid
	}
	if err := os.Rename(staging, destination); err != nil {
		return Acquisition{}, err
	}
	return result, nil
}

func extractSkillZIP(zipPath, destination string) error {
	info, err := os.Lstat(zipPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxBundleTotalBytes*2 {
		return fmt.Errorf("%w: invalid ZIP source", ErrBundleInvalid)
	}
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("%w: invalid ZIP source", ErrBundleInvalid)
	}
	defer reader.Close()
	if len(reader.File) == 0 || len(reader.File) > MaxBundleFiles+MaxBundleDepth {
		return fmt.Errorf("%w: ZIP entry limit", ErrBundleInvalid)
	}
	prefix, err := zipSkillPrefix(reader.File)
	if err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}
	var total int64
	for _, entry := range reader.File {
		name := strings.TrimPrefix(strings.ReplaceAll(entry.Name, "\\", "/"), prefix)
		name = strings.TrimSuffix(name, "/")
		if name == "" {
			continue
		}
		relative := filepath.FromSlash(name)
		if !safeRelativePath(relative) || pathDepth(name) > MaxBundleDepth || entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: unsafe ZIP entry", ErrBundleInvalid)
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if !entry.Mode().IsRegular() || entry.UncompressedSize64 > MaxBundleFileBytes {
			return fmt.Errorf("%w: invalid ZIP file", ErrBundleInvalid)
		}
		total += int64(entry.UncompressedSize64)
		if total > MaxBundleTotalBytes {
			return fmt.Errorf("%w: ZIP expanded size limit", ErrBundleInvalid)
		}
		target := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			input.Close()
			return err
		}
		written, copyErr := io.Copy(output, io.LimitReader(input, MaxBundleFileBytes+1))
		closeInput, closeOutput := input.Close(), output.Close()
		if copyErr != nil || closeInput != nil || closeOutput != nil || written != int64(entry.UncompressedSize64) {
			return fmt.Errorf("%w: ZIP extraction failed", ErrBundleInvalid)
		}
	}
	return nil
}

func zipSkillPrefix(entries []*zip.File) (string, error) {
	direct := false
	root := ""
	rootSkill := false
	for _, entry := range entries {
		name := strings.TrimPrefix(strings.ReplaceAll(entry.Name, "\\", "/"), "./")
		rawRelative := filepath.FromSlash(strings.TrimSuffix(name, "/"))
		if rawRelative == "" || !safeRelativePath(rawRelative) {
			return "", fmt.Errorf("%w: unsafe ZIP entry", ErrBundleInvalid)
		}
		if name == SkillFileName {
			direct = true
		}
		parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
		if len(parts) > 1 {
			if root == "" {
				root = parts[0]
			} else if root != parts[0] {
				root = "!multiple"
			}
			if len(parts) == 2 && parts[1] == SkillFileName {
				rootSkill = true
			}
		}
	}
	if direct {
		return "", nil
	}
	if root != "" && root != "!multiple" && rootSkill {
		return root + "/", nil
	}
	return "", fmt.Errorf("%w: ZIP must contain one Skill with a root SKILL.md", ErrBundleInvalid)
}
