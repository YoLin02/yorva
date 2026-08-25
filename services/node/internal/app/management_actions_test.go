package app

import "testing"

func TestManagementActionsAreStableUniqueAndValid(t *testing.T) {
	want := []ManagementAction{
		"runtime.health.read",
		"runtime.logs.read",
		"runtime.security.audit",
		"skill.read",
		"skill.install",
		"skill.update",
		"skill.remove",
		"skill.configure",
		"mcp.read",
		"mcp.install",
		"mcp.authenticate",
		"mcp.test",
		"mcp.remove",
		"mcp.configure",
		"backup.read",
		"backup.create",
		"backup.restore",
		"backup.delete",
		"runtime.upgrade.plan",
		"runtime.upgrade",
		"runtime.rollback",
	}

	got := ManagementActions()
	if len(got) != len(want) {
		t.Fatalf("ManagementActions() length = %d, want %d", len(got), len(want))
	}
	seen := make(map[ManagementAction]struct{}, len(got))
	for i, action := range got {
		if action != want[i] {
			t.Fatalf("ManagementActions()[%d] = %q, want %q", i, action, want[i])
		}
		if !action.Valid() {
			t.Fatalf("declared action %q is not valid", action)
		}
		if _, duplicate := seen[action]; duplicate {
			t.Fatalf("duplicate management action %q", action)
		}
		seen[action] = struct{}{}
	}

	got[0] = "changed"
	if ManagementActions()[0] != ActionRuntimeHealthRead {
		t.Fatal("ManagementActions returned mutable package state")
	}
	if ManagementAction("").Valid() || ManagementAction("shell.exec").Valid() {
		t.Fatal("unknown management action was accepted")
	}
}

func TestPhase7ManagementActorIsClosed(t *testing.T) {
	if !ManagementActorLocalDesktop.Valid() {
		t.Fatal("LOCAL_DESKTOP actor is invalid")
	}
	if ManagementActor("").Valid() || ManagementActor("ADMIN").Valid() {
		t.Fatal("unknown actor was accepted")
	}
}
