package mcpmanagement

import (
	"errors"
	"regexp"
	"strings"
)

const maxProfileIDLength = 64

var (
	ErrProfileScopeInvalid = errors.New("Hermes MCP Profile scope is invalid")
	profileIDPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
)

// ProfileScope is an exact, already-normalized Hermes Profile identity. It
// deliberately does not trim, lowercase, or fall back to the default Profile.
type ProfileScope struct {
	profileID string
}

func NewProfileScope(profileID string) (ProfileScope, error) {
	if profileID == "default" {
		return ProfileScope{profileID: profileID}, nil
	}
	if profileID == "" || len(profileID) > maxProfileIDLength ||
		strings.TrimSpace(profileID) != profileID || !profileIDPattern.MatchString(profileID) ||
		reservedProfileID(profileID) {
		return ProfileScope{}, ErrProfileScopeInvalid
	}
	return ProfileScope{profileID: profileID}, nil
}

func (s ProfileScope) ProfileID() string { return s.profileID }

func (s ProfileScope) valid() bool {
	validated, err := NewProfileScope(s.profileID)
	return err == nil && validated.profileID == s.profileID
}

func reservedProfileID(value string) bool {
	switch value {
	case "hermes", "root", "sudo", "test", "tmp":
		return true
	default:
		return false
	}
}
