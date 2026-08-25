package app

import (
	"context"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const maxManagementSkillItems = 256

// ManagementSkills exposes Runtime-neutral Skill use cases for one resolved
// Instance target. Hermes-specific scope and transport details remain owned by
// the Runtime adapter.
type ManagementSkills struct {
	targets ManagementTargetResolver
}

func NewManagementSkills(targets ManagementTargetResolver) *ManagementSkills {
	return &ManagementSkills{targets: targets}
}

func (s *ManagementSkills) ListSkills(ctx context.Context, instanceID string) ([]yorvaruntime.Skill, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.SkillRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	items, err := target.Bundle.SkillRead.ListSkills(ctx, target.Installation, target.NativeID)
	if err != nil {
		return nil, managementQueryError(ctx, err)
	}
	if len(items) > maxManagementSkillItems {
		return nil, ErrManagementQueryFailed
	}
	result := make([]yorvaruntime.Skill, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Validate() != nil {
			return nil, ErrManagementQueryFailed
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return nil, ErrManagementQueryFailed
		}
		seen[item.ID] = struct{}{}
		result = append(result, item)
	}
	return result, nil
}

func (s *ManagementSkills) InspectSkill(ctx context.Context, instanceID, skillID string) (yorvaruntime.Skill, error) {
	if err := validateSkillID(skillID); err != nil {
		return yorvaruntime.Skill{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if target.Bundle.SkillRead == nil {
		return yorvaruntime.Skill{}, ErrManagementCapabilityUnsupported
	}

	item, err := target.Bundle.SkillRead.InspectSkill(ctx, target.Installation, target.NativeID, skillID)
	if err != nil {
		return yorvaruntime.Skill{}, managementQueryError(ctx, err)
	}
	if item.Validate() != nil || item.ID != skillID {
		return yorvaruntime.Skill{}, ErrManagementQueryFailed
	}
	return item, nil
}

// InstallSkill is a synchronous application boundary only. No HTTP handler is
// exposed until an Operation owner/worker exists. A missing mutation adapter is
// always reported as capability unsupported.
func (s *ManagementSkills) InstallSkill(ctx context.Context, instanceID string, request yorvaruntime.SkillInstallRequest, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.Skill{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if target.Bundle.SkillMutate == nil {
		return yorvaruntime.Skill{}, ErrManagementCapabilityUnsupported
	}
	item, err := target.Bundle.SkillMutate.InstallSkill(ctx, target.Installation, target.NativeID, request, progress)
	return validateSkillMutationResult(ctx, item, "", err)
}

// UpdateSkill is not exposed through HTTP until durable Operation orchestration
// is implemented.
func (s *ManagementSkills) UpdateSkill(ctx context.Context, instanceID, skillID string, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	if err := validateSkillID(skillID); err != nil {
		return yorvaruntime.Skill{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if target.Bundle.SkillMutate == nil {
		return yorvaruntime.Skill{}, ErrManagementCapabilityUnsupported
	}
	item, err := target.Bundle.SkillMutate.UpdateSkill(ctx, target.Installation, target.NativeID, skillID, progress)
	return validateSkillMutationResult(ctx, item, skillID, err)
}

// RemoveSkill is not exposed through HTTP until durable Operation orchestration
// is implemented.
func (s *ManagementSkills) RemoveSkill(ctx context.Context, instanceID, skillID string, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	if err := validateSkillID(skillID); err != nil {
		return yorvaruntime.Skill{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if target.Bundle.SkillMutate == nil {
		return yorvaruntime.Skill{}, ErrManagementCapabilityUnsupported
	}
	item, err := target.Bundle.SkillMutate.RemoveSkill(ctx, target.Installation, target.NativeID, skillID, progress)
	return validateSkillMutationResult(ctx, item, skillID, err)
}

// ConfigureSkill is not exposed through HTTP until durable Operation
// orchestration is implemented.
func (s *ManagementSkills) ConfigureSkill(ctx context.Context, instanceID string, request yorvaruntime.SkillConfigureRequest, progress yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.Skill{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	if target.Bundle.SkillMutate == nil {
		return yorvaruntime.Skill{}, ErrManagementCapabilityUnsupported
	}
	item, err := target.Bundle.SkillMutate.ConfigureSkill(ctx, target.Installation, target.NativeID, request, progress)
	return validateSkillMutationResult(ctx, item, request.SkillID, err)
}

func (s *ManagementSkills) resolve(ctx context.Context, instanceID string) (ManagementTarget, error) {
	if s == nil || s.targets == nil {
		return ManagementTarget{}, ErrManagementQueryFailed
	}
	if instanceID == "" {
		return ManagementTarget{}, ErrInstanceNotFound
	}
	target, err := s.targets.ResolveManagementTarget(ctx, instanceID)
	if err != nil {
		return ManagementTarget{}, managementQueryError(ctx, err)
	}
	return target, nil
}

func validateSkillID(skillID string) error {
	return (yorvaruntime.SkillConfigureRequest{SkillID: skillID}).Validate()
}

func validateSkillMutationResult(ctx context.Context, item yorvaruntime.Skill, expectedID string, err error) (yorvaruntime.Skill, error) {
	if err != nil {
		return yorvaruntime.Skill{}, managementQueryError(ctx, err)
	}
	if item.Validate() != nil || expectedID != "" && item.ID != expectedID {
		return yorvaruntime.Skill{}, ErrManagementQueryFailed
	}
	return item, nil
}
