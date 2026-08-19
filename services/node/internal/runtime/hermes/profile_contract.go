package hermes

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Pinned Hermes 0.20.2 Profile contract (official commit
// df4b65147d7ddd74dd449f9067aabbca5aef0ec7).
//
// D1 surface: documented Profile CLI from the active generation.
// REST /api/profiles requires an already-running Hermes web dashboard
// and is not on the public-path allowlist. TUI gateway exposes
// profiles.list/create/describe/configure/set_asset/get_asset and has
// no delete method. Phase 4 does not start a Hermes service.
// Official `hermes profile list` has no --json / structured offline
// output. The table printer in hermes_cli/main.py:cmd_profile is the
// exact-version list format; unknown output fails closed.

const (
	profileOfficialVersion = "0.20.2"
	profileOfficialCommit  = "df4b65147d7ddd74dd449f9067aabbca5aef0ec7"

	profileListAction   = "list"
	profileCreateAction = "create"
	profileDeleteAction = "delete"

	profileNoAliasFlag  = "--no-alias"
	profileNoSkillsFlag = "--no-skills"
	profileYesFlag      = "--yes"

	// Official on-disk id: first [a-z0-9], then up to 63 [a-z0-9_-]
	// (total 1–64). From hermes_cli/profiles.py _PROFILE_ID_RE.
	officialProfileNameMaxLen = 64
)

// Official reserved names from hermes_cli/profiles.py _RESERVED_NAMES.
// validate_profile_name allows "default" via an early return; create and
// delete still reject it. YORVA create ingress rejects the whole set.
var officialReservedProfileNames = map[string]struct{}{
	"hermes":  {},
	"default": {},
	"test":    {},
	"tmp":     {},
	"root":    {},
	"sudo":    {},
}

var (
	// Official on-disk identity. Directory names already stored on disk
	// may start with a digit; YORVA create uses a closed subset.
	officialProfileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	// YORVA create ingress: official grammar closed to a leading letter.
	yorvaCreateProfileNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
)

func profileListArgs() []string {
	return []string{"profile", profileListAction}
}

func profileCreateArgs(name string) []string {
	return []string{"profile", profileCreateAction, name, profileNoAliasFlag, profileNoSkillsFlag}
}

func profileDeleteArgs(nativeID string) []string {
	return []string{"profile", profileDeleteAction, nativeID, profileYesFlag}
}

func officialNormalizeProfileName(name string) (string, error) {
	stripped := strings.TrimSpace(name)
	if stripped == "" {
		return "", fmt.Errorf("%w: empty", errProfileNameInvalid)
	}
	if strings.EqualFold(stripped, "default") {
		return "default", nil
	}
	return strings.ToLower(stripped), nil
}

func officialValidateProfileName(name string) error {
	if name == "default" {
		return nil
	}
	if !officialProfileNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %q must match [a-z0-9][a-z0-9_-]{0,63}", errProfileNameInvalid, name)
	}
	if _, reserved := officialReservedProfileNames[name]; reserved {
		return fmt.Errorf("%w: %q is reserved", errProfileNameInvalid, name)
	}
	if utf8.RuneCountInString(name) > officialProfileNameMaxLen {
		return fmt.Errorf("%w: %q exceeds %d characters", errProfileNameInvalid, name, officialProfileNameMaxLen)
	}
	return nil
}

func validateYORVACreateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", errProfileNameInvalid)
	}
	if !yorvaCreateProfileNamePattern.MatchString(name) {
		return fmt.Errorf("%w: %q must match [a-z][a-z0-9_-]{0,63}", errProfileNameInvalid, name)
	}
	if _, reserved := officialReservedProfileNames[name]; reserved {
		return fmt.Errorf("%w: %q is reserved", errProfileNameInvalid, name)
	}
	return nil
}
