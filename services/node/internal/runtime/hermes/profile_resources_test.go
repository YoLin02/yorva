package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestProfileResourceReaderListsExactProfileSkillsAndSafeMCPState(t *testing.T) {
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles", "work")
	writeProfileResourceFixture(t, filepath.Join(profileRoot, "skills", "development", "github", "SKILL.md"), "---\nname: github\nversion: 2.1.0\ndescription: Manage GitHub work.\n---\n# GitHub\nPassword is `must-not-project`\nUse <section> and https://example.invalid/docs.")
	writeProfileResourceFixture(t, filepath.Join(profileRoot, "config.yaml"), "skills:\n  disabled: [github]\nmcp_servers:\n  github:\n    url: https://example.invalid/mcp\n    headers:\n      Authorization: Bearer must-not-project\n  local_tools:\n    command: npx\n    env:\n      TOKEN: must-not-project\n")
	observedAt := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	reader := newProfileResourceReaderAt(root)
	reader.now = func() time.Time { return observedAt }
	installation := profileResourceInstallation(root)

	skills, err := reader.ListSkills(context.Background(), installation, "work")
	if err != nil || len(skills) != 1 {
		t.Fatalf("ListSkills() = %#v, %v", skills, err)
	}
	if skill := skills[0]; skill.ID != "github" || skill.Version != "2.1.0" || skill.Description != "Manage GitHub work." || skill.Preview != "" || skill.Ownership != yorvaruntime.SkillOwnershipExternal || skill.InstallationState != yorvaruntime.SkillInstalled || skill.EnabledState != yorvaruntime.SkillDisabled {
		t.Fatalf("skill = %#v", skill)
	}
	inspected, err := reader.InspectSkill(context.Background(), installation, "work", "github")
	if err != nil || !strings.Contains(inspected.Preview, "# GitHub") || !strings.Contains(inspected.Preview, "Password=[REDACTED]") || !strings.Contains(inspected.Preview, "Use <section> and https://example.invalid/docs.") || strings.Contains(inspected.Preview, "must-not-project") {
		t.Fatalf("InspectSkill() = %#v, %v", inspected, err)
	}

	servers, err := reader.ListMCPServers(context.Background(), installation, "work")
	if err != nil || len(servers) != 2 {
		t.Fatalf("ListMCPServers() = %#v, %v", servers, err)
	}
	if servers[0].ID != "github" || servers[0].PresetID != "github" || servers[0].State != yorvaruntime.MCPConfigured || servers[0].ObservedAt != observedAt || servers[1].ID != "local_tools" {
		t.Fatalf("servers = %#v", servers)
	}
}

func TestProfileResourceReaderKeepsProfilesSeparate(t *testing.T) {
	root := t.TempDir()
	writeProfileResourceFixture(t, filepath.Join(root, "skills", "default-only", "SKILL.md"), "---\nname: default-only\n---\n")
	writeProfileResourceFixture(t, filepath.Join(root, "profiles", "work", "skills", "work-only", "SKILL.md"), "---\nname: work-only\n---\n")
	reader := newProfileResourceReaderAt(root)
	installation := profileResourceInstallation(root)

	defaults, err := reader.ListSkills(context.Background(), installation, "default")
	if err != nil || len(defaults) != 1 || defaults[0].ID != "default-only" {
		t.Fatalf("default skills = %#v, %v", defaults, err)
	}
	work, err := reader.ListSkills(context.Background(), installation, "work")
	if err != nil || len(work) != 1 || work[0].ID != "work-only" {
		t.Fatalf("work skills = %#v, %v", work, err)
	}
}

func TestManagementResolverPublishesResourcesWithoutAPIListener(t *testing.T) {
	root := t.TempDir()
	writeProfileResourceFixture(t, filepath.Join(root, "skills", "github", "SKILL.md"), "---\nname: github\n---\n")
	resources := newProfileResourceReaderAt(root)
	resolver := &ManagementResolver{resources: resources, api: &APIManagementReader{}}
	features, err := resolver.ResolveInstanceManagement(context.Background(), profileResourceInstallation(root), "default")
	if err != nil || features.SkillRead == nil || features.MCPRead == nil || features.Health == nil || features.Logs == nil {
		t.Fatalf("features = %#v, %v", features, err)
	}
}

func writeProfileResourceFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func profileResourceInstallation(root string) yorvaruntime.Installation {
	return yorvaruntime.Installation{RuntimeKind: Kind, Version: "0.20.5", Path: filepath.Join(root, "bin", "hermes.exe"), SupportState: yorvaruntime.DiscoverySupported}
}
