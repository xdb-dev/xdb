// Package xdbjson provides JSON <---> Record conversion utilities.
package xdbjson

import (
	"github.com/xdb-dev/xdb/core"
)

// FromRecord converts a record to JSON.
func FromRecord(record *core.Record) ([]byte, error) {
	return nil, nil
}

// ToRecord decodes JSON data into a record.
func ToRecord(data []byte, record *core.Record) error {
	return nil
}
