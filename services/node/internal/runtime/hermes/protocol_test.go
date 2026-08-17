package hermes

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestParseAndValidateReviewedManifest(t *testing.T) {
	payload, err := json.Marshal(reviewedManifest())
	if err != nil {
		t.Fatal(err)
	}
	if err := parseAndValidateManifest("banner\n" + string(payload) + "\n"); err != nil {
		t.Fatal(err)
	}
}

func TestManifestRejectsUnknownMissingDuplicateReorderedAndInteractiveChanges(t *testing.T) {
	base := reviewedManifest()
	tests := []struct {
		name string
		mut  func(*installerManifest)
	}{
		{name: "unknown", mut: func(m *installerManifest) { m.Stages[0].Name = "unexpected" }},
		{name: "missing", mut: func(m *installerManifest) { m.Stages = m.Stages[1:] }},
		{name: "duplicate", mut: func(m *installerManifest) { m.Stages[1].Name = "uv" }},
		{name: "reordered", mut: func(m *installerManifest) { m.Stages[0], m.Stages[1] = m.Stages[1], m.Stages[0] }},
		{name: "interactive", mut: func(m *installerManifest) { m.Stages[0].NeedsUserInput = true }},
		{name: "desktop present", mut: func(m *installerManifest) {
			m.Stages = append([]manifestStage{{Name: "desktop", Category: "install"}}, m.Stages...)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := reviewedManifest()
			test.mut(&manifest)
			payload, _ := json.Marshal(manifest)
			if err := parseAndValidateManifest(string(payload)); installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallManifestMismatch {
				t.Fatalf("error = %v", err)
			}
		})
	}
	_ = base
}

func TestExcludedStagesNeverAppearInApprovedSpawnSet(t *testing.T) {
	for _, excluded := range excludedInstallStages() {
		if isApprovedStage(excluded) {
			t.Fatalf("excluded stage %s is approved", excluded)
		}
		_, err := stageInvocation("powershell.exe", `C:\op\install.ps1`, excluded, `C:\h`, `C:\h\hermes-agent`)
		if err == nil {
			t.Fatalf("stageInvocation(%s) unexpectedly succeeded", excluded)
		}
	}
	for _, owned := range yorvaOwnedOfficialStages() {
		_, err := stageInvocation("powershell.exe", `C:\op\install.ps1`, owned, `C:\h`, `C:\h\hermes-agent`)
		if err == nil {
			t.Fatalf("stageInvocation(%s) unexpectedly succeeded", owned)
		}
	}
	if slices.Contains(approvedInstallStages(), "desktop") {
		t.Fatal("desktop must never be an approved execution stage")
	}
}

func TestApprovedStageUsesClosedArgv(t *testing.T) {
	got, err := stageInvocation(`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, `D:\state\operations\op1\install.ps1`, "uv", `C:\Users\a\AppData\Local\hermes`, `C:\Users\a\AppData\Local\hermes\hermes-agent`)
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := []string{
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", `D:\state\operations\op1\install.ps1`,
		"-Stage", "uv", "-NonInteractive", "-Json",
		"-Branch", "main", "-Commit", officialCommit,
	}
	if !slices.Equal(got.Args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("argv = %#v", got.Args)
	}
	joined := strings.Join(got.Args, " ")
	if strings.Contains(joined, "cmd /c") || strings.Contains(joined, "irm") {
		t.Fatalf("argv looks like a shell string: %s", joined)
	}
}

func TestParseProtocolVersionAndStageResult(t *testing.T) {
	if err := parseProtocolVersion("1\n"); err != nil {
		t.Fatal(err)
	}
	if err := parseProtocolVersion("2"); installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallProtocolUnsupported {
		t.Fatalf("protocol error = %v", err)
	}
	got, err := parseStageResult("uv", "noise\n{\"stage\":\"uv\",\"ok\":true,\"skipped\":false,\"reason\":\"\",\"duration_ms\":12}\n")
	if err != nil || !got.OK || got.Stage != "uv" {
		t.Fatalf("parseStageResult() = %#v, %v", got, err)
	}
	if _, err := parseStageResult("uv", "{\"stage\":\"python\",\"ok\":true}"); err == nil {
		t.Fatal("mismatched stage unexpectedly accepted")
	}
}

func TestInstallerEnvironmentExcludesSentinelSecrets(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-secret")
	t.Setenv("HERMES_TOKEN", "hermes-secret")
	t.Setenv("PYTHONPATH", "evil")
	t.Setenv("UV_INDEX", "http://evil")
	t.Setenv("UV_DEFAULT_INDEX", "http://evil-python")
	t.Setenv("PIP_INDEX_URL", "http://evil-pip")
	t.Setenv("NPM_CONFIG_REGISTRY", "http://evil-npm")
	env := installerEnvironment(`C:\Users\a\AppData\Local\hermes`)
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"sk-secret", "hermes-secret", "http://evil", "OPENAI_API_KEY", "HERMES_TOKEN=", "PYTHONPATH=", "UV_INDEX="} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("environment leaked %q: %s", forbidden, joined)
		}
	}
	if !strings.Contains(joined, `HERMES_HOME=C:\Users\a\AppData\Local\hermes`) {
		t.Fatalf("fixed HERMES_HOME missing: %s", joined)
	}
	for _, fixed := range []string{
		"UV_DEFAULT_INDEX=" + hermesPythonIndex,
		"UV_INDEX_STRATEGY=first-index",
		"PIP_INDEX_URL=" + hermesPythonIndex,
		"PIP_CONFIG_FILE=NUL",
		"NPM_CONFIG_REGISTRY=" + hermesNPMRegistry,
	} {
		if count := strings.Count(joined, fixed); count != 1 {
			t.Fatalf("fixed environment %q count = %d: %s", fixed, count, joined)
		}
	}
}
