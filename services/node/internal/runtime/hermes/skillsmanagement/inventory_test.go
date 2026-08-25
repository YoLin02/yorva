package skillsmanagement

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParseEnabledInventoryExact0205Fixture(t *testing.T) {
	body, err := os.ReadFile("testdata/v0.20.5/enabled-skills.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	scope, err := NewProfileScope("work")
	if err != nil {
		t.Fatalf("NewProfileScope() error = %v", err)
	}

	inventory, err := ParseEnabledInventory(scope, body)
	if err != nil {
		t.Fatalf("ParseEnabledInventory() error = %v", err)
	}
	if inventory.Scope.ProfileID() != "work" {
		t.Fatalf("Scope.ProfileID() = %q, want work", inventory.Scope.ProfileID())
	}
	if len(inventory.Skills) != 2 {
		t.Fatalf("len(Skills) = %d, want 2", len(inventory.Skills))
	}
	want := []EnabledSkill{
		{
			ID: "ascii-art", Description: "ASCII art generation", Category: "creative",
			Installed: InstalledStateInstalled, Enabled: EnabledStateEnabled, Scan: ScanStateUnknown,
		},
		{
			ID: "github", Description: "GitHub workflow skill", Category: "",
			Installed: InstalledStateInstalled, Enabled: EnabledStateEnabled, Scan: ScanStateUnknown,
		},
	}
	for index := range want {
		if inventory.Skills[index] != want[index] {
			t.Fatalf("Skills[%d] = %#v, want %#v", index, inventory.Skills[index], want[index])
		}
		projected := inventory.Skills[index].RuntimeSkill()
		if err := projected.Validate(); err != nil {
			t.Fatalf("Skills[%d].RuntimeSkill().Validate() error = %v", index, err)
		}
		if projected.SourceID != "" || projected.Version != "" || projected.UpdateAvailable {
			t.Fatalf("Skills[%d].RuntimeSkill() inferred unsupported fields: %#v", index, projected)
		}
	}
}

func TestParseEnabledInventoryAcceptsClosedBoundaries(t *testing.T) {
	scope := mustProfileScope(t, "default")
	body := fmt.Sprintf(
		`{"data":[{"category":%q,"description":%q,"name":%q}],"object":"list"}`,
		strings.Repeat("c", MaxSkillCategoryBytes),
		strings.Repeat("d", MaxSkillDescriptionBytes),
		strings.Repeat("n", MaxSkillNameBytes),
	)
	inventory, err := ParseEnabledInventory(scope, []byte(body))
	if err != nil {
		t.Fatalf("ParseEnabledInventory() error = %v", err)
	}
	if len(inventory.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1", len(inventory.Skills))
	}
}

func TestParseEnabledInventoryAcceptsOfficialMixedCaseAndReplacementText(t *testing.T) {
	scope := mustProfileScope(t, "default")
	body := []byte(`{"object":"list","data":[{"name":"My-Skill","description":"upstream replacement: \ufffd","category":"Creative-Tools"},{"name":"my-skill","description":"case-sensitive identity","category":null},{"name":"My.Skill_2","description":"core-valid punctuation","category":null}]}`)
	inventory, err := ParseEnabledInventory(scope, body)
	if err != nil {
		t.Fatalf("ParseEnabledInventory() error = %v", err)
	}
	if len(inventory.Skills) != 3 {
		t.Fatalf("len(Skills) = %d, want 3", len(inventory.Skills))
	}
	if inventory.Skills[0].ID != "My-Skill" || inventory.Skills[0].Category != "Creative-Tools" || !strings.Contains(inventory.Skills[0].Description, "�") {
		t.Fatalf("mixed-case/replacement projection = %#v", inventory.Skills[0])
	}
	for index := range inventory.Skills {
		if err := inventory.Skills[index].RuntimeSkill().Validate(); err != nil {
			t.Fatalf("Skills[%d].RuntimeSkill().Validate() error = %v", index, err)
		}
	}
}

func TestParseEnabledInventoryRejectsUnknownMissingWrongAndTrailingData(t *testing.T) {
	scope := mustProfileScope(t, "default")
	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "top level array", body: `[]`, want: ErrInventoryContractMismatch},
		{name: "unknown envelope field", body: `{"object":"list","data":[],"count":0}`, want: ErrInventoryContractMismatch},
		{name: "wrong envelope marker", body: `{"object":"collection","data":[]}`, want: ErrInventoryContractMismatch},
		{name: "missing marker", body: `{"data":[]}`, want: ErrInventoryContractMismatch},
		{name: "missing data", body: `{"object":"list"}`, want: ErrInventoryContractMismatch},
		{name: "null data", body: `{"object":"list","data":null}`, want: ErrInventoryContractMismatch},
		{name: "unknown item field", body: oneSkillBody(`"version":"1.0.0"`), want: ErrInventoryContractMismatch},
		{name: "missing name", body: `{"object":"list","data":[{"description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "missing description", body: `{"object":"list","data":[{"name":"skill","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "missing category", body: `{"object":"list","data":[{"name":"skill","description":"d"}]}`, want: ErrInventoryContractMismatch},
		{name: "numeric name", body: `{"object":"list","data":[{"name":1,"description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "object description", body: `{"object":"list","data":[{"name":"skill","description":{},"category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "numeric category", body: `{"object":"list","data":[{"name":"skill","description":"d","category":1}]}`, want: ErrInventoryContractMismatch},
		{name: "empty category string", body: `{"object":"list","data":[{"name":"skill","description":"d","category":""}]}`, want: ErrInventoryContractMismatch},
		{name: "whitespace name", body: `{"object":"list","data":[{"name":"My Skill","description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "leading whitespace name", body: `{"object":"list","data":[{"name":" My-Skill","description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "control name", body: `{"object":"list","data":[{"name":"My\u000aSkill","description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "path name", body: `{"object":"list","data":[{"name":"../skill","description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "backslash path name", body: `{"object":"list","data":[{"name":"category\\skill","description":"d","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "path category", body: `{"object":"list","data":[{"name":"skill","description":"d","category":"a/b"}]}`, want: ErrInventoryContractMismatch},
		{name: "whitespace category", body: `{"object":"list","data":[{"name":"skill","description":"d","category":"Creative Tools"}]}`, want: ErrInventoryContractMismatch},
		{name: "control category", body: `{"object":"list","data":[{"name":"skill","description":"d","category":"Creative\u0009Tools"}]}`, want: ErrInventoryContractMismatch},
		{name: "decoded null", body: `{"object":"list","data":[{"name":"skill","description":"a\u0000b","category":null}]}`, want: ErrInventoryContractMismatch},
		{name: "trailing object", body: `{"object":"list","data":[]} {}`, want: ErrInventoryContractMismatch},
		{name: "truncated", body: `{"object":"list","data":[`, want: ErrInventoryMalformed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseEnabledInventory(scope, []byte(test.body))
			if !errors.Is(err, test.want) {
				t.Fatalf("ParseEnabledInventory() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestParseEnabledInventoryRejectsDuplicateKeysAndIdentities(t *testing.T) {
	scope := mustProfileScope(t, "default")
	for _, body := range []string{
		`{"object":"list","object":"list","data":[]}`,
		`{"object":"list","data":[],"data":[]}`,
		`{"object":"list","data":[{"name":"skill","name":"other","description":"d","category":null}]}`,
		`{"object":"list","data":[{"name":"skill","description":"d","description":"other","category":null}]}`,
		`{"object":"list","data":[{"name":"skill","description":"d","category":null,"category":null}]}`,
		`{"object":"list","data":[{"name":"skill","description":"one","category":null},{"name":"skill","description":"two","category":"work"}]}`,
	} {
		if _, err := ParseEnabledInventory(scope, []byte(body)); !errors.Is(err, ErrInventoryDuplicate) {
			t.Fatalf("ParseEnabledInventory(%s) error = %v, want ErrInventoryDuplicate", body, err)
		}
	}
}

func TestParseEnabledInventoryEnforcesBodyCountAndStringBounds(t *testing.T) {
	scope := mustProfileScope(t, "default")
	tests := []struct {
		name string
		body []byte
	}{
		{name: "body", body: []byte(strings.Repeat(" ", MaxInventoryBodyBytes+1))},
		{name: "name", body: []byte(fmt.Sprintf(`{"object":"list","data":[{"name":%q,"description":"d","category":null}]}`, strings.Repeat("n", MaxSkillNameBytes+1)))},
		{name: "description", body: []byte(fmt.Sprintf(`{"object":"list","data":[{"name":"skill","description":%q,"category":null}]}`, strings.Repeat("d", MaxSkillDescriptionBytes+1)))},
		{name: "category", body: []byte(fmt.Sprintf(`{"object":"list","data":[{"name":"skill","description":"d","category":%q}]}`, strings.Repeat("c", MaxSkillCategoryBytes+1)))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseEnabledInventory(scope, test.body); !errors.Is(err, ErrInventoryTooLarge) {
				t.Fatalf("ParseEnabledInventory() error = %v, want ErrInventoryTooLarge", err)
			}
		})
	}

	items := make([]string, 0, MaxInventorySkills+1)
	for index := 0; index <= MaxInventorySkills; index++ {
		items = append(items, fmt.Sprintf(`{"name":"skill_%d","description":"d","category":null}`, index))
	}
	body := `{"object":"list","data":[` + strings.Join(items, ",") + `]}`
	if _, err := ParseEnabledInventory(scope, []byte(body)); !errors.Is(err, ErrInventoryTooLarge) {
		t.Fatalf("over-count ParseEnabledInventory() error = %v, want ErrInventoryTooLarge", err)
	}
}

func TestParseEnabledInventoryRejectsEmptyAndInvalidUTF8(t *testing.T) {
	scope := mustProfileScope(t, "default")
	for _, body := range [][]byte{nil, {}, {0xff}} {
		if _, err := ParseEnabledInventory(scope, body); !errors.Is(err, ErrInventoryMalformed) {
			t.Fatalf("ParseEnabledInventory(%v) error = %v, want ErrInventoryMalformed", body, err)
		}
	}
}

func oneSkillBody(extraField string) string {
	return `{"object":"list","data":[{"name":"skill","description":"d","category":null,` + extraField + `}]}`
}

func mustProfileScope(t *testing.T, profile string) ProfileScope {
	t.Helper()
	scope, err := NewProfileScope(profile)
	if err != nil {
		t.Fatalf("NewProfileScope(%q) error = %v", profile, err)
	}
	return scope
}
