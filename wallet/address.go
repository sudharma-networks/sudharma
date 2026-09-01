package wallet

import (
	"fmt"
	"regexp"
	"strings"
)

const AddressLength = 40

var addressPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ValidateAddress checks the canonical Sudharma address representation.
func ValidateAddress(address string) error {
	if !addressPattern.MatchString(address) {
		return fmt.Errorf(
			"address must be exactly %d lowercase hexadecimal characters",
			AddressLength,
		)
	}
	return nil
}

// NormalizeAddress lowercases and validates a Sudharma address.
func NormalizeAddress(address string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(address))
	if err := ValidateAddress(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}
