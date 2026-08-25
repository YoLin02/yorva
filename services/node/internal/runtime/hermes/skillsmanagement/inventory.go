package skillsmanagement

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"unicode"
	"unicode/utf8"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	MaxInventoryBodyBytes    = 512 * 1024
	MaxInventorySkills       = 256
	MaxSkillNameBytes        = 64
	MaxSkillDescriptionBytes = 1024
	MaxSkillCategoryBytes    = 64
)

var (
	ErrInventoryTooLarge         = errors.New("Hermes enabled Skill inventory exceeds its bound")
	ErrInventoryMalformed        = errors.New("Hermes enabled Skill inventory is malformed")
	ErrInventoryContractMismatch = errors.New("Hermes enabled Skill inventory contract is unknown")
	ErrInventoryDuplicate        = errors.New("Hermes enabled Skill inventory contains a duplicate")

	// Hermes 0.20.5 takes the frontmatter name as-is (up to 64
	// characters) and deduplicates it case-sensitively. Keep the subset that
	// can cross the Runtime-neutral Skill ID contract without normalization.
	skillNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

// EnabledSkill is the closed projection supported by Hermes 0.20.5
// authenticated GET /v1/skills. The endpoint does not prove provenance,
// version, update availability, or a scan verdict, so those fields are absent.
type EnabledSkill struct {
	ID          string
	Description string
	Category    string
	Installed   InstalledState
	Enabled     EnabledState
	Scan        ScanState
}

// RuntimeSkill projects only facts the enabled-only endpoint proves. Missing
// source/version/update evidence stays empty/false and scan remains UNKNOWN.
func (s EnabledSkill) RuntimeSkill() yorvaruntime.Skill {
	return yorvaruntime.Skill{
		ID:                s.ID,
		InstallationState: s.Installed,
		EnabledState:      s.Enabled,
		ScanState:         s.Scan,
	}
}

type EnabledInventory struct {
	Scope  ProfileScope
	Skills []EnabledSkill
}

// ParseEnabledInventory strictly parses the official Hermes 0.20.5 response:
// {"object":"list","data":[{"name", "description", "category"}, ...]}.
// Unknown, missing, repeated, oversized, or incorrectly typed fields fail
// closed. The result is enabled-only and never infers a scan verdict.
func ParseEnabledInventory(scope ProfileScope, body []byte) (EnabledInventory, error) {
	if err := scope.validate(); err != nil {
		return EnabledInventory{}, err
	}
	if len(body) == 0 || !utf8.Valid(body) {
		return EnabledInventory{}, ErrInventoryMalformed
	}
	if len(body) > MaxInventoryBodyBytes {
		return EnabledInventory{}, ErrInventoryTooLarge
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	inventory, err := parseInventoryObject(decoder, scope)
	if err != nil {
		return EnabledInventory{}, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return EnabledInventory{}, ErrInventoryContractMismatch
		}
		return EnabledInventory{}, ErrInventoryMalformed
	}
	return inventory, nil
}

func parseInventoryObject(decoder *json.Decoder, scope ProfileScope) (EnabledInventory, error) {
	if err := expectDelimiter(decoder, '{'); err != nil {
		return EnabledInventory{}, err
	}

	seen := make(map[string]struct{}, 2)
	objectSeen := false
	dataSeen := false
	var skills []EnabledSkill
	for decoder.More() {
		key, err := readObjectKey(decoder, seen)
		if err != nil {
			return EnabledInventory{}, err
		}
		switch key {
		case "object":
			value, err := readString(decoder)
			if err != nil {
				return EnabledInventory{}, err
			}
			if value != "list" {
				return EnabledInventory{}, ErrInventoryContractMismatch
			}
			objectSeen = true
		case "data":
			parsed, err := parseSkillArray(decoder)
			if err != nil {
				return EnabledInventory{}, err
			}
			skills = parsed
			dataSeen = true
		default:
			return EnabledInventory{}, ErrInventoryContractMismatch
		}
	}
	if err := expectDelimiter(decoder, '}'); err != nil {
		return EnabledInventory{}, err
	}
	if !objectSeen || !dataSeen {
		return EnabledInventory{}, ErrInventoryContractMismatch
	}
	return EnabledInventory{Scope: scope, Skills: skills}, nil
}

func parseSkillArray(decoder *json.Decoder) ([]EnabledSkill, error) {
	if err := expectDelimiter(decoder, '['); err != nil {
		return nil, err
	}

	skills := make([]EnabledSkill, 0)
	identities := make(map[string]struct{})
	for decoder.More() {
		if len(skills) == MaxInventorySkills {
			return nil, ErrInventoryTooLarge
		}
		skill, err := parseSkillObject(decoder)
		if err != nil {
			return nil, err
		}
		if _, exists := identities[skill.ID]; exists {
			return nil, ErrInventoryDuplicate
		}
		identities[skill.ID] = struct{}{}
		skills = append(skills, skill)
	}
	if err := expectDelimiter(decoder, ']'); err != nil {
		return nil, err
	}
	return skills, nil
}

func parseSkillObject(decoder *json.Decoder) (EnabledSkill, error) {
	if err := expectDelimiter(decoder, '{'); err != nil {
		return EnabledSkill{}, err
	}

	seen := make(map[string]struct{}, 3)
	var name, description, category string
	nameSeen := false
	descriptionSeen := false
	categorySeen := false
	for decoder.More() {
		key, err := readObjectKey(decoder, seen)
		if err != nil {
			return EnabledSkill{}, err
		}
		switch key {
		case "name":
			name, err = readString(decoder)
			if err != nil {
				return EnabledSkill{}, err
			}
			if !boundedString(name, MaxSkillNameBytes) {
				return EnabledSkill{}, ErrInventoryTooLarge
			}
			if !skillNamePattern.MatchString(name) || name == "." || name == ".." {
				return EnabledSkill{}, ErrInventoryContractMismatch
			}
			nameSeen = true
		case "description":
			description, err = readString(decoder)
			if err != nil {
				return EnabledSkill{}, err
			}
			if !boundedString(description, MaxSkillDescriptionBytes) {
				return EnabledSkill{}, ErrInventoryTooLarge
			}
			descriptionSeen = true
		case "category":
			var isNull bool
			category, isNull, err = readNullableString(decoder)
			if err != nil {
				return EnabledSkill{}, err
			}
			if !isNull && !boundedString(category, MaxSkillCategoryBytes) {
				return EnabledSkill{}, ErrInventoryTooLarge
			}
			if !isNull && !validInventoryCategory(category) {
				return EnabledSkill{}, ErrInventoryContractMismatch
			}
			categorySeen = true
		default:
			return EnabledSkill{}, ErrInventoryContractMismatch
		}
	}
	if err := expectDelimiter(decoder, '}'); err != nil {
		return EnabledSkill{}, err
	}
	if !nameSeen || !descriptionSeen || !categorySeen {
		return EnabledSkill{}, ErrInventoryContractMismatch
	}
	return EnabledSkill{
		ID:          name,
		Description: description,
		Category:    category,
		Installed:   InstalledStateInstalled,
		Enabled:     EnabledStateEnabled,
		Scan:        ScanStateUnknown,
	}, nil
}

func readObjectKey(decoder *json.Decoder, seen map[string]struct{}) (string, error) {
	key, err := readString(decoder)
	if err != nil {
		return "", err
	}
	if _, exists := seen[key]; exists {
		return "", ErrInventoryDuplicate
	}
	seen[key] = struct{}{}
	return key, nil
}

func readString(decoder *json.Decoder) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", ErrInventoryMalformed
	}
	value, ok := token.(string)
	if !ok || containsNUL(value) {
		return "", ErrInventoryContractMismatch
	}
	return value, nil
}

func readNullableString(decoder *json.Decoder) (string, bool, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", false, ErrInventoryMalformed
	}
	if token == nil {
		return "", true, nil
	}
	value, ok := token.(string)
	if !ok || containsNUL(value) {
		return "", false, ErrInventoryContractMismatch
	}
	return value, false, nil
}

func expectDelimiter(decoder *json.Decoder, expected json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return ErrInventoryMalformed
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != expected {
		return ErrInventoryContractMismatch
	}
	return nil
}

func boundedString(value string, maxBytes int) bool {
	return len(value) <= maxBytes && utf8.RuneCountInString(value) <= maxBytes
}

func containsNUL(value string) bool {
	for _, character := range value {
		if character == '\x00' {
			return true
		}
	}
	return false
}

func validInventoryCategory(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, character := range value {
		if character == '/' || character == '\\' || unicode.IsSpace(character) || unicode.IsControl(character) {
			return false
		}
	}
	return true
}
