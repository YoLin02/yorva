package skillsmanagement

import (
	"errors"
	"strings"
	"testing"
)

func TestProfileScopeBuildsExactInventoryPaths(t *testing.T) {
	tests := []struct {
		profile string
		path    string
	}{
		{profile: "default", path: "/v1/skills"},
		{profile: "work_2", path: "/p/work_2/v1/skills"},
		{profile: "2nd-profile", path: "/p/2nd-profile/v1/skills"},
	}
	for _, test := range tests {
		t.Run(test.profile, func(t *testing.T) {
			scope, err := NewProfileScope(test.profile)
			if err != nil {
				t.Fatalf("NewProfileScope() error = %v", err)
			}
			if scope.ProfileID() != test.profile {
				t.Fatalf("ProfileID() = %q, want %q", scope.ProfileID(), test.profile)
			}
			path, err := scope.EnabledInventoryPath()
			if err != nil {
				t.Fatalf("EnabledInventoryPath() error = %v", err)
			}
			if path != test.path {
				t.Fatalf("EnabledInventoryPath() = %q, want %q", path, test.path)
			}
		})
	}
}

func TestProfileScopeRejectsAmbientOrUnsafeIdentity(t *testing.T) {
	invalid := []string{
		"",
		"Default",
		" named",
		"named ",
		"UPPER",
		"../other",
		"a/b",
		"root",
		"sudo",
		"test",
		"tmp",
		"hermes",
		strings.Repeat("a", maxProfileIDLength+1),
	}
	for _, profile := range invalid {
		t.Run(profile, func(t *testing.T) {
			_, err := NewProfileScope(profile)
			if !errors.Is(err, ErrProfileScopeInvalid) {
				t.Fatalf("NewProfileScope(%q) error = %v, want ErrProfileScopeInvalid", profile, err)
			}
		})
	}
}

func TestZeroProfileScopeCannotParseOrBuildPath(t *testing.T) {
	var scope ProfileScope
	if _, err := scope.EnabledInventoryPath(); !errors.Is(err, ErrProfileScopeInvalid) {
		t.Fatalf("EnabledInventoryPath() error = %v, want ErrProfileScopeInvalid", err)
	}
	if _, err := ParseEnabledInventory(scope, []byte(`{"object":"list","data":[]}`)); !errors.Is(err, ErrProfileScopeInvalid) {
		t.Fatalf("ParseEnabledInventory() error = %v, want ErrProfileScopeInvalid", err)
	}
}
