// Package driver defines core driver interfaces for XDB database backends.
package driver

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("xdb/driver: not found")
)

// SchemaReader is an interface for reading schemas.
type SchemaReader interface {
	GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)
	ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error)
}

// SchemaWriter is an interface for writing & deleting schemas.
type SchemaWriter interface {
	PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error
	DeleteSchema(ctx context.Context, uri *core.URI) error
}

// SchemaDriver is an interface for managing schemas.
type SchemaDriver interface {
	SchemaReader
	SchemaWriter
}

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*core.Tuple) error
	DeleteTuples(ctx context.Context, uris []*core.URI) error
}

// TupleDriver is an interface for managing tuples.
type TupleDriver interface {
	TupleReader
	TupleWriter
}

// RecordReader is an interface for reading records.
type RecordReader interface {
	GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error)
}

// RecordWriter is an interface for writing & deleting records.
type RecordWriter interface {
	PutRecords(ctx context.Context, records []*core.Record) error
	DeleteRecords(ctx context.Context, uris []*core.URI) error
}

// RecordDriver is an interface for managing records.
type RecordDriver interface {
	RecordReader
	RecordWriter
}
