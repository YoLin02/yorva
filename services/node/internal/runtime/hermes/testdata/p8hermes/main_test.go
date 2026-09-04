package main

import "testing"

func TestValidProfileNameUsesClosedQualificationScope(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"default", "p8_soak_a", "profile-1"} {
		if !validProfileName(value) {
			t.Fatalf("validProfileName(%q) = false", value)
		}
	}
	for _, value := range []string{"", ".", "..", "1profile", "UPPER", `../escape`, `a\\b`, "a/b", "a:b"} {
		if validProfileName(value) {
			t.Fatalf("validProfileName(%q) = true", value)
		}
	}
}
