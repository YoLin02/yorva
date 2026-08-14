package hermes

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const supportedRange = ">=0.19.0 <0.21.0"

var versionBannerPattern = regexp.MustCompile(`(?im)^\s*Hermes(?:\s+Agent)?\s+v?([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z.-]+)?(?:\s|\(|$)`)

type version struct {
	major      uint64
	minor      uint64
	patch      uint64
	prerelease string
}

func parseVersionBanner(output string) (version, error) {
	matches := versionBannerPattern.FindAllStringSubmatch(output, -1)
	if len(matches) != 1 {
		return version{}, errors.New("version banner must contain exactly one Hermes package version")
	}

	major, err := parseVersionNumber(matches[0][1])
	if err != nil {
		return version{}, fmt.Errorf("parse major version: %w", err)
	}
	minor, err := parseVersionNumber(matches[0][2])
	if err != nil {
		return version{}, fmt.Errorf("parse minor version: %w", err)
	}
	patch, err := parseVersionNumber(matches[0][3])
	if err != nil {
		return version{}, fmt.Errorf("parse patch version: %w", err)
	}
	prerelease := strings.TrimPrefix(matches[0][4], "-")
	if prerelease != "" && !validPrerelease(prerelease) {
		return version{}, errors.New("invalid semantic-version prerelease")
	}
	return version{major: major, minor: minor, patch: patch, prerelease: prerelease}, nil
}

func parseVersionNumber(value string) (uint64, error) {
	if len(value) > 1 && value[0] == '0' {
		return 0, errors.New("numeric version identifiers cannot contain leading zeroes")
	}
	number, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, errors.New("numeric version identifier is out of range")
	}
	return number, nil
}

func validPrerelease(value string) bool {
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		numeric := true
		for _, character := range identifier {
			if (character < '0' || character > '9') &&
				(character < 'A' || character > 'Z') &&
				(character < 'a' || character > 'z') && character != '-' {
				return false
			}
			if character < '0' || character > '9' {
				numeric = false
			}
		}
		if numeric && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func (v version) String() string {
	value := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.prerelease != "" {
		value += "-" + v.prerelease
	}
	return value
}

func (v version) supported() bool {
	return v.major == 0 && (v.minor == 19 || v.minor == 20) && v.prerelease == ""
}
