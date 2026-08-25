package skillsmanagement

import (
	"errors"
	"testing"
)

func TestReviewedSourceRegistryIsEmptyUntilQualification(t *testing.T) {
	if got := ReviewedSources(); len(got) != 0 {
		t.Fatalf("ReviewedSources() returned %d unqualified entries", len(got))
	}
	for _, id := range []string{"official", "clawhub", "github", "owner_skill"} {
		if _, err := LookupReviewedSource(id); !errors.Is(err, ErrSourceNotReviewed) {
			t.Fatalf("LookupReviewedSource(%q) error = %v, want ErrSourceNotReviewed", id, err)
		}
	}
}

func TestReviewedSourceRegistryRejectsCallerControlledLocatorForms(t *testing.T) {
	for _, id := range []string{
		"",
		" official",
		"OFFICIAL",
		"https://example.invalid/skill.zip",
		"../skill",
		"owner/repository",
		"C:\\skills\\skill",
	} {
		if _, err := LookupReviewedSource(id); !errors.Is(err, ErrSourceIDInvalid) {
			t.Fatalf("LookupReviewedSource(%q) error = %v, want ErrSourceIDInvalid", id, err)
		}
	}
}
