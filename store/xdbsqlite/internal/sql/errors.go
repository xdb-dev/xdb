package sql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// ErrNoTable reports that a per-schema backing table does not exist.
// Query methods translate SQLite's "no such table" error into this
// sentinel so callers check with [errors.Is] instead of matching on
// the driver's error string.
var ErrNoTable = errors.New("sqlite: no such table")

// mapErr translates SQLite driver errors into the store's domain
// sentinels. "no such table" becomes [ErrNoTable]; a "UNIQUE constraint
// failed" becomes [core.ErrUniqueViolation], naming the offending field.
// Every other error passes through unchanged. Both messages are emitted
// by SQLite itself and are stable across Go drivers, so matching them
// here — in one place — keeps the string match out of the callers.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "no such table") {
		return ErrNoTable
	}
	if strings.Contains(msg, "UNIQUE constraint failed") {
		if field := uniqueField(msg); field != "" {
			return fmt.Errorf("%w: field %q", core.ErrUniqueViolation, field)
		}
		return fmt.Errorf("%w", core.ErrUniqueViolation)
	}
	return err
}

// uniqueField extracts the field (column) name from SQLite's
// "UNIQUE constraint failed: <table>.<column>" message. It returns the
// text after the final "." with surrounding quotes trimmed, or "" when
// the message does not follow that shape.
func uniqueField(msg string) string {
	_, detail, ok := strings.Cut(msg, "UNIQUE constraint failed:")
	if !ok {
		return ""
	}
	detail = strings.TrimSpace(detail)
	// A composite index lists "t.a, t.b"; take the first column.
	if comma := strings.IndexByte(detail, ','); comma >= 0 {
		detail = detail[:comma]
	}
	if dot := strings.LastIndexByte(detail, '.'); dot >= 0 {
		detail = detail[dot+1:]
	}
	return strings.Trim(strings.TrimSpace(detail), `"`)
}
