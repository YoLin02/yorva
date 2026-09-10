package app

import (
	"context"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

type managementCapabilityFixture struct{}

func (managementCapabilityFixture) InspectRuntimeHealth(context.Context, yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	return yorvaruntime.HealthObservation{}, nil
}

func (managementCapabilityFixture) InspectInstanceHealth(context.Context, yorvaruntime.Installation, string) (yorvaruntime.HealthObservation, error) {
	return yorvaruntime.HealthObservation{}, nil
}

func (managementCapabilityFixture) ReadLogSnapshot(context.Context, yorvaruntime.Installation, string, yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	return yorvaruntime.LogSnapshot{}, nil
}

func (managementCapabilityFixture) AuditRuntimeSecurity(context.Context, yorvaruntime.Installation) (yorvaruntime.SecurityAuditResult, error) {
	return yorvaruntime.SecurityAuditResult{}, nil
}

func (managementCapabilityFixture) AuditInstanceSecurity(context.Context, yorvaruntime.Installation, string) (yorvaruntime.SecurityAuditResult, error) {
	return yorvaruntime.SecurityAuditResult{}, nil
}

func (managementCapabilityFixture) ListSkills(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.Skill, error) {
	return nil, nil
}

func (managementCapabilityFixture) InspectSkill(context.Context, yorvaruntime.Installation, string, string) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) InstallSkill(context.Context, yorvaruntime.Installation, string, yorvaruntime.SkillInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) UpdateSkill(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) RemoveSkill(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) ConfigureSkill(context.Context, yorvaruntime.Installation, string, yorvaruntime.SkillConfigureRequest, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) ListSkillProjections(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.Skill, error) {
	return nil, nil
}

func (managementCapabilityFixture) InspectSkillProjection(context.Context, yorvaruntime.Installation, string, string) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) ProjectSkill(context.Context, yorvaruntime.Installation, string, yorvaruntime.SkillProjectRequest, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) UnprojectSkill(context.Context, yorvaruntime.Installation, string, string, string, yorvaruntime.ProgressSink) (yorvaruntime.Skill, error) {
	return yorvaruntime.Skill{}, nil
}

func (managementCapabilityFixture) ListMCPServers(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPServer, error) {
	return nil, nil
}

func (managementCapabilityFixture) ListMCPPresets(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPPreset, error) {
	return nil, nil
}

func (managementCapabilityFixture) InstallMCPPreset(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, nil
}

func (managementCapabilityFixture) AuthenticateMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPAuthenticateRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, nil
}

func (managementCapabilityFixture) TestMCP(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.MCPTestResult, error) {
	return yorvaruntime.MCPTestResult{}, nil
}

func (managementCapabilityFixture) ConfigureMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPConfigureRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, nil
}

func (managementCapabilityFixture) RemoveMCP(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, nil
}

func (managementCapabilityFixture) ListBackups(context.Context, yorvaruntime.Installation) ([]yorvaruntime.Backup, error) {
	return nil, nil
}

func (managementCapabilityFixture) GetBackup(context.Context, yorvaruntime.Installation, string) (yorvaruntime.Backup, error) {
	return yorvaruntime.Backup{}, nil
}

func (managementCapabilityFixture) CreateBackup(context.Context, yorvaruntime.Installation, yorvaruntime.BackupCreateRequest, yorvaruntime.ProgressSink) (yorvaruntime.Backup, error) {
	return yorvaruntime.Backup{}, nil
}

func (managementCapabilityFixture) DeleteBackup(context.Context, yorvaruntime.Installation, string, yorvaruntime.ProgressSink) error {
	return nil
}

func (managementCapabilityFixture) RestoreBackup(context.Context, yorvaruntime.Installation, yorvaruntime.BackupRestoreRequest, yorvaruntime.ProgressSink) (yorvaruntime.RestoreResult, error) {
	return yorvaruntime.RestoreResult{}, nil
}

func (managementCapabilityFixture) PlanUpgrade(context.Context, yorvaruntime.Installation) (yorvaruntime.UpgradePlan, error) {
	return yorvaruntime.UpgradePlan{}, nil
}

func (managementCapabilityFixture) UpgradeRuntime(context.Context, yorvaruntime.Installation, yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	return yorvaruntime.UpgradeResult{}, nil
}

func (managementCapabilityFixture) RollbackRuntime(context.Context, yorvaruntime.Installation, yorvaruntime.ProgressSink) (yorvaruntime.UpgradeResult, error) {
	return yorvaruntime.UpgradeResult{}, nil
}

func TestInstanceCapabilitiesProjectRegistryManagementWiring(t *testing.T) {
	fixture := managementCapabilityFixture{}
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor:      yorvaruntime.Descriptor{Kind: "hermes", Name: "Fixture Hermes"},
		Health:          fixture,
		Logs:            fixture,
		Security:        fixture,
		SkillRead:       fixture,
		SkillMutate:     fixture,
		SkillProjection: fixture,
		MCPRead:         fixture,
		MCPMutate:       fixture,
		BackupRead:      fixture,
		BackupMutate:    fixture,
		Restore:         fixture,
		UpgradePlan:     fixture,
		Upgrade:         fixture,
		Rollback:        fixture,
	}); err != nil {
		t.Fatal(err)
	}

	inventory := &InstanceInventory{discovery: &RuntimeDiscovery{registry: registry}}
	if got := inventory.capabilities("hermes"); got != (InstanceCapabilities{
		HealthRead: true, LogsRead: true, SecurityAudit: true,
		SkillRead: true, SkillMutate: true, MCPRead: true, MCPMutate: true, MCPTest: true,
		BackupRead: true, BackupMutate: true, Restore: true, UpgradePlan: true,
		Upgrade: true, Rollback: true,
	}) {
		t.Fatalf("projected fixture capabilities = %#v", got)
	}
}

func TestInstanceCapabilitiesExposeHermesUpgradePlanReadOnly(t *testing.T) {
	registry := yorvaruntime.NewRegistry()
	if err := hermes.Register(registry); err != nil {
		t.Fatal(err)
	}
	inventory := &InstanceInventory{discovery: &RuntimeDiscovery{registry: registry}}
	got := inventory.capabilities(hermes.Kind)
	if !got.Instances || !got.Lifecycle || !got.SkillMutate || !got.MCPRead || !got.MCPMutate || !got.MCPTest || !got.UpgradePlan || got.NativeSkills.NativeInstall.Supported || got.NativeSkills.NativeInstall.Reason != "deferred_upstream" {
		t.Fatalf("actual Hermes capabilities = %#v, want managed projection with deferred native mutation", got)
	}
}
