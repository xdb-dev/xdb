// Package xdbjson provides JSON <---> Record conversion utilities.
package xdbjson

import (
	"github.com/xdb-dev/xdb/types"
)

// FromRecord converts a record to JSON.
func FromRecord(record *types.Record) ([]byte, error) {
	return nil, nil
}

// ToRecord decodes JSON data into a record.
func ToRecord(data []byte, record *types.Record) error {
	return nil
}
