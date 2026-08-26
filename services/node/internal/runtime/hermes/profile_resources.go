package hermes

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/managementhealth"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement"
	"gopkg.in/yaml.v3"
)

const (
	profileResourceLimit      = 256
	profileSkillMetadataLimit = 16 * 1024
	profileSkillPreviewLimit  = 4 * 1024
)

var errProfileResourcesUnavailable = errors.New("Hermes Profile resources are unavailable")

// ProfileResourceReader projects the same Profile-owned Skills and MCP
// definitions used by Hermes' desktop/browser management surfaces. It is
// deliberately read-only and never returns MCP transport or credential data.
type ProfileResourceReader struct {
	root string
	now  func() time.Time
}

func NewProfileResourceReader() *ProfileResourceReader {
	return &ProfileResourceReader{root: officialHermesHome(), now: func() time.Time { return time.Now().UTC() }}
}

func newProfileResourceReaderAt(root string) *ProfileResourceReader {
	return &ProfileResourceReader{root: root, now: func() time.Time { return time.Now().UTC() }}
}

func (r *ProfileResourceReader) ListSkills(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.Skill, error) {
	profileRoot, err := r.resolveProfile(ctx, installation, nativeID)
	if err != nil {
		return nil, err
	}
	skillsRoot := filepath.Join(profileRoot, "skills")
	disabled, err := readDisabledSkills(filepath.Join(profileRoot, "config.yaml"), runtime.GOOS)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Lstat(skillsRoot); os.IsNotExist(statErr) {
		return []yorvaruntime.Skill{}, nil
	} else if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
		return nil, errProfileResourcesUnavailable
	}

	items := make([]yorvaruntime.Skill, 0)
	seen := make(map[string]struct{})
	err = filepath.WalkDir(skillsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errProfileResourcesUnavailable
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == skillsRoot {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			info, infoErr := entry.Info()
			if infoErr != nil || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "SKILL.md" {
			return nil
		}
		name, version, description, _, metadataErr := readSkillMetadata(path)
		if metadataErr != nil || name == "" {
			return nil
		}
		enabledState := yorvaruntime.SkillEnabled
		if _, isDisabled := disabled[name]; isDisabled {
			enabledState = yorvaruntime.SkillDisabled
		}
		item := yorvaruntime.Skill{
			ID: name, Version: version, Description: description, Ownership: yorvaruntime.SkillOwnershipExternal,
			ProjectionState:   yorvaruntime.SkillProjectionUnknown,
			InstallationState: yorvaruntime.SkillInstalled,
			EnabledState:      enabledState,
			ScanState:         yorvaruntime.SkillScanUnknown,
		}
		if item.Validate() != nil {
			return nil
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return nil
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
		if len(items) > profileResourceLimit {
			return errProfileResourcesUnavailable
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *ProfileResourceReader) InspectSkill(ctx context.Context, installation yorvaruntime.Installation, nativeID, skillID string) (yorvaruntime.Skill, error) {
	items, err := r.ListSkills(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	for _, item := range items {
		if item.ID == skillID {
			profileRoot, resolveErr := r.resolveProfile(ctx, installation, nativeID)
			if resolveErr != nil {
				return yorvaruntime.Skill{}, resolveErr
			}
			preview, previewErr := inspectProfileSkill(ctx, filepath.Join(profileRoot, "skills"), skillID)
			if previewErr != nil {
				return yorvaruntime.Skill{}, previewErr
			}
			item.Preview = preview
			if item.Validate() != nil {
				return yorvaruntime.Skill{}, errProfileResourcesUnavailable
			}
			return item, nil
		}
	}
	return yorvaruntime.Skill{}, errProfileResourcesUnavailable
}

func (r *ProfileResourceReader) ListMCPServers(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.MCPServer, error) {
	profileRoot, err := r.resolveProfile(ctx, installation, nativeID)
	if err != nil {
		return nil, err
	}
	config, err := readBoundedYAML(filepath.Join(profileRoot, "config.yaml"))
	if err != nil {
		return nil, errProfileResourcesUnavailable
	}
	servers := mappingValue(config, "mcp_servers")
	if servers == nil {
		return []yorvaruntime.MCPServer{}, nil
	}
	if servers.Kind != yaml.MappingNode || len(servers.Content)%2 != 0 || len(servers.Content)/2 > profileResourceLimit {
		return nil, errProfileResourcesUnavailable
	}
	observedAt := r.clock()
	items := make([]yorvaruntime.MCPServer, 0, len(servers.Content)/2)
	for index := 0; index < len(servers.Content); index += 2 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := servers.Content[index].Value
		item := yorvaruntime.MCPServer{ID: name, PresetID: name, State: yorvaruntime.MCPConfigured, ObservedAt: observedAt}
		if item.Validate() != nil {
			return nil, errProfileResourcesUnavailable
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *ProfileResourceReader) ListMCPPresets(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.MCPPreset, error) {
	if _, err := r.resolveProfile(ctx, installation, nativeID); err != nil {
		return nil, err
	}
	catalog := mcpmanagement.NewRegistry().Catalog()
	items := make([]yorvaruntime.MCPPreset, 0, len(catalog))
	for _, preset := range catalog {
		items = append(items, yorvaruntime.MCPPreset{
			ID: preset.ID, DisplayName: preset.DisplayName, Description: preset.Description,
			HomepageURL: preset.HomepageURL, DocumentationURL: preset.DocumentationURL,
			AllowedToolIDs:     append([]string(nil), preset.AllowedToolIDs...),
			CredentialRequired: preset.CredentialClass != mcpmanagement.CredentialClassNone,
		})
	}
	return items, nil
}

func (r *ProfileResourceReader) resolveProfile(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if r == nil || r.root == "" || !filepath.IsAbs(r.root) || installation.RuntimeKind != Kind || installation.Path == "" || !filepath.IsAbs(installation.Path) {
		return "", errProfileResourcesUnavailable
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || normalized != nativeID || officialValidateProfileName(nativeID) != nil {
		return "", errProfileResourcesUnavailable
	}
	relative, err := filepath.Rel(r.root, installation.Path)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errProfileResourcesUnavailable
	}
	profileRoot := r.root
	if nativeID != "default" {
		profileRoot = filepath.Join(r.root, "profiles", nativeID)
	}
	info, err := os.Lstat(profileRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
		return "", errProfileResourcesUnavailable
	}
	return profileRoot, nil
}

func (r *ProfileResourceReader) clock() time.Time {
	if r != nil && r.now != nil {
		return r.now().UTC()
	}
	return time.Now().UTC()
}

func readSkillMetadata(path string) (string, string, string, string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
		return "", "", "", "", errProfileResourcesUnavailable
	}
	file, err := os.Open(path)
	if err != nil {
		return "", "", "", "", errProfileResourcesUnavailable
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, profileSkillMetadataLimit))
	if err != nil || !utf8.Valid(data) {
		return "", "", "", "", errProfileResourcesUnavailable
	}
	name := filepath.Base(filepath.Dir(path))
	trimmed := bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !bytes.HasPrefix(trimmed, []byte("---\n")) && !bytes.HasPrefix(trimmed, []byte("---\r\n")) {
		return name, "", "", projectProfileSkillText(trimmed), nil
	}
	lines := bytes.Split(trimmed, []byte("\n"))
	end := -1
	for index := 1; index < len(lines); index++ {
		if string(bytes.TrimSpace(lines[index])) == "---" {
			end = index
			break
		}
	}
	if end < 0 {
		return name, "", "", projectProfileSkillText(trimmed), nil
	}
	frontmatter := bytes.Join(lines[1:end], []byte("\n"))
	var node yaml.Node
	if yaml.Unmarshal(frontmatter, &node) != nil || len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return name, "", "", projectProfileSkillText(trimmed), nil
	}
	count := 0
	if validateYAMLNode(node.Content[0], &count) != nil {
		return name, "", "", projectProfileSkillText(trimmed), nil
	}
	if value := scalarMappingValue(node.Content[0], "name"); value != "" {
		name = value
	}
	body := bytes.Join(lines[end+1:], []byte("\n"))
	description := projectProfileSkillText([]byte(scalarMappingValue(node.Content[0], "description")))
	return name, scalarMappingValue(node.Content[0], "version"), description, projectProfileSkillText(body), nil
}

func inspectProfileSkill(ctx context.Context, skillsRoot, skillID string) (string, error) {
	var preview string
	found := false
	err := filepath.WalkDir(skillsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errProfileResourcesUnavailable
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == skillsRoot {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			info, infoErr := entry.Info()
			if infoErr != nil || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "SKILL.md" {
			return nil
		}
		name, _, _, candidate, metadataErr := readSkillMetadata(path)
		if metadataErr == nil && name == skillID {
			preview = candidate
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil || !found {
		return "", errProfileResourcesUnavailable
	}
	return preview, nil
}

func projectProfileSkillText(data []byte) string {
	value := managementhealth.RedactCredentialValues(strings.ToValidUTF8(string(data), "�"))
	value = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, value)
	value = strings.TrimSpace(value)
	if len(value) <= profileSkillPreviewLimit {
		return value
	}
	limit := profileSkillPreviewLimit
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit]
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func scalarMappingValue(node *yaml.Node, key string) string {
	value := mappingValue(node, key)
	if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		return ""
	}
	return value.Value
}

func readDisabledSkills(configPath, platform string) (map[string]struct{}, error) {
	config, err := readBoundedYAML(configPath)
	if err != nil {
		return nil, errProfileResourcesUnavailable
	}
	disabled := make(map[string]struct{})
	skills := mappingValue(config, "skills")
	appendStringSet(disabled, mappingValue(skills, "disabled"))
	platformDisabled := mappingValue(skills, "platform_disabled")
	appendStringSet(disabled, mappingValue(platformDisabled, platform))
	return disabled, nil
}

func appendStringSet(target map[string]struct{}, node *yaml.Node) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode && item.Tag == "!!str" && item.Value != "" {
			target[item.Value] = struct{}{}
		}
	}
}
