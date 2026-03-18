// Package validate provides input validation for CLI commands.
package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// URI validates and parses a raw URI string.
// It performs defense-in-depth checks on the raw string before parsing.
func URI(raw string) error {
	if strings.Contains(raw, " ") || strings.Contains(raw, "\t") {
		return fmt.Errorf("URI contains whitespace")
	}

	for i := range len(raw) {
		if raw[i] < 0x20 {
			return fmt.Errorf("URI contains control characters")
		}
	}

	if strings.Contains(raw, "?") {
		return fmt.Errorf("URI contains query parameters")
	}

	if strings.Contains(raw, "..") {
		return fmt.Errorf("URI contains path traversal sequence '..'")
	}

	if strings.Contains(raw, "%") {
		return fmt.Errorf("URI contains percent-encoding")
	}

	_, err := core.ParseURI(raw)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	return nil
}

// FilePath validates and canonicalizes a file path.
// Rejects paths outside CWD, symlinks that escape sandbox,
// and dangerous prefixes (/dev/, /proc/, /etc/, /sys/).
func FilePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	cleaned := filepath.Clean(abs)

	dangerousPrefixes := []string{"/dev/", "/proc/", "/etc/", "/sys/"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			return "", fmt.Errorf("dangerous path prefix: %s", prefix)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}

	cwdPrefix := cwd + string(filepath.Separator)
	if cleaned != cwd && !strings.HasPrefix(cleaned, cwdPrefix) {
		return "", fmt.Errorf("path %q is outside working directory %q", cleaned, cwd)
	}

	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return cleaned, nil
		}

		return "", fmt.Errorf("cannot evaluate symlinks: %w", err)
	}

	if resolved != cwd && !strings.HasPrefix(resolved, cwdPrefix) {
		return "", fmt.Errorf(
			"resolved path %q is outside working directory %q",
			resolved,
			cwd,
		)
	}

	return cleaned, nil
}

// Payload validates a JSON payload against size and depth limits.
func Payload(data []byte, maxSize int) error {
	if len(data) > maxSize {
		return fmt.Errorf("payload exceeds maximum size of %d bytes", maxSize)
	}

	if !json.Valid(data) {
		return fmt.Errorf("payload contains invalid JSON")
	}

	const maxDepth = 20

	depth := 0

	for _, b := range data {
		switch b {
		case '{', '[':
			depth++
			if depth > maxDepth {
				return fmt.Errorf(
					"payload exceeds maximum nesting depth of %d",
					maxDepth,
				)
			}
		case '}', ']':
			depth--
		}
	}

	return nil
}

// MutuallyExclusive checks that at most one of the named flags is set.
func MutuallyExclusive(flags map[string]bool) error {
	var set []string

	for name, active := range flags {
		if active {
			set = append(set, name)
		}
	}

	if len(set) > 1 {
		sort.Strings(set)

		names := make([]string, len(set))
		for i, name := range set {
			names[i] = "--" + name
		}

		return fmt.Errorf("specify only one of %s", strings.Join(names, ", "))
	}

	return nil
}
