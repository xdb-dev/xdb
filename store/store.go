// Package store defines core store interfaces for XDB database backends.
package store

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

var (
	// ErrSchemaModeChanged is returned when a schema mode is changed.
	ErrSchemaModeChanged = errors.New("[xdb/store] cannot change schema mode")

	// ErrFieldChangeType is returned when a field type is changed.
	ErrFieldChangeType = errors.New("[xdb/store] cannot change field type")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("[xdb/store] requested resource not found")
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

// SchemaStore is an interface for managing schemas.
type SchemaStore interface {
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

// TupleStore is an interface for managing tuples.
type TupleStore interface {
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

// RecordStore is an interface for managing records.
type RecordStore interface {
	RecordReader
	RecordWriter
}

// HealthChecker is an optional interface that stores can implement
// to provide health check capabilities. Stores that do not implement
// this interface are assumed to be healthy.
type HealthChecker interface {
	// Health returns nil if the store is healthy, or an error describing
	// the health issue. This method should complete quickly (< 1 second).
	Health(ctx context.Context) error
}
