// Package xdbproto provides Protobuf <---> Record conversion utilities.
package xdbproto

import (
	"google.golang.org/protobuf/proto"

	"github.com/xdb-dev/xdb/core"
)

// FromRecord converts a record to a protobuf message.
func FromRecord(record *core.Record, m proto.Message) error {
	return nil
}

// ToRecord converts a protobuf message to a record.
func ToRecord(m proto.Message) (*core.Record, error) {
	return nil, nil
}
