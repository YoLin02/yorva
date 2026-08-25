package skillsmanagement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/YoLin02/yorva/services/node/internal/managedskills"
)

const (
	ProjectionMarkerSchema = 1
	projectionWorkspaceDir = ".yorva-skills"
)

type ProjectionErrorCode string

const (
	ErrorProjectionInvalid   ProjectionErrorCode = "SKILL_PROJECTION_INVALID"
	ErrorDestinationConflict ProjectionErrorCode = "SKILL_DESTINATION_CONFLICT"
	ErrorProjectionModified  ProjectionErrorCode = "SKILL_PROJECTION_MODIFIED"
	ErrorProjectionOwnership ProjectionErrorCode = "SKILL_OWNERSHIP_CONFLICT"
	ErrorProjectionVersion   ProjectionErrorCode = "SKILL_PROJECTION_VERSION_UNQUALIFIED"
	ErrorProjectionIO        ProjectionErrorCode = "SKILL_PROJECTION_IO"
)

var (
	ErrProjection          = errors.New("Hermes managed Skill projection failed")
	closedSkillIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	closedSourceIDPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	closedVersionPattern   = regexp.MustCompile(`^[0-9][0-9A-Za-z.+-]{0,63}$`)
	closedDeploymentIDExpr = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

type ProjectionError struct {
	Code  ProjectionErrorCode
	Cause error
}

func (e *ProjectionError) Error() string {
	if e.Cause == nil {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Cause.Error()
}

func (e *ProjectionError) Unwrap() error { return ErrProjection }

func (e *ProjectionError) ErrorCode() ProjectionErrorCode { return e.Code }

type ProjectRequest struct {
	RuntimeVersion string
	ProfileID      string
	SkillID        string
	SourceID       string
	Version        string
	SourceDir      string
	ContentSHA256  string
}

type Projection struct {
	ProfileID      string
	SkillID        string
	SourceID       string
	Version        string
	ContentSHA256  string
	DestinationDir string
}

type projectionMarker struct {
	Schema        int    `json:"schema"`
	DeploymentID  string `json:"deploymentID"`
	SkillID       string `json:"skillID"`
	ContentSHA256 string `json:"contentSHA256"`
	SourceID      string `json:"sourceID"`
	Version       string `json:"version"`
}

// Projector copies immutable YORVA-managed sources into the exact Hermes
// 0.20.5 discovery directories. It never invokes native Hermes mutation.
type Projector struct {
	hermesHome   string
	managedRoot  string
	deploymentID string
	mu           sync.Mutex
}

func NewProjector(hermesHome, managedRoot, deploymentID string) (*Projector, error) {
	home, err := cleanAbsolute(hermesHome)
	if err != nil {
		return nil, projectionError(ErrorProjectionInvalid, err)
	}
	managed, err := cleanAbsolute(managedRoot)
	if err != nil {
		return nil, projectionError(ErrorProjectionInvalid, err)
	}
	if !closedDeploymentIDExpr.MatchString(deploymentID) {
		return nil, projectionError(ErrorProjectionInvalid, errors.New("invalid deployment ID"))
	}
	return &Projector{hermesHome: home, managedRoot: managed, deploymentID: deploymentID}, nil
}

// NewWindowsProjector derives the exact Hermes 0.20.5 home from LOCALAPPDATA.
func NewWindowsProjector(managedRoot, deploymentID string) (*Projector, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return nil, projectionError(ErrorProjectionInvalid, errors.New("LOCALAPPDATA is not set"))
	}
	return NewProjector(filepath.Join(localAppData, "hermes"), managedRoot, deploymentID)
}

func (p *Projector) Project(ctx context.Context, request ProjectRequest) (Projection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := validateProjectRequest(request); err != nil {
		return Projection{}, err
	}
	if err := requireExactProjectionVersion(request.RuntimeVersion); err != nil {
		return Projection{}, err
	}
	scope, err := NewProfileScope(request.ProfileID)
	if err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	source, err := cleanAbsolute(request.SourceDir)
	if err != nil || !isContained(p.managedRoot, source) || source == p.managedRoot {
		return Projection{}, projectionError(ErrorProjectionInvalid, errors.New("source is outside the YORVA managed root"))
	}
	if err := rejectReparseAncestors(source); err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	sourceBundle, err := managedskills.InspectSourceDirectory(source)
	if err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	if err := validateManagedSourceLocation(p.managedRoot, source, request.SkillID, sourceBundle.ContentSHA256); err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	if request.ContentSHA256 != "" && request.ContentSHA256 != sourceBundle.ContentSHA256 {
		return Projection{}, projectionError(ErrorProjectionInvalid, errors.New("requested content digest does not match managed source"))
	}

	profileRoot := p.profileRoot(scope)
	if err := requireRegularDirectory(profileRoot); err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	if err := rejectReparseAncestors(profileRoot); err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	skillsRoot := filepath.Join(profileRoot, "skills")
	if err := ensureSkillsRoot(skillsRoot); err != nil {
		return Projection{}, projectionError(ErrorProjectionIO, err)
	}
	if err := rejectReparseAncestors(skillsRoot); err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	destination, found, err := findDestination(skillsRoot, request.SkillID)
	if err != nil {
		return Projection{}, err
	}

	var previous projectionMarker
	if found {
		previous, err = p.inspectOwnedDestination(destination, request.SkillID)
		if err != nil {
			return Projection{}, err
		}
		if previous.SourceID != request.SourceID {
			return Projection{}, projectionError(ErrorProjectionOwnership, errors.New("source identity does not match the owned projection"))
		}
		if previous.ContentSHA256 == sourceBundle.ContentSHA256 && previous.Version == request.Version {
			return projectionFromMarker(request.ProfileID, destination, previous), nil
		}
	} else {
		destination = filepath.Join(skillsRoot, request.SkillID)
	}

	workspaceRoot := filepath.Join(profileRoot, projectionWorkspaceDir)
	if err := ensureWorkspaceRoot(workspaceRoot); err != nil {
		return Projection{}, projectionError(ErrorProjectionIO, err)
	}
	operationRoot, err := os.MkdirTemp(workspaceRoot, "project-")
	if err != nil {
		return Projection{}, projectionError(ErrorProjectionIO, err)
	}
	newPath := filepath.Join(operationRoot, "new")
	oldPath := filepath.Join(operationRoot, "old")
	defer cleanupOperation(workspaceRoot, operationRoot)
	staged, err := managedskills.CopySourceDirectory(ctx, source, newPath)
	if err != nil {
		return Projection{}, projectionError(ErrorProjectionInvalid, err)
	}
	if staged.ContentSHA256 != sourceBundle.ContentSHA256 {
		return Projection{}, projectionError(ErrorProjectionInvalid, errors.New("source changed while staging"))
	}
	marker := projectionMarker{
		Schema: ProjectionMarkerSchema, DeploymentID: p.deploymentID,
		SkillID: request.SkillID, ContentSHA256: staged.ContentSHA256,
		SourceID: request.SourceID, Version: request.Version,
	}
	if err := writeMarker(newPath, marker); err != nil {
		return Projection{}, projectionError(ErrorProjectionIO, err)
	}

	if found {
		if err := os.Rename(destination, oldPath); err != nil {
			return Projection{}, projectionError(ErrorProjectionIO, err)
		}
		if err := os.Rename(newPath, destination); err != nil {
			if restoreErr := os.Rename(oldPath, destination); restoreErr != nil {
				return Projection{}, projectionError(ErrorProjectionIO, fmt.Errorf("publish failed: %v; restore failed: %v", err, restoreErr))
			}
			return Projection{}, projectionError(ErrorProjectionIO, err)
		}
		installed, inspectErr := p.inspectOwnedDestination(destination, request.SkillID)
		if inspectErr != nil || installed.ContentSHA256 != marker.ContentSHA256 || installed.Version != marker.Version {
			failedPath := filepath.Join(operationRoot, "failed")
			_ = os.Rename(destination, failedPath)
			_ = os.Rename(oldPath, destination)
			if inspectErr != nil {
				return Projection{}, inspectErr
			}
			return Projection{}, projectionError(ErrorProjectionIO, errors.New("published projection did not verify"))
		}
		return projectionFromMarker(request.ProfileID, destination, installed), nil
	}

	if err := os.Rename(newPath, destination); err != nil {
		return Projection{}, projectionError(ErrorProjectionIO, err)
	}
	installed, inspectErr := p.inspectOwnedDestination(destination, request.SkillID)
	if inspectErr != nil {
		failedPath := filepath.Join(operationRoot, "failed")
		_ = os.Rename(destination, failedPath)
		return Projection{}, inspectErr
	}
	return projectionFromMarker(request.ProfileID, destination, installed), nil
}

func (p *Projector) Unproject(ctx context.Context, runtimeVersion, profileID, skillID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := requireExactProjectionVersion(runtimeVersion); err != nil {
		return err
	}
	if !closedSkillIDPattern.MatchString(skillID) {
		return projectionError(ErrorProjectionInvalid, errors.New("invalid closed Skill ID"))
	}
	scope, err := NewProfileScope(profileID)
	if err != nil {
		return projectionError(ErrorProjectionInvalid, err)
	}
	profileRoot := p.profileRoot(scope)
	if err := requireRegularDirectory(profileRoot); err != nil {
		return projectionError(ErrorProjectionInvalid, err)
	}
	skillsRoot := filepath.Join(profileRoot, "skills")
	if err := requireRegularDirectory(skillsRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return projectionError(ErrorProjectionInvalid, err)
	}
	destination, found, err := findDestination(skillsRoot, skillID)
	if err != nil || !found {
		return err
	}
	if _, err := p.inspectOwnedDestination(destination, skillID); err != nil {
		return err
	}
	workspaceRoot := filepath.Join(profileRoot, projectionWorkspaceDir)
	if err := ensureWorkspaceRoot(workspaceRoot); err != nil {
		return projectionError(ErrorProjectionIO, err)
	}
	operationRoot, err := os.MkdirTemp(workspaceRoot, "unproject-")
	if err != nil {
		return projectionError(ErrorProjectionIO, err)
	}
	oldPath := filepath.Join(operationRoot, "old")
	defer cleanupOperation(workspaceRoot, operationRoot)
	if err := os.Rename(destination, oldPath); err != nil {
		return projectionError(ErrorProjectionIO, err)
	}
	if _, err := os.Lstat(destination); !errors.Is(err, os.ErrNotExist) {
		_ = os.Rename(oldPath, destination)
		return projectionError(ErrorProjectionIO, errors.New("projection remained visible after unproject"))
	}
	return nil
}

func (p *Projector) Inspect(runtimeVersion, profileID, skillID string) (Projection, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.inspect(runtimeVersion, profileID, skillID)
}

func (p *Projector) inspect(runtimeVersion, profileID, skillID string) (Projection, bool, error) {
	if err := requireExactProjectionVersion(runtimeVersion); err != nil {
		return Projection{}, false, err
	}
	if !closedSkillIDPattern.MatchString(skillID) {
		return Projection{}, false, projectionError(ErrorProjectionInvalid, errors.New("invalid closed Skill ID"))
	}
	scope, err := NewProfileScope(profileID)
	if err != nil {
		return Projection{}, false, projectionError(ErrorProjectionInvalid, err)
	}
	skillsRoot := filepath.Join(p.profileRoot(scope), "skills")
	if _, err := os.Lstat(skillsRoot); errors.Is(err, os.ErrNotExist) {
		return Projection{}, false, nil
	}
	destination, found, err := findDestination(skillsRoot, skillID)
	if err != nil || !found {
		return Projection{}, false, err
	}
	marker, err := p.inspectOwnedDestination(destination, skillID)
	if err != nil {
		return Projection{}, false, err
	}
	return projectionFromMarker(profileID, destination, marker), true, nil
}

func (p *Projector) List(runtimeVersion, profileID string) ([]Projection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := requireExactProjectionVersion(runtimeVersion); err != nil {
		return nil, err
	}
	scope, err := NewProfileScope(profileID)
	if err != nil {
		return nil, projectionError(ErrorProjectionInvalid, err)
	}
	skillsRoot := filepath.Join(p.profileRoot(scope), "skills")
	entries, err := os.ReadDir(skillsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return []Projection{}, nil
	}
	if err != nil {
		return nil, projectionError(ErrorProjectionIO, err)
	}
	result := make([]Projection, 0)
	for _, entry := range entries {
		if entry.Name() == projectionWorkspaceDir || !closedSkillIDPattern.MatchString(entry.Name()) {
			continue
		}
		markerPath := filepath.Join(skillsRoot, entry.Name(), managedskills.ManagedMarkerName)
		if _, err := os.Lstat(markerPath); errors.Is(err, os.ErrNotExist) {
			continue
		}
		marker, err := p.inspectOwnedDestination(filepath.Join(skillsRoot, entry.Name()), entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, projectionFromMarker(profileID, filepath.Join(skillsRoot, entry.Name()), marker))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SkillID < result[j].SkillID })
	return result, nil
}

func (p *Projector) inspectOwnedDestination(destination, skillID string) (projectionMarker, error) {
	info, err := os.Lstat(destination)
	if err != nil {
		return projectionMarker{}, projectionError(ErrorProjectionIO, err)
	}
	if !info.IsDir() || isProjectionReparsePoint(info) {
		return projectionMarker{}, projectionError(ErrorDestinationConflict, errors.New("destination is external or unsafe"))
	}
	marker, err := readMarker(destination)
	if errors.Is(err, os.ErrNotExist) {
		return projectionMarker{}, projectionError(ErrorDestinationConflict, errors.New("destination is not YORVA-managed"))
	}
	if err != nil {
		return projectionMarker{}, projectionError(ErrorProjectionOwnership, err)
	}
	if marker.Schema != ProjectionMarkerSchema || (p.deploymentID != "" && marker.DeploymentID != p.deploymentID) || marker.SkillID != skillID {
		return projectionMarker{}, projectionError(ErrorProjectionOwnership, errors.New("marker ownership does not match"))
	}
	bundle, err := managedskills.InspectProjectedDirectory(destination)
	if err != nil || bundle.ContentSHA256 != marker.ContentSHA256 {
		return projectionMarker{}, projectionError(ErrorProjectionModified, errors.New("managed projection content was modified"))
	}
	return marker, nil
}

func (p *Projector) profileRoot(scope ProfileScope) string {
	if scope.ProfileID() == "default" {
		return p.hermesHome
	}
	return filepath.Join(p.hermesHome, "profiles", scope.ProfileID())
}

func validateProjectRequest(request ProjectRequest) error {
	if !closedSkillIDPattern.MatchString(request.SkillID) || !closedSourceIDPattern.MatchString(request.SourceID) || !closedVersionPattern.MatchString(request.Version) {
		return projectionError(ErrorProjectionInvalid, errors.New("invalid closed projection identifier"))
	}
	return nil
}

func requireExactProjectionVersion(version string) error {
	if version != QualifiedHermesVersion {
		return projectionError(ErrorProjectionVersion, fmt.Errorf("requires Hermes %s", QualifiedHermesVersion))
	}
	return nil
}

func projectionFromMarker(profileID, destination string, marker projectionMarker) Projection {
	return Projection{ProfileID: profileID, SkillID: marker.SkillID, SourceID: marker.SourceID, Version: marker.Version, ContentSHA256: marker.ContentSHA256, DestinationDir: destination}
}

func projectionError(code ProjectionErrorCode, cause error) error {
	return &ProjectionError{Code: code, Cause: cause}
}

func writeMarker(directory string, marker projectionMarker) error {
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(directory, managedskills.ManagedMarkerName), data, 0o600)
}

func readMarker(directory string) (projectionMarker, error) {
	path := filepath.Join(directory, managedskills.ManagedMarkerName)
	info, err := os.Lstat(path)
	if err != nil {
		return projectionMarker{}, err
	}
	if !info.Mode().IsRegular() || isProjectionReparsePoint(info) || info.Size() > 4096 {
		return projectionMarker{}, errors.New("marker is not a small regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return projectionMarker{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var marker projectionMarker
	if err := decoder.Decode(&marker); err != nil {
		return projectionMarker{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return projectionMarker{}, errors.New("marker has trailing JSON content")
	}
	if marker.DeploymentID == "" || marker.SkillID == "" || marker.ContentSHA256 == "" || marker.SourceID == "" || marker.Version == "" {
		return projectionMarker{}, errors.New("marker is incomplete")
	}
	return marker, nil
}

func findDestination(skillsRoot, skillID string) (string, bool, error) {
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return "", false, projectionError(ErrorProjectionIO, err)
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), skillID) {
			if entry.Name() != skillID {
				return "", false, projectionError(ErrorDestinationConflict, errors.New("case-fold destination collision"))
			}
			return filepath.Join(skillsRoot, entry.Name()), true, nil
		}
	}
	return filepath.Join(skillsRoot, skillID), false, nil
}

func cleanAbsolute(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil || filepath.Clean(path) != absolute {
		return "", errors.New("path must be absolute and clean")
	}
	return absolute, nil
}

func isContained(root, child string) bool {
	relative, err := filepath.Rel(root, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func validateManagedSourceLocation(root, source, skillID, digest string) error {
	relative, err := filepath.Rel(root, source)
	if err != nil || filepath.IsAbs(relative) {
		return errors.New("managed source path is invalid")
	}
	parts := strings.Split(filepath.Clean(relative), string(filepath.Separator))
	if len(parts) != 3 || !closedDeploymentIDExpr.MatchString(parts[0]) || parts[1] != skillID || parts[2] != digest {
		return errors.New("managed source must be an instance/skill/content-digest directory")
	}
	return nil
}

func requireRegularDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || isProjectionReparsePoint(info) {
		return errors.New("path is not a regular directory")
	}
	return nil
}

func ensureSkillsRoot(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.IsDir() || isProjectionReparsePoint(info) {
			return errors.New("skills root is not a regular directory")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Mkdir(path, 0o700)
}

func ensureWorkspaceRoot(path string) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.IsDir() || isProjectionReparsePoint(info) {
			return errors.New("projection workspace is not a regular directory")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Mkdir(path, 0o700)
}

func rejectReparseAncestors(path string) error {
	current := filepath.Clean(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if isProjectionReparsePoint(info) {
			return errors.New("path contains a symlink or reparse point")
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func cleanupOperation(workspaceRoot, operationRoot string) {
	if !isContained(workspaceRoot, operationRoot) || operationRoot == workspaceRoot {
		return
	}
	info, err := os.Lstat(operationRoot)
	if err == nil && info.IsDir() && !isProjectionReparsePoint(info) {
		_ = os.RemoveAll(operationRoot)
	}
}
