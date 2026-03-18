// Package validate provides input validation for CLI commands.
package validate

import "fmt"

// URI validates and parses a raw URI string.
func URI(raw string) error {
	return fmt.Errorf("validate: URI not implemented")
}

// FilePath validates and canonicalizes a file path.
// Rejects paths outside CWD, symlinks that escape sandbox,
// and dangerous prefixes (/dev/, /proc/, /etc/).
func FilePath(path string) (string, error) {
	return "", fmt.Errorf("validate: FilePath not implemented")
}

// Payload validates a JSON payload against size and depth limits.
func Payload(data []byte, maxSize int) error {
	return fmt.Errorf("validate: Payload not implemented")
}

// MutuallyExclusive checks that at most one of the named flags is set.
func MutuallyExclusive(flags map[string]bool) error {
	return fmt.Errorf("validate: MutuallyExclusive not implemented")
}
