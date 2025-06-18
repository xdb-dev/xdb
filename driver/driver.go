// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"

	"github.com/xdb-dev/xdb/types"
)

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*types.Tuple) error
	DeleteTuples(ctx context.Context, keys []*types.Key) error
}

// RecordReader is an interface for reading records.
type RecordReader interface {
	GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error)
}

// RecordWriter is an interface for writing & deleting records.
type RecordWriter interface {
	PutRecords(ctx context.Context, records []*types.Record) error
	DeleteRecords(ctx context.Context, keys []*types.Key) error
}
