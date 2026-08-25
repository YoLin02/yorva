package app

import (
	"context"
	"errors"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeManagementTargetResolver struct {
	target ManagementTarget
	err    error
	calls  int
}

func (f *fakeManagementTargetResolver) ResolveManagementTarget(context.Context, string) (ManagementTarget, error) {
	f.calls++
	return f.target, f.err
}

type fakeSkillReader struct {
	listed       []yorvaruntime.Skill
	inspected    yorvaruntime.Skill
	err          error
	installation yorvaruntime.Installation
	nativeID     string
	skillID      string
}

func (f *fakeSkillReader) ListSkills(_ context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.Skill, error) {
	f.installation = installation
	f.nativeID = nativeID
	return f.listed, f.err
}

func (f *fakeSkillReader) InspectSkill(_ context.Context, installation yorvaruntime.Installation, nativeID, skillID string) (yorvaruntime.Skill, error) {
	f.installation = installation
	f.nativeID = nativeID
	f.skillID = skillID
	return f.inspected, f.err
}

type fakeSkillManager struct {
	calls  int
	result yorvaruntime.Skill
}

func (f *fakeSkillManager) InstallSkill(context.Context, yorvaruntime.Installation, string, yorvaruntime.SkillInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.calls++
	return f.result, nil
}

func (f *fakeSkillManager) UpdateSkill(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.calls++
	return f.result, nil
}

func (f *fakeSkillManager) RemoveSkill(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.calls++
	return f.result, nil
}

func (f *fakeSkillManager) ConfigureSkill(context.Context, yorvaruntime.Installation, string, yorvaruntime.SkillConfigureRequest, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.calls++
	return f.result, nil
}

func validSkill(id string) yorvaruntime.Skill {
	return yorvaruntime.Skill{
		ID:                id,
		SourceID:          "approved.source",
		Version:           "1.0.0",
		InstallationState: yorvaruntime.SkillInstalled,
		EnabledState:      yorvaruntime.SkillEnabled,
		ScanState:         yorvaruntime.SkillScanClean,
	}
}

func TestManagementSkillsListAndInspectUseResolvedTarget(t *testing.T) {
	installation := yorvaruntime.Installation{RuntimeKind: "hermes", Path: "C:/managed/hermes.exe", Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported}
	reader := &fakeSkillReader{listed: []yorvaruntime.Skill{validSkill("writer")}, inspected: validSkill("writer")}
	resolver := &fakeManagementTargetResolver{target: ManagementTarget{
		Installation: installation,
		NativeID:     "profile-one",
		Bundle:       yorvaruntime.Bundle{SkillRead: reader},
	}}
	service := NewManagementSkills(resolver)

	listed, err := service.ListSkills(context.Background(), "inst_1")
	if err != nil || len(listed) != 1 || listed[0].ID != "writer" {
		t.Fatalf("ListSkills() = %#v, %v", listed, err)
	}
	if reader.installation != installation || reader.nativeID != "profile-one" {
		t.Fatalf("list target = %#v, %q", reader.installation, reader.nativeID)
	}

	inspected, err := service.InspectSkill(context.Background(), "inst_1", "writer")
	if err != nil || inspected.ID != "writer" {
		t.Fatalf("InspectSkill() = %#v, %v", inspected, err)
	}
	if reader.installation != installation || reader.nativeID != "profile-one" || reader.skillID != "writer" {
		t.Fatalf("inspect target = %#v, %q, %q", reader.installation, reader.nativeID, reader.skillID)
	}
}

func TestManagementSkillsRejectsInvalidAdapterResults(t *testing.T) {
	tests := []struct {
		name   string
		reader *fakeSkillReader
		call   func(*ManagementSkills) error
	}{
		{
			name:   "invalid list item",
			reader: &fakeSkillReader{listed: []yorvaruntime.Skill{{ID: "../escape"}}},
			call: func(service *ManagementSkills) error {
				_, err := service.ListSkills(context.Background(), "inst_1")
				return err
			},
		},
		{
			name:   "duplicate list item",
			reader: &fakeSkillReader{listed: []yorvaruntime.Skill{validSkill("writer"), validSkill("writer")}},
			call: func(service *ManagementSkills) error {
				_, err := service.ListSkills(context.Background(), "inst_1")
				return err
			},
		},
		{
			name:   "inspect id mismatch",
			reader: &fakeSkillReader{inspected: validSkill("other")},
			call: func(service *ManagementSkills) error {
				_, err := service.InspectSkill(context.Background(), "inst_1", "writer")
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{SkillRead: test.reader}}}
			if err := test.call(NewManagementSkills(resolver)); !errors.Is(err, ErrManagementQueryFailed) {
				t.Fatalf("error = %v, want ErrManagementQueryFailed", err)
			}
		})
	}
}

func TestManagementSkillsCapabilityFalseIsStable(t *testing.T) {
	resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{}}}
	service := NewManagementSkills(resolver)
	if _, err := service.ListSkills(context.Background(), "inst_1"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if _, err := service.InspectSkill(context.Background(), "inst_1", "writer"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("InspectSkill() error = %v", err)
	}
}

func TestManagementSkillsMutationValidatesBeforeAdapter(t *testing.T) {
	tests := []struct {
		name string
		call func(*ManagementSkills) error
	}{
		{"install", func(service *ManagementSkills) error {
			_, err := service.InstallSkill(context.Background(), "inst_1", yorvaruntime.SkillInstallRequest{SourceID: "../unapproved"}, nil)
			return err
		}},
		{"update", func(service *ManagementSkills) error {
			_, err := service.UpdateSkill(context.Background(), "inst_1", "../escape", nil)
			return err
		}},
		{"remove", func(service *ManagementSkills) error {
			_, err := service.RemoveSkill(context.Background(), "inst_1", "../escape", nil)
			return err
		}},
		{"configure", func(service *ManagementSkills) error {
			_, err := service.ConfigureSkill(context.Background(), "inst_1", yorvaruntime.SkillConfigureRequest{SkillID: "../escape"}, nil)
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := &fakeSkillManager{result: validSkill("writer")}
			resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{SkillMutate: manager}}}
			if err := test.call(NewManagementSkills(resolver)); !errors.Is(err, yorvaruntime.ErrInvalidManagementContract) {
				t.Fatalf("invalid mutation error = %v", err)
			}
			if resolver.calls != 0 || manager.calls != 0 {
				t.Fatalf("invalid request reached resolver/adapter: resolver=%d adapter=%d", resolver.calls, manager.calls)
			}
		})
	}

	manager := &fakeSkillManager{result: validSkill("writer")}
	resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{SkillMutate: manager}}}
	service := NewManagementSkills(resolver)
	if _, err := service.InstallSkill(context.Background(), "inst_1", yorvaruntime.SkillInstallRequest{SourceID: "approved.source"}, nil); err != nil {
		t.Fatalf("valid InstallSkill() error = %v", err)
	}
	if resolver.calls != 1 || manager.calls != 1 {
		t.Fatalf("valid request calls: resolver=%d adapter=%d", resolver.calls, manager.calls)
	}
}

func TestManagementSkillsMutationRequiresCapability(t *testing.T) {
	resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{}}}
	service := NewManagementSkills(resolver)
	if _, err := service.ConfigureSkill(context.Background(), "inst_1", yorvaruntime.SkillConfigureRequest{SkillID: "writer", Enabled: true}, nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("ConfigureSkill() error = %v", err)
	}
}
