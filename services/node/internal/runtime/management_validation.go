package runtime

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	managementIDMaxBytes      = 128
	managementSecretMaxBytes  = 16 * 1024
	managementTextMaxBytes    = 4 * 1024
	managementCollectionLimit = 256
)

var ErrInvalidManagementContract = errors.New("invalid Runtime management contract")

func validateManagementID(field, value string) error {
	if value == "" || len(value) > managementIDMaxBytes || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return fmt.Errorf("%w: invalid %s", ErrInvalidManagementContract, field)
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("%w: invalid %s", ErrInvalidManagementContract, field)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%w: invalid %s", ErrInvalidManagementContract, field)
	}
	return nil
}

func validateBoundedText(field, value string) error {
	if len(value) > managementTextMaxBytes || !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%w: invalid %s", ErrInvalidManagementContract, field)
	}
	return nil
}

func validateSecret(value []byte) error {
	if len(value) == 0 || len(value) > managementSecretMaxBytes {
		return fmt.Errorf("%w: invalid credential", ErrInvalidManagementContract)
	}
	return nil
}

func validateUniqueIDs(field string, values []string) error {
	if len(values) > managementCollectionLimit {
		return fmt.Errorf("%w: too many %s", ErrInvalidManagementContract, field)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateManagementID(field, value); err != nil {
			return err
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate %s", ErrInvalidManagementContract, field)
		}
		seen[value] = struct{}{}
	}
	return nil
}
