// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"

	"github.com/xdb-dev/xdb/types"
)

// SchemaReader is an interface for reading schemas.
type SchemaReader interface {
	GetAllSchemas(ctx context.Context) ([]*types.Schema, error)
	GetSchema(ctx context.Context, key *types.Key) (*types.Schema, error)
}

// SchemaWriter is an interface for writing & deleting schemas.

type SchemaWriter interface {
	CreateSchema(ctx context.Context, schema *types.Schema) error
	UpdateSchema(ctx context.Context, schema *types.Schema) error
}

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
