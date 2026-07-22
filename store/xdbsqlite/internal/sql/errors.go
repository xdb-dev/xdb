package sql

import (
	"errors"
	"strings"
)

// ErrNoTable reports that a per-schema backing table does not exist.
// Query methods translate SQLite's "no such table" error into this
// sentinel so callers check with [errors.Is] instead of matching on
// the driver's error string.
var ErrNoTable = errors.New("sqlite: no such table")

// mapErr translates SQLite's missing-table error into [ErrNoTable].
// Every other error passes through unchanged. The "no such table"
// text is emitted by SQLite itself and is stable across Go drivers,
// so matching it here — in one place — keeps the string match out of
// the callers.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "no such table") {
		return ErrNoTable
	}
	return err
}
