package app

// ManagementActor identifies the authenticated trust context that initiated a
// management use case. Phase 7 intentionally has one actor and does not model
// principals, grants, or roles.
type ManagementActor string

const ManagementActorLocalDesktop ManagementActor = "LOCAL_DESKTOP"

func (a ManagementActor) Valid() bool {
	return a == ManagementActorLocalDesktop
}

// ManagementAction is the stable authorization and audit identity for a Phase
// 7 use case. It is never a Runtime command or argument.
type ManagementAction string

const (
	ActionRuntimeHealthRead    ManagementAction = "runtime.health.read"
	ActionRuntimeLogsRead      ManagementAction = "runtime.logs.read"
	ActionRuntimeSecurityAudit ManagementAction = "runtime.security.audit"

	ActionSkillRead      ManagementAction = "skill.read"
	ActionSkillInstall   ManagementAction = "skill.install"
	ActionSkillUpdate    ManagementAction = "skill.update"
	ActionSkillRemove    ManagementAction = "skill.remove"
	ActionSkillConfigure ManagementAction = "skill.configure"

	ActionMCPRead         ManagementAction = "mcp.read"
	ActionMCPInstall      ManagementAction = "mcp.install"
	ActionMCPAuthenticate ManagementAction = "mcp.authenticate"
	ActionMCPTest         ManagementAction = "mcp.test"
	ActionMCPRemove       ManagementAction = "mcp.remove"
	ActionMCPConfigure    ManagementAction = "mcp.configure"

	ActionBackupRead    ManagementAction = "backup.read"
	ActionBackupCreate  ManagementAction = "backup.create"
	ActionBackupRestore ManagementAction = "backup.restore"
	ActionBackupDelete  ManagementAction = "backup.delete"

	ActionRuntimeUpgradePlan ManagementAction = "runtime.upgrade.plan"
	ActionRuntimeUpgrade     ManagementAction = "runtime.upgrade"
	ActionRuntimeRollback    ManagementAction = "runtime.rollback"
)

var managementActions = [...]ManagementAction{
	ActionRuntimeHealthRead,
	ActionRuntimeLogsRead,
	ActionRuntimeSecurityAudit,
	ActionSkillRead,
	ActionSkillInstall,
	ActionSkillUpdate,
	ActionSkillRemove,
	ActionSkillConfigure,
	ActionMCPRead,
	ActionMCPInstall,
	ActionMCPAuthenticate,
	ActionMCPTest,
	ActionMCPRemove,
	ActionMCPConfigure,
	ActionBackupRead,
	ActionBackupCreate,
	ActionBackupRestore,
	ActionBackupDelete,
	ActionRuntimeUpgradePlan,
	ActionRuntimeUpgrade,
	ActionRuntimeRollback,
}

func (a ManagementAction) Valid() bool {
	for _, candidate := range managementActions {
		if a == candidate {
			return true
		}
	}
	return false
}

// ManagementActions returns a copy so callers cannot change the validation
// allowlist.
func ManagementActions() []ManagementAction {
	actions := make([]ManagementAction, len(managementActions))
	copy(actions, managementActions[:])
	return actions
}
