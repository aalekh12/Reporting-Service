package querybuilder

import (
	"fmt"
	"regexp"
)

// identifierRe is the whitelist pattern every table/column/alias name must
// match. It is applied to values read from the (trusted) config tables as
// defense-in-depth, and would also reject anything that somehow originated
// from unsanitized user input.
var identifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validIdentifier(name string) error {
	if !identifierRe.MatchString(name) {
		return fmt.Errorf("invalid identifier %q", name)
	}
	return nil
}

// qualify renders "alias.column", validating both parts as identifiers.
func qualify(alias, column string) (string, error) {
	if err := validIdentifier(alias); err != nil {
		return "", err
	}
	if err := validIdentifier(column); err != nil {
		return "", err
	}
	return alias + "." + column, nil
}
