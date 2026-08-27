package managedskills

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const builtInCatalogRoot = "catalog"

var (
	//go:embed catalog/**
	builtInCatalog embed.FS

	managedIDPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	instanceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

	ErrCatalogEntryInvalid = errors.New("managed Skill catalog entry is invalid")
	ErrCatalogNotFound     = errors.New("managed Skill catalog entry was not found")
	ErrManagedPathConflict = errors.New("managed Skill immutable path conflicts with existing content")
)

type CatalogEntry struct {
	SourceID    string
	SkillID     string
	Version     string
	Description string
}

type Acquisition struct {
	SourceID      string
	SkillID       string
	Version       string
	ContentSHA256 string
	RelativePath  string
	AbsolutePath  string
}

type catalogSource struct {
	descriptor CatalogEntry
	filesystem fs.FS
	directory  string
}

type Store struct {
	root    string
	sources []catalogSource
}

var builtInSources = []catalogSource{{
	descriptor: CatalogEntry{
		SourceID:    "yorva-demo",
		SkillID:     "yorva-managed-demo",
		Version:     "1.0.0",
		Description: "A reviewed prose-only demonstration Skill managed by YORVA.",
	},
	filesystem: builtInCatalog,
	directory:  path.Join(builtInCatalogRoot, "yorva-managed-demo"),
}}

func NewStore(dataDir string) (*Store, error) {
	absolute, err := filepath.Abs(dataDir)
	if err != nil || filepath.Clean(dataDir) != absolute {
		return nil, fmt.Errorf("%w: data directory must be absolute and clean", ErrCatalogEntryInvalid)
	}
	return newStore(filepath.Join(absolute, "skills"), builtInSources), nil
}

func newStore(root string, sources []catalogSource) *Store {
	copySources := append([]catalogSource(nil), sources...)
	return &Store{root: root, sources: copySources}
}

func (s *Store) Root() string { return s.root }

func ListCatalog() []CatalogEntry {
	return listCatalog(builtInSources)
}

func (s *Store) ListCatalog() []CatalogEntry {
	return listCatalog(s.sources)
}

func listCatalog(sources []catalogSource) []CatalogEntry {
	result := make([]CatalogEntry, 0, len(sources))
	for _, source := range sources {
		result = append(result, source.descriptor)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceID == result[j].SourceID {
			return result[i].SkillID < result[j].SkillID
		}
		return result[i].SourceID < result[j].SourceID
	})
	return result
}

func (s *Store) AcquireToManaged(ctx context.Context, instanceID, skillID, sourceID string) (Acquisition, error) {
	if !instanceIDPattern.MatchString(instanceID) || !managedIDPattern.MatchString(skillID) || !managedIDPattern.MatchString(sourceID) {
		return Acquisition{}, fmt.Errorf("%w: invalid closed identifier", ErrCatalogEntryInvalid)
	}
	source, err := s.lookup(skillID, sourceID)
	if err != nil {
		return Acquisition{}, err
	}
	files, digest, err := validateCatalogSource(source)
	if err != nil {
		return Acquisition{}, err
	}
	relative := filepath.Join(instanceID, skillID, digest)
	destination := filepath.Join(s.root, relative)
	if err := ensureContained(s.root, destination); err != nil {
		return Acquisition{}, err
	}
	result := Acquisition{
		SourceID:      sourceID,
		SkillID:       skillID,
		Version:       source.descriptor.Version,
		ContentSHA256: digest,
		RelativePath:  relative,
		AbsolutePath:  destination,
	}

	if info, statErr := os.Lstat(destination); statErr == nil {
		if !info.IsDir() || isReparsePoint(info) {
			return Acquisition{}, ErrManagedPathConflict
		}
		bundle, inspectErr := InspectSourceDirectory(destination)
		if inspectErr != nil || bundle.ContentSHA256 != digest {
			return Acquisition{}, ErrManagedPathConflict
		}
		return result, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return Acquisition{}, fmt.Errorf("inspect immutable destination: %w", statErr)
	}

	if err := ctx.Err(); err != nil {
		return Acquisition{}, err
	}
	if err := os.MkdirAll(filepath.Join(s.root, instanceID, skillID), 0o700); err != nil {
		return Acquisition{}, fmt.Errorf("create managed Skill parent: %w", err)
	}
	stagingParent := filepath.Join(s.root, ".staging")
	if err := os.MkdirAll(stagingParent, 0o700); err != nil {
		return Acquisition{}, fmt.Errorf("create managed Skill staging parent: %w", err)
	}
	staging, err := os.MkdirTemp(stagingParent, "acquire-")
	if err != nil {
		return Acquisition{}, fmt.Errorf("create managed Skill staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return Acquisition{}, err
		}
		to := filepath.Join(staging, filepath.FromSlash(file.relative))
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return Acquisition{}, fmt.Errorf("create managed Skill directory: %w", err)
		}
		if err := os.WriteFile(to, file.content, 0o600); err != nil {
			return Acquisition{}, fmt.Errorf("write managed Skill file: %w", err)
		}
	}
	materialized, err := InspectSourceDirectory(staging)
	if err != nil || materialized.ContentSHA256 != digest {
		return Acquisition{}, fmt.Errorf("%w: materialized catalog digest mismatch", ErrCatalogEntryInvalid)
	}
	if err := os.Rename(staging, destination); err != nil {
		if existing, inspectErr := InspectSourceDirectory(destination); inspectErr == nil && existing.ContentSHA256 == digest {
			return result, nil
		}
		return Acquisition{}, fmt.Errorf("publish immutable managed Skill: %w", err)
	}
	return result, nil
}

func (s *Store) lookup(skillID, sourceID string) (catalogSource, error) {
	var found *catalogSource
	for i := range s.sources {
		source := &s.sources[i]
		if source.descriptor.SkillID == skillID && source.descriptor.SourceID == sourceID {
			if found != nil {
				return catalogSource{}, fmt.Errorf("%w: duplicate catalog identity", ErrCatalogEntryInvalid)
			}
			found = source
		}
	}
	if found == nil {
		return catalogSource{}, ErrCatalogNotFound
	}
	return *found, nil
}

type catalogFile struct {
	relative string
	content  []byte
}

func validateCatalogSource(source catalogSource) ([]catalogFile, string, error) {
	descriptor := source.descriptor
	if !managedIDPattern.MatchString(descriptor.SourceID) || !managedIDPattern.MatchString(descriptor.SkillID) || descriptor.Version == "" {
		return nil, "", fmt.Errorf("%w: invalid descriptor", ErrCatalogEntryInvalid)
	}
	var files []catalogFile
	caseFolded := make(map[string]string)
	var total int64
	err := fs.WalkDir(source.filesystem, source.directory, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == source.directory {
			return nil
		}
		prefix := strings.TrimSuffix(source.directory, "/") + "/"
		relative := strings.TrimPrefix(name, prefix)
		if relative == name || !validCatalogRelative(relative) {
			return fmt.Errorf("unsafe catalog path %q", name)
		}
		folded := strings.ToLower(relative)
		if previous, exists := caseFolded[folded]; exists && previous != relative {
			return fmt.Errorf("case-fold collision between %q and %q", previous, relative)
		}
		caseFolded[folded] = relative
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular catalog file %q", relative)
		}
		if relative != SkillFileName && !(strings.HasPrefix(relative, "references/") && strings.HasSuffix(strings.ToLower(relative), ".md")) {
			return fmt.Errorf("prose-only catalog rejects %q", relative)
		}
		content, err := fs.ReadFile(source.filesystem, name)
		if err != nil {
			return err
		}
		if len(content) > MaxBundleFileBytes || !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0 {
			return fmt.Errorf("invalid UTF-8 prose file %q", relative)
		}
		total += int64(len(content))
		if total > MaxBundleTotalBytes || len(files)+1 > MaxBundleFiles {
			return fmt.Errorf("catalog source exceeds limits")
		}
		files = append(files, catalogFile{relative: relative, content: content})
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrCatalogEntryInvalid, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relative < files[j].relative })
	if len(files) == 0 || files[0].relative != SkillFileName {
		return nil, "", fmt.Errorf("%w: root SKILL.md is required", ErrCatalogEntryInvalid)
	}
	if err := validateSkillFrontmatter(files[0].content, descriptor); err != nil {
		return nil, "", err
	}
	hash := sha256.New()
	for _, file := range files {
		_ = binary.Write(hash, binary.BigEndian, uint32(len(file.relative)))
		_, _ = hash.Write([]byte(file.relative))
		_ = binary.Write(hash, binary.BigEndian, uint64(len(file.content)))
		_, _ = hash.Write(file.content)
	}
	return files, hex.EncodeToString(hash.Sum(nil)), nil
}

type skillFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Author      string   `yaml:"author"`
	License     string   `yaml:"license"`
	Platforms   []string `yaml:"platforms"`
}

func validateSkillFrontmatter(content []byte, descriptor CatalogEntry) error {
	metadata, err := parseSkillFrontmatter(content)
	if err != nil {
		return err
	}
	if metadata.Name != descriptor.SkillID || metadata.Version != descriptor.Version || strings.TrimSpace(metadata.Description) == "" || strings.TrimSpace(metadata.Author) == "" || strings.TrimSpace(metadata.License) == "" {
		return fmt.Errorf("%w: frontmatter does not match reviewed descriptor", ErrCatalogEntryInvalid)
	}
	wanted := map[string]bool{"windows": false, "linux": false, "macos": false}
	for _, platform := range metadata.Platforms {
		if _, ok := wanted[platform]; !ok || wanted[platform] {
			return fmt.Errorf("%w: platforms must be the closed cross-platform set", ErrCatalogEntryInvalid)
		}
		wanted[platform] = true
	}
	for _, present := range wanted {
		if !present {
			return fmt.Errorf("%w: platforms must include Windows, Linux, and macOS", ErrCatalogEntryInvalid)
		}
	}
	return nil
}

func parseSkillFrontmatter(content []byte) (skillFrontmatter, error) {
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return skillFrontmatter{}, fmt.Errorf("%w: SKILL.md frontmatter is required", ErrCatalogEntryInvalid)
	}
	end := strings.Index(normalized[4:], "\n---\n")
	if end < 0 {
		return skillFrontmatter{}, fmt.Errorf("%w: SKILL.md frontmatter is unterminated", ErrCatalogEntryInvalid)
	}
	frontmatter := normalized[4 : 4+end]
	decoder := yaml.NewDecoder(strings.NewReader(frontmatter))
	decoder.KnownFields(true)
	var metadata skillFrontmatter
	if err := decoder.Decode(&metadata); err != nil {
		return skillFrontmatter{}, fmt.Errorf("%w: strict frontmatter: %v", ErrCatalogEntryInvalid, err)
	}
	return metadata, nil
}

func validCatalogRelative(relative string) bool {
	if relative == "" || relative == "." || path.IsAbs(relative) || path.Clean(relative) != relative || strings.Contains(relative, "\\") || strings.Contains(relative, ":") || strings.HasPrefix(relative, "../") || pathDepth(relative) > MaxBundleDepth {
		return false
	}
	for _, component := range strings.Split(relative, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func ensureContained(root, child string) error {
	relative, err := filepath.Rel(root, child)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("%w: path escaped managed root", ErrCatalogEntryInvalid)
	}
	return nil
}
