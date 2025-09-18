// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"

	"github.com/xdb-dev/xdb/core"
)

// SchemaReader is an interface for reading schemas.
type SchemaReader interface {
	GetAllSchemas(ctx context.Context) ([]*core.Schema, error)
	GetSchema(ctx context.Context, key *core.Key) (*core.Schema, error)
}

// SchemaWriter is an interface for writing & deleting schemas.

type SchemaWriter interface {
	CreateSchema(ctx context.Context, schema *core.Schema) error
	UpdateSchema(ctx context.Context, schema *core.Schema) error
}

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*core.Tuple) error
	DeleteTuples(ctx context.Context, keys []*core.Key) error
}

// RecordReader is an interface for reading records.
type RecordReader interface {
	GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error)
}

// RecordWriter is an interface for writing & deleting records.
type RecordWriter interface {
	PutRecords(ctx context.Context, records []*core.Record) error
	DeleteRecords(ctx context.Context, keys []*core.Key) error
}
