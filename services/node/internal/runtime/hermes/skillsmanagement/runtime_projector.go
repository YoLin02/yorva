package skillsmanagement

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// RuntimeProjector adapts the exact-version copy projector to the shared
// Runtime contract. It is intentionally stateless; deployment ownership comes
// from each closed mutation request, while inventory remains read-only.
type RuntimeProjector struct{}

func NewRuntimeProjector() *RuntimeProjector { return &RuntimeProjector{} }

func (r *RuntimeProjector) ListSkillProjections(ctx context.Context, installation yorvaruntime.Installation, profileID string) ([]yorvaruntime.Skill, error) {
	projector, err := newReadProjector()
	if err != nil {
		return nil, err
	}
	projections, err := projector.List(installation.Version, profileID)
	if err != nil {
		return nil, err
	}
	result := make([]yorvaruntime.Skill, 0, len(projections))
	for _, projection := range projections {
		result = append(result, runtimeSkill(projection, yorvaruntime.SkillProjectionProjected))
	}
	return result, nil
}

func (r *RuntimeProjector) InspectSkillProjection(ctx context.Context, installation yorvaruntime.Installation, profileID, skillID string) (yorvaruntime.Skill, error) {
	projector, err := newReadProjector()
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	projection, found, err := projector.Inspect(installation.Version, profileID, skillID)
	if err != nil {
		var projectionErr *ProjectionError
		if errors.As(err, &projectionErr) {
			switch projectionErr.Code {
			case ErrorDestinationConflict, ErrorProjectionOwnership:
				return runtimeProjectionState(skillID, yorvaruntime.SkillOwnershipExternal, yorvaruntime.SkillProjectionConflict), nil
			case ErrorProjectionModified:
				return runtimeProjectionState(skillID, yorvaruntime.SkillOwnershipYORVAManaged, yorvaruntime.SkillProjectionDriftModified), nil
			}
		}
		return yorvaruntime.Skill{}, err
	}
	if !found {
		return runtimeProjectionState(skillID, yorvaruntime.SkillOwnershipUnknown, yorvaruntime.SkillProjectionNotProjected), nil
	}
	return runtimeSkill(projection, yorvaruntime.SkillProjectionProjected), nil
}

func (r *RuntimeProjector) ProjectSkill(ctx context.Context, installation yorvaruntime.Installation, profileID string, request yorvaruntime.SkillProjectRequest, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.Skill{}, err
	}
	home, err := exactHermesHome()
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	absoluteSource, err := filepath.Abs(request.SourceDir)
	if err != nil || filepath.Clean(request.SourceDir) != absoluteSource {
		return yorvaruntime.Skill{}, projectionError(ErrorProjectionInvalid, errors.New("managed source path must be absolute and clean"))
	}
	managedRoot := filepath.Dir(filepath.Dir(filepath.Dir(absoluteSource)))
	projector, err := NewProjector(home, managedRoot, request.DeploymentID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	projection, err := projector.Project(ctx, ProjectRequest{
		RuntimeVersion: installation.Version,
		ProfileID:      profileID,
		SkillID:        request.SkillID,
		SourceID:       request.SourceID,
		Version:        request.Version,
		SourceDir:      absoluteSource,
		ContentSHA256:  request.ContentSHA256,
	})
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "projected"})
	}
	return runtimeSkill(projection, yorvaruntime.SkillProjectionProjected), nil
}

func (r *RuntimeProjector) UnprojectSkill(ctx context.Context, installation yorvaruntime.Installation, profileID, skillID, deploymentID string, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	home, err := exactHermesHome()
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	projector, err := NewProjector(home, home, deploymentID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if err := projector.Unproject(ctx, installation.Version, profileID, skillID); err != nil {
		return yorvaruntime.Skill{}, err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "unprojected"})
	}
	return runtimeProjectionState(skillID, yorvaruntime.SkillOwnershipYORVAManaged, yorvaruntime.SkillProjectionNotProjected), nil
}

func newReadProjector() (*Projector, error) {
	home, err := exactHermesHome()
	if err != nil {
		return nil, err
	}
	return &Projector{hermesHome: home, managedRoot: home}, nil
}

func exactHermesHome() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", projectionError(ErrorProjectionInvalid, errors.New("LOCALAPPDATA is not set"))
	}
	return cleanAbsolute(filepath.Join(localAppData, "hermes"))
}

func runtimeSkill(projection Projection, state yorvaruntime.SkillProjectionState) yorvaruntime.Skill {
	return yorvaruntime.Skill{
		ID: projection.SkillID, SourceID: projection.SourceID, Version: projection.Version,
		Ownership: yorvaruntime.SkillOwnershipYORVAManaged, ProjectionState: state,
		InstallationState: yorvaruntime.SkillInstalled, EnabledState: yorvaruntime.SkillEnabledUnknown,
		ScanState: yorvaruntime.SkillScanNotScanned,
	}
}

func runtimeProjectionState(skillID string, ownership yorvaruntime.SkillOwnership, state yorvaruntime.SkillProjectionState) yorvaruntime.Skill {
	installation := yorvaruntime.SkillNotInstalled
	if state == yorvaruntime.SkillProjectionConflict || state == yorvaruntime.SkillProjectionDriftModified {
		installation = yorvaruntime.SkillInstallationUnknown
	}
	return yorvaruntime.Skill{
		ID: skillID, Ownership: ownership, ProjectionState: state,
		InstallationState: installation, EnabledState: yorvaruntime.SkillEnabledUnknown,
		ScanState: yorvaruntime.SkillScanUnknown,
	}
}
