package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/managedskills"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
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
	if reader.installation != installation || reader.nativeID != "profile-one" {
		t.Fatalf("inspect target = %#v, %q", reader.installation, reader.nativeID)
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
			reader: &fakeSkillReader{listed: []yorvaruntime.Skill{validSkill("other")}},
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

func TestManagementSkillsManagedMutationValidatesBeforeResolution(t *testing.T) {
	tests := []struct {
		name string
		call func(*ManagementSkills) error
	}{
		{"install source", func(service *ManagementSkills) error {
			_, err := service.StartInstall(context.Background(), "inst_1", "writer", "../unapproved", "key")
			return err
		}},
		{"import source reference", func(service *ManagementSkills) error {
			_, err := service.StartImport(context.Background(), "inst_1", "writer", "C:/unsafe/skill", "key")
			return err
		}},
		{"update", func(service *ManagementSkills) error {
			_, err := service.StartUpdate(context.Background(), "inst_1", "../escape", "key")
			return err
		}},
		{"remove", func(service *ManagementSkills) error {
			_, err := service.StartRemove(context.Background(), "inst_1", "../escape", "key")
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := &fakeManagementTargetResolver{}
			if err := test.call(NewManagementSkills(resolver)); !errors.Is(err, yorvaruntime.ErrInvalidManagementContract) {
				t.Fatalf("invalid mutation error = %v", err)
			}
			if resolver.calls != 0 {
				t.Fatalf("invalid request reached resolver: %d", resolver.calls)
			}
		})
	}
}

func TestManagementSkillsManagedMutationRequiresComposition(t *testing.T) {
	resolver := &fakeManagementTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{}}}
	service := NewManagementSkills(resolver)
	if _, err := service.StartInstall(context.Background(), "inst_1", "writer", "approved.source", "key"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("StartInstall() error = %v", err)
	}
}

type fakeManagedSkillProjector struct {
	item            yorvaruntime.Skill
	projectRequests []yorvaruntime.SkillProjectRequest
	unprojectCalls  int
}

func (f *fakeManagedSkillProjector) ListSkillProjections(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.Skill, error) {
	if f.item.ID == "" {
		return nil, nil
	}
	return []yorvaruntime.Skill{f.item}, nil
}

func (f *fakeManagedSkillProjector) InspectSkillProjection(_ context.Context, _ yorvaruntime.Installation, _ string, skillID string) (yorvaruntime.Skill, error) {
	if f.item.ID != "" {
		return f.item, nil
	}
	return projectedSkill(skillID, yorvaruntime.SkillOwnershipUnknown, yorvaruntime.SkillProjectionNotProjected), nil
}

func (f *fakeManagedSkillProjector) ProjectSkill(_ context.Context, _ yorvaruntime.Installation, _ string, request yorvaruntime.SkillProjectRequest, _ yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.projectRequests = append(f.projectRequests, request)
	f.item = projectedSkill(request.SkillID, yorvaruntime.SkillOwnershipYORVAManaged, yorvaruntime.SkillProjectionProjected)
	f.item.SourceID, f.item.Version = request.SourceID, request.Version
	return f.item, nil
}

func (f *fakeManagedSkillProjector) UnprojectSkill(_ context.Context, _ yorvaruntime.Installation, _ string, skillID, _ string, _ yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	f.unprojectCalls++
	f.item = projectedSkill(skillID, yorvaruntime.SkillOwnershipUnknown, yorvaruntime.SkillProjectionNotProjected)
	return f.item, nil
}

func projectedSkill(id string, ownership yorvaruntime.SkillOwnership, state yorvaruntime.SkillProjectionState) yorvaruntime.Skill {
	return yorvaruntime.Skill{
		ID: id, Ownership: ownership, ProjectionState: state,
		InstallationState: yorvaruntime.SkillInstalled, EnabledState: yorvaruntime.SkillEnabledUnknown,
		ScanState: yorvaruntime.SkillScanClean,
	}
}

type fakeSkillLifecycle struct {
	state    yorvaruntime.LifecycleState
	restarts int
}

func (f *fakeSkillLifecycle) Status(context.Context, yorvaruntime.LifecycleInstallation, string) (yorvaruntime.LifecycleStatus, error) {
	return yorvaruntime.LifecycleStatus{State: f.state}, nil
}
func (*fakeSkillLifecycle) Start(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	return nil
}
func (*fakeSkillLifecycle) Stop(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	return nil
}
func (f *fakeSkillLifecycle) Restart(context.Context, yorvaruntime.LifecycleInstallation, string) error {
	f.restarts++
	return nil
}

func TestManagementSkillsImportRunsDurableProjectionAndAuthoritativeReadback(t *testing.T) {
	_, db, instanceID, _ := newB3ManagementTargetFixture(t)
	dataDir := t.TempDir()
	store, err := managedskills.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	sourceRef := strings.Repeat("i", 43)
	staged := filepath.Join(dataDir, "skill-imports", sourceRef, "directory")
	if err := os.MkdirAll(staged, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: imported-skill\ndescription: Imported lifecycle test.\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nplatforms: [windows, linux, macos]\n---\n\n# Imported Skill\n"
	if err := os.WriteFile(filepath.Join(staged, managedskills.SkillFileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	projector := &fakeManagedSkillProjector{}
	lifecycle := &fakeSkillLifecycle{state: yorvaruntime.LifecycleStopped}
	target := ManagementTarget{
		Installation: yorvaruntime.Installation{Path: "C:/hermes.exe", Version: "0.20.5"}, NativeID: "coder",
		Bundle: yorvaruntime.Bundle{SkillProjection: projector, Lifecycle: lifecycle},
	}
	service := NewManagedManagementSkills(&fakeManagementTargetResolver{target: target}, db, store, nil)
	started, err := service.StartImport(context.Background(), instanceID, "imported-skill", sourceRef, "import-lifecycle-key")
	if err != nil {
		t.Fatalf("StartImport() error = %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		stored, readErr := db.GetOperation(context.Background(), started.Operation.ID)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if operation.IsTerminal(stored.Status) {
			if stored.Status != operation.StatusSucceeded {
				t.Fatalf("import operation = %#v", stored)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("import operation did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	record, err := db.GetManagedSkill(context.Background(), instanceID, "imported-skill")
	if err != nil || record.SourceID != managedskills.LocalImportSourceID || record.ProjectionState != yorvaruntime.SkillProjectionProjected {
		t.Fatalf("managed import = %#v, %v", record, err)
	}
	if len(projector.projectRequests) != 1 || projector.projectRequests[0].ContentSHA256 != record.ContentSHA256 {
		t.Fatalf("projection requests = %#v", projector.projectRequests)
	}
	items, err := service.ListSkills(context.Background(), instanceID)
	if err != nil || len(items) != 1 || items[0].ID != "imported-skill" || items[0].ProjectionState != yorvaruntime.SkillProjectionProjected {
		t.Fatalf("authoritative readback = %#v, %v", items, err)
	}
}

func TestManagedSkillDisabledUpdateKeepsOwnershipAndEnableRemoveRemainAvailable(t *testing.T) {
	_, db, instanceID, _ := newB3ManagementTargetFixture(t)
	store, err := managedskills.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	acquired, err := store.AcquireToManaged(context.Background(), instanceID, "yorva-managed-demo", "yorva-demo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	record := sqlite.ManagedSkill{
		ID: "msk_test", InstanceID: instanceID, SkillID: acquired.SkillID, SourceID: acquired.SourceID,
		SourceVersion: acquired.Version, ContentSHA256: acquired.ContentSHA256,
		ManagedRelativePath: filepath.ToSlash(acquired.RelativePath), ProjectionRelativePath: acquired.SkillID,
		DeploymentID: "deploy_stable", DesiredEnabled: false, ProjectionState: yorvaruntime.SkillProjectionNotProjected,
		InstalledAt: now, UpdatedAt: now,
	}
	if err := db.CreateManagedSkill(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	projector := &fakeManagedSkillProjector{}
	lifecycle := &fakeSkillLifecycle{state: yorvaruntime.LifecycleStopped}
	target := ManagementTarget{
		Installation: yorvaruntime.Installation{Path: "C:/hermes.exe", Version: "0.20.5"}, NativeID: "coder",
		Bundle: yorvaruntime.Bundle{SkillProjection: projector, Lifecycle: lifecycle},
	}
	service := NewManagedManagementSkills(&fakeManagementTargetResolver{target: target}, db, store, nil)
	op := operation.Operation{TargetID: instanceID, Message: record.SkillID, SourcePin: record.SourceID, OwnershipNonce: "unused_new_deployment"}
	if err := service.updateManaged(context.Background(), op, target); err != nil {
		t.Fatalf("disabled update error = %v", err)
	}
	updated, err := db.GetManagedSkill(context.Background(), instanceID, record.SkillID)
	if err != nil || updated.DesiredEnabled || updated.DeploymentID != "deploy_stable" || len(projector.projectRequests) != 0 || lifecycle.restarts != 0 {
		t.Fatalf("disabled update = %#v err=%v projects=%d restarts=%d", updated, err, len(projector.projectRequests), lifecycle.restarts)
	}
	if err := service.enableManaged(context.Background(), op, target); err != nil {
		t.Fatalf("enable after disabled projection error = %v", err)
	}
	if len(projector.projectRequests) != 1 || projector.projectRequests[0].DeploymentID != "deploy_stable" || lifecycle.restarts != 0 {
		t.Fatalf("enable project=%#v restarts=%d", projector.projectRequests, lifecycle.restarts)
	}
	lifecycle.state = yorvaruntime.LifecycleRunning
	if err := service.disableManaged(context.Background(), op, target); err != nil {
		t.Fatalf("disable error = %v", err)
	}
	if lifecycle.restarts != 1 || projector.unprojectCalls != 1 {
		t.Fatalf("running disable restarts=%d unprojects=%d", lifecycle.restarts, projector.unprojectCalls)
	}
	if err := service.enableManaged(context.Background(), op, target); err != nil {
		t.Fatalf("enable after disable error = %v", err)
	}
	if lifecycle.restarts != 2 {
		t.Fatalf("running enable restarts=%d", lifecycle.restarts)
	}
}

func TestManagedSkillExternalProjectionCannotBeMutatedAndMissingIsReported(t *testing.T) {
	_, db, instanceID, _ := newB3ManagementTargetFixture(t)
	store, err := managedskills.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	acquired, err := store.AcquireToManaged(context.Background(), instanceID, "yorva-managed-demo", "yorva-demo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	record := sqlite.ManagedSkill{
		ID: "msk_external", InstanceID: instanceID, SkillID: acquired.SkillID, SourceID: acquired.SourceID,
		SourceVersion: acquired.Version, ContentSHA256: acquired.ContentSHA256,
		ManagedRelativePath: filepath.ToSlash(acquired.RelativePath), ProjectionRelativePath: acquired.SkillID,
		DeploymentID: "deploy_external", DesiredEnabled: true, ProjectionState: yorvaruntime.SkillProjectionProjected,
		InstalledAt: now, UpdatedAt: now,
	}
	if err := db.CreateManagedSkill(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	projector := &fakeManagedSkillProjector{item: projectedSkill(record.SkillID, yorvaruntime.SkillOwnershipExternal, yorvaruntime.SkillProjectionProjected)}
	target := ManagementTarget{Bundle: yorvaruntime.Bundle{SkillProjection: projector}}
	service := NewManagedManagementSkills(&fakeManagementTargetResolver{target: target}, db, store, nil)
	op := operation.Operation{TargetID: instanceID, Message: record.SkillID}
	if err := service.removeManaged(context.Background(), op, target); !errors.Is(err, ErrSkillOwnershipConflict) {
		t.Fatalf("external remove error = %v", err)
	}
	if projector.unprojectCalls != 0 {
		t.Fatalf("external projection was unprojected")
	}
	projector.item = yorvaruntime.Skill{}
	items, err := service.ListSkills(context.Background(), instanceID)
	if err != nil || len(items) != 1 || items[0].ProjectionState != yorvaruntime.SkillProjectionDriftMissing {
		t.Fatalf("missing projection inventory = %#v, %v", items, err)
	}
}
