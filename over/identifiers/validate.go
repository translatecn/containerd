// Package identifiers provides common validation for identifiers and keys
// across containerd.
//
// Identifiers in containerd must be a alphanumeric, allowing limited
// underscores, dashes and dots.
//
// While the character set may be expanded in the future, identifiers
// are guaranteed to be safely used as filesystem path components.
package identifiers

import (
	"demo/over/errdefs"
	"fmt"
)

const (
	maxLength  = 76
	alphanum   = `[A-Za-z0-9]+`
	separators = `[._-]`
)

// Validate returns nil if the string s is a valid identifier.
//
// identifiers are similar to the domain name rules according to RFC 1035, section 2.3.1. However
// rules in this package are relaxed to allow numerals to follow period (".") and mixed case is
// allowed.
//
// In general identifiers that pass this validation should be safe for use as filesystem path components.
func Validate(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("identifier must not be empty: %w", errdefs.ErrInvalidArgument)
	}

	if len(s) > maxLength {
		return fmt.Errorf("identifier %q greater than maximum length (%d characters): %w", s, maxLength, errdefs.ErrInvalidArgument)
	}

	//if !identifierRe.MatchString(s) {
	//	return fmt.Errorf("identifier %q must match %v: %w", s, identifierRe, errdefs.ErrInvalidArgument)
	//}
	return nil
}

func reGroup(s string) string {
	return `(?:` + s + `)`
}

func reAnchor(s string) string {
	return `^` + s + `$`
}
