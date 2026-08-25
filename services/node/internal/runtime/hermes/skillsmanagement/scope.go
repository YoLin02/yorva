package skillsmanagement

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxProfileIDLength = 64
	enabledSkillsPath  = "/v1/skills"
)

var (
	ErrProfileScopeInvalid = errors.New("Hermes Skill Profile scope is invalid")
	profileIDPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	reservedProfileIDs     = map[string]struct{}{
		"hermes": {},
		"root":   {},
		"sudo":   {},
		"test":   {},
		"tmp":    {},
	}
)

// ProfileScope is an exact Hermes Profile target. Its fields are private so a
// caller cannot combine a validated Profile ID with a different inventory path.
type ProfileScope struct {
	profileID string
}

// NewProfileScope validates an already-normalized Hermes Profile identity.
// It deliberately does not trim or lowercase caller input: ambient or
// corrected scope selection is unsafe for an authoritative inventory read.
func NewProfileScope(profileID string) (ProfileScope, error) {
	if profileID == "default" {
		return ProfileScope{profileID: profileID}, nil
	}
	if strings.TrimSpace(profileID) != profileID ||
		utf8.RuneCountInString(profileID) > maxProfileIDLength ||
		!profileIDPattern.MatchString(profileID) {
		return ProfileScope{}, ErrProfileScopeInvalid
	}
	if _, reserved := reservedProfileIDs[profileID]; reserved {
		return ProfileScope{}, ErrProfileScopeInvalid
	}
	return ProfileScope{profileID: profileID}, nil
}

func (s ProfileScope) ProfileID() string {
	return s.profileID
}

// EnabledInventoryPath returns the official Hermes 0.20.5 API-server route.
// Named Profiles use the documented multiplex prefix; the default Profile owns
// the unprefixed listener.
func (s ProfileScope) EnabledInventoryPath() (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	if s.profileID == "default" {
		return enabledSkillsPath, nil
	}
	return "/p/" + s.profileID + enabledSkillsPath, nil
}

func (s ProfileScope) validate() error {
	validated, err := NewProfileScope(s.profileID)
	if err != nil || validated.profileID != s.profileID {
		return ErrProfileScopeInvalid
	}
	return nil
}
