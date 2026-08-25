package skillsmanagement

import (
	"errors"
	"regexp"
)

var (
	ErrSourceIDInvalid    = errors.New("Skill source ID is invalid")
	ErrSourceNotReviewed  = errors.New("Skill source is not reviewed for Hermes 0.20.5")
	reviewedSourcePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
)

// SourceDescriptor can only be constructed inside this package. All fields
// that could select a native registry or identifier remain unexported and
// therefore cannot be replaced by API/adapter callers.
type SourceDescriptor struct {
	id                 string
	displayName        string
	hermesSourceID     string
	hermesIdentifier   string
	reviewedProvenance string
}

func (d SourceDescriptor) ID() string                 { return d.id }
func (d SourceDescriptor) DisplayName() string        { return d.displayName }
func (d SourceDescriptor) HermesSourceID() string     { return d.hermesSourceID }
func (d SourceDescriptor) HermesIdentifier() string   { return d.hermesIdentifier }
func (d SourceDescriptor) ReviewedProvenance() string { return d.reviewedProvenance }

// No Hermes 0.20.5 install source has passed the Phase 7 closed-source and
// completed-postcondition qualification. Keep the compile-time allowlist empty
// until an Owner-reviewed descriptor is added in source and reviewed with tests.
var reviewedSources = [...]SourceDescriptor{}

func ReviewedSources() []SourceDescriptor {
	result := make([]SourceDescriptor, len(reviewedSources))
	copy(result, reviewedSources[:])
	return result
}

func LookupReviewedSource(id string) (SourceDescriptor, error) {
	if !reviewedSourcePattern.MatchString(id) {
		return SourceDescriptor{}, ErrSourceIDInvalid
	}
	for _, descriptor := range reviewedSources {
		if descriptor.id == id {
			return descriptor, nil
		}
	}
	return SourceDescriptor{}, ErrSourceNotReviewed
}
