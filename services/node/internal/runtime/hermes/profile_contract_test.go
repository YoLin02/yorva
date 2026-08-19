package hermes

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestProfileSurfaceIsPinnedCLIWithoutStructuredList(t *testing.T) {
	if profileOfficialVersion != "0.20.2" || profileOfficialCommit != "df4b65147d7ddd74dd449f9067aabbca5aef0ec7" {
		t.Fatalf("pinned profile contract = %s @ %s", profileOfficialVersion, profileOfficialCommit)
	}
	if got := profileListArgs(); !equalStrings(got, []string{"profile", "list"}) {
		t.Fatalf("profileListArgs() = %#v", got)
	}
	for _, arg := range profileListArgs() {
		if strings.Contains(arg, "json") || strings.HasPrefix(arg, "--format") {
			t.Fatalf("profile list argv must not request structured output: %#v", profileListArgs())
		}
	}
}

func TestProfileCreateArgsAreNoCloneNoAliasNoSkills(t *testing.T) {
	got := profileCreateArgs("coder")
	want := []string{"profile", "create", "coder", "--no-alias", "--no-skills"}
	if !equalStrings(got, want) {
		t.Fatalf("profileCreateArgs() = %#v, want %#v", got, want)
	}
	forbidden := []string{"--clone", "--clone-all", "--clone-from", "--description", "--alias", "--name", "--force", "-y", "--yes"}
	joined := strings.Join(got, " ")
	for _, flag := range forbidden {
		if strings.Contains(joined, flag) {
			t.Fatalf("create argv contains forbidden %q: %#v", flag, got)
		}
	}
}

func TestProfileDeleteArgsRequireYesAndNeverAcceptPath(t *testing.T) {
	got := profileDeleteArgs("coder")
	want := []string{"profile", "delete", "coder", "--yes"}
	if !equalStrings(got, want) {
		t.Fatalf("profileDeleteArgs() = %#v, want %#v", got, want)
	}
	if strings.Contains(got[2], `\`) || strings.Contains(got[2], `/`) || strings.Contains(got[2], "..") {
		t.Fatalf("delete argv native id looks like a path: %#v", got)
	}
}

func TestOfficialNormalizeAndValidateProfileName(t *testing.T) {
	normalized, err := officialNormalizeProfileName("  Librarian ")
	if err != nil || normalized != "librarian" {
		t.Fatalf("normalize Librarian = %q, %v", normalized, err)
	}
	normalized, err = officialNormalizeProfileName("Default")
	if err != nil || normalized != "default" {
		t.Fatalf("normalize Default = %q, %v", normalized, err)
	}
	if _, err := officialNormalizeProfileName("   "); !errors.Is(err, errProfileNameInvalid) {
		t.Fatalf("empty normalize error = %v", err)
	}

	if err := officialValidateProfileName("default"); err != nil {
		t.Fatalf("official validate default: %v", err)
	}
	for _, name := range []string{"coder", "work-bot", "a1", "my_agent", "1bot"} {
		if err := officialValidateProfileName(name); err != nil {
			t.Fatalf("official validate %q: %v", name, err)
		}
	}
	for _, name := range []string{"UPPER", "has space", ".hidden", "-leading", "hermes", "test", "tmp", "root", "sudo"} {
		if err := officialValidateProfileName(name); !errors.Is(err, errProfileNameInvalid) {
			t.Fatalf("official validate %q error = %v", name, err)
		}
	}
}

func TestYORVACreateProfileNameIsClosedOfficialSubset(t *testing.T) {
	for _, name := range []string{"coder", "work-bot", "a1", "my_agent"} {
		if err := validateYORVACreateProfileName(name); err != nil {
			t.Fatalf("create name %q: %v", name, err)
		}
	}
	// Official allows a leading digit; YORVA create does not.
	if err := officialValidateProfileName("1bot"); err != nil {
		t.Fatalf("official 1bot should be valid: %v", err)
	}
	if err := validateYORVACreateProfileName("1bot"); !errors.Is(err, errProfileNameInvalid) {
		t.Fatalf("YORVA create 1bot error = %v", err)
	}
	rejected := []string{
		"default", "hermes", "test", "tmp", "root", "sudo",
		"UPPER", "Coder", "has space", ".hidden", "-leading",
		"", "a/b", "..", strings.Repeat("x", 65),
	}
	for _, name := range rejected {
		if err := validateYORVACreateProfileName(name); !errors.Is(err, errProfileNameInvalid) {
			t.Fatalf("YORVA create %q error = %v", name, err)
		}
	}
	if got := utf8.RuneCountInString(strings.Repeat("a", officialProfileNameMaxLen)); got != 64 {
		t.Fatalf("max length pin = %d", officialProfileNameMaxLen)
	}
	if err := validateYORVACreateProfileName(strings.Repeat("a", 64)); err != nil {
		t.Fatalf("64-char create name: %v", err)
	}
}

func TestParseOfficialProfileListFixtures(t *testing.T) {
	defaultOnly := readProfileFixture(t, "list-default-only.txt")
	named := readProfileFixture(t, "list-default-and-named.txt")
	if defaultOnly != formatOfficialProfileList(t, []officialListRow{{Name: "default", Active: true}}) {
		t.Fatal("list-default-only.txt drifted from the pinned 0.20.2 table printer")
	}
	if named != formatOfficialProfileList(t, []officialListRow{
		{Name: "default", Model: "—"},
		{Name: "work", Model: "anthropic/claude-sonnet-4", Gateway: "stopped", Alias: "work", Active: true},
		{Name: "dev", Model: "—"},
	}) {
		t.Fatal("list-default-and-named.txt drifted from the pinned 0.20.2 table printer")
	}

	parsed, err := parseOfficialProfileList(defaultOnly)
	if err != nil || len(parsed.profiles) != 1 || parsed.profiles[0].name != "default" || !parsed.profiles[0].isDefault || !parsed.profiles[0].active {
		t.Fatalf("default-only = %#v, %v", parsed, err)
	}

	parsed, err = parseOfficialProfileList(named)
	if err != nil {
		t.Fatalf("named list: %v", err)
	}
	if len(parsed.profiles) != 3 {
		t.Fatalf("named list count = %d", len(parsed.profiles))
	}
	if parsed.profiles[0].name != "default" || !parsed.profiles[0].isDefault || parsed.profiles[0].active {
		t.Fatalf("default row = %#v", parsed.profiles[0])
	}
	if parsed.profiles[1].name != "work" || !parsed.profiles[1].active || parsed.profiles[1].isDefault {
		t.Fatalf("work row = %#v", parsed.profiles[1])
	}
	if parsed.profiles[2].name != "dev" || parsed.profiles[2].active {
		t.Fatalf("dev row = %#v", parsed.profiles[2])
	}

	empty, err := parseOfficialProfileList(readProfileFixture(t, "list-no-profiles.txt"))
	if err != nil || len(empty.profiles) != 0 {
		t.Fatalf("no-profiles = %#v, %v", empty, err)
	}
}

func TestParseOfficialProfileListRejectsUnknownOrUnsafeOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantErr error
	}{
		{name: "docs star format", output: readProfileFixture(t, "list-docs-star-format.txt"), wantErr: errProfileListUnrecognized},
		{name: "malformed header", output: readProfileFixture(t, "list-malformed-header.txt"), wantErr: errProfileListUnrecognized},
		{name: "truncated row", output: readProfileFixture(t, "list-truncated.txt"), wantErr: errProfileListUnrecognized},
		{name: "duplicate native id", output: readProfileFixture(t, "list-duplicate-names.txt"), wantErr: errProfileListDuplicateName},
		{name: "header only", output: officialProfileListHeader() + "\n" + officialProfileListSeparator() + "\n\n", wantErr: errProfileListEmptyTable},
		{name: "nul", output: officialProfileListHeader() + "\n\x00", wantErr: errProfileListUnrecognized},
		{name: "json object", output: `{"profiles":[{"name":"default"}]}` + "\n", wantErr: errProfileListUnrecognized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOfficialProfileList(test.output)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("parseOfficialProfileList() = %#v, %v, want %v", got, err, test.wantErr)
			}
		})
	}

	oversized := strings.Repeat("x", int(commandOutputLimit)+1)
	if _, err := parseOfficialProfileList(oversized); !errors.Is(err, errProfileListOversized) {
		t.Fatalf("oversized error = %v", err)
	}
}

func TestParseOfficialProfileListDoesNotTreatModelOrGatewayAsIdentity(t *testing.T) {
	output := formatOfficialProfileList(t, []officialListRow{
		{Name: "default", Model: "openai/gpt-4", Gateway: "running", Active: true},
		{Name: "coder", Model: "anthropic/claude-sonnet-4", Gateway: "stopped"},
	})
	parsed, err := parseOfficialProfileList(output)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.profiles[0].name != "default" || parsed.profiles[1].name != "coder" {
		t.Fatalf("identity columns leaked: %#v", parsed.profiles)
	}
}

type officialListRow struct {
	Name     string
	Model    string
	Gateway  string
	Alias    string
	Active   bool
	DistName string
	DistVer  string
}

func formatOfficialProfileList(t *testing.T, rows []officialListRow) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(officialProfileListHeader())
	b.WriteByte('\n')
	b.WriteString(officialProfileListSeparator())
	b.WriteByte('\n')
	for _, row := range rows {
		marker := "  "
		if row.Active {
			marker = " ◆"
		}
		model := row.Model
		if model == "" {
			model = "—"
		}
		if utf8.RuneCountInString(model) > 26 {
			model = string([]rune(model)[:26])
		}
		gw := row.Gateway
		if gw == "" {
			gw = "stopped"
		}
		alias := "—"
		if row.Name != "default" && row.Alias != "" {
			alias = row.Alias
		}
		dist := "—"
		if row.DistName != "" {
			dist = row.DistName + "@"
			if row.DistVer == "" {
				dist += "?"
			} else {
				dist += row.DistVer
			}
			if utf8.RuneCountInString(dist) > 30 {
				dist = string([]rune(dist)[:30])
			}
		}
		b.WriteString(marker)
		b.WriteString(padRight(row.Name, 15))
		b.WriteByte(' ')
		b.WriteString(padRight(model, 28))
		b.WriteByte(' ')
		b.WriteString(padRight(gw, 12))
		b.WriteByte(' ')
		b.WriteString(padRight(alias, 12))
		b.WriteByte(' ')
		b.WriteString(dist)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

func officialProfileListHeader() string {
	return "\n " + padRight("Profile", 16) + " " + padRight("Model", 28) + " " + padRight("Gateway", 12) + " " + padRight("Alias", 12) + " Distribution"
}

func officialProfileListSeparator() string {
	return " " + strings.Repeat("─", 15) + "    " + strings.Repeat("─", 27) + "    " + strings.Repeat("─", 11) + "    " + strings.Repeat("─", 11) + "    " + strings.Repeat("─", 20)
}

func padRight(value string, width int) string {
	count := utf8.RuneCountInString(value)
	if count >= width {
		return value
	}
	return value + strings.Repeat(" ", width-count)
}

func readProfileFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "profile", name))
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return string(data)
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
