package hermes

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	errProfileNameInvalid       = errors.New("invalid hermes profile name")
	errProfileListUnrecognized  = errors.New("hermes profile list output is unrecognized")
	errProfileListOversized     = errors.New("hermes profile list output exceeds the bound")
	errProfileListDuplicateName = errors.New("hermes profile list output contains a duplicate native identity")
	errProfileListEmptyTable    = errors.New("hermes profile list table has no profile rows")
)

const (
	officialListActiveMarker = '◆'
	officialListEmptyMessage = "No profiles found."
)

type parsedProfile struct {
	name      string
	isDefault bool
	active    bool
}

type parsedProfileList struct {
	profiles []parsedProfile
}

func parseOfficialProfileList(output string) (parsedProfileList, error) {
	if len(output) > int(commandOutputLimit) {
		return parsedProfileList{}, errProfileListOversized
	}
	if strings.ContainsRune(output, 0) {
		return parsedProfileList{}, errProfileListUnrecognized
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == officialListEmptyMessage {
		return parsedProfileList{}, nil
	}

	lines := splitProfileListLines(output)
	headerAt := -1
	for i, line := range lines {
		if isOfficialProfileListHeader(line) {
			headerAt = i
			break
		}
	}
	if headerAt < 0 {
		return parsedProfileList{}, errProfileListUnrecognized
	}
	if anyNonEmpty(lines[:headerAt]) {
		return parsedProfileList{}, errProfileListUnrecognized
	}

	separatorAt := nextNonEmpty(lines, headerAt+1)
	if separatorAt < 0 || !isOfficialProfileListSeparator(lines[separatorAt]) {
		return parsedProfileList{}, errProfileListUnrecognized
	}

	var parsed parsedProfileList
	seen := make(map[string]struct{})
	rowStart := nextNonEmpty(lines, separatorAt+1)
	if rowStart < 0 {
		return parsedProfileList{}, errProfileListEmptyTable
	}
	for i := rowStart; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			if anyNonEmpty(lines[i+1:]) {
				return parsedProfileList{}, errProfileListUnrecognized
			}
			break
		}
		row, err := parseOfficialProfileListRow(line)
		if err != nil {
			return parsedProfileList{}, err
		}
		if _, dup := seen[row.name]; dup {
			return parsedProfileList{}, errProfileListDuplicateName
		}
		seen[row.name] = struct{}{}
		parsed.profiles = append(parsed.profiles, row)
	}
	if len(parsed.profiles) == 0 {
		return parsedProfileList{}, errProfileListEmptyTable
	}
	return parsed, nil
}

func parseOfficialProfileListRow(line string) (parsedProfile, error) {
	runes := []rune(line)
	if len(runes) < 3 || runes[0] != ' ' {
		return parsedProfile{}, errProfileListUnrecognized
	}
	active := false
	switch runes[1] {
	case officialListActiveMarker:
		active = true
	case ' ':
	default:
		return parsedProfile{}, errProfileListUnrecognized
	}
	fields := strings.Fields(string(runes[2:]))
	// Official printer always emits name + model + gateway + alias + distribution.
	if len(fields) < 5 {
		return parsedProfile{}, errProfileListUnrecognized
	}
	name := fields[0]
	if fields[2] != "running" && fields[2] != "stopped" {
		return parsedProfile{}, errProfileListUnrecognized
	}
	if err := officialValidateListedProfileName(name); err != nil {
		return parsedProfile{}, err
	}
	return parsedProfile{
		name:      name,
		isDefault: name == "default",
		active:    active,
	}, nil
}

func officialValidateListedProfileName(name string) error {
	if name == "default" {
		return nil
	}
	if !officialProfileNamePattern.MatchString(name) {
		return errProfileListUnrecognized
	}
	if utf8.RuneCountInString(name) > officialProfileNameMaxLen {
		return errProfileListUnrecognized
	}
	return nil
}

func isOfficialProfileListHeader(line string) bool {
	fields := strings.Fields(line)
	return len(fields) == 5 &&
		fields[0] == "Profile" &&
		fields[1] == "Model" &&
		fields[2] == "Gateway" &&
		fields[3] == "Alias" &&
		fields[4] == "Distribution"
}

func isOfficialProfileListSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r != '─' && r != ' ' {
			return false
		}
	}
	return strings.ContainsRune(trimmed, '─')
}

func splitProfileListLines(output string) []string {
	normalized := strings.ReplaceAll(output, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func nextNonEmpty(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}

func anyNonEmpty(lines []string) bool {
	return nextNonEmpty(lines, 0) >= 0
}
