package store

import (
	"context"
	"errors"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// TupleReader reads tuples from storage.
type TupleReader interface {
	// GetTuples returns tuples at the given attribute URIs. Absent
	// attributes are omitted. The facade reports missing attributes as
	// errors for single-tuple reads.
	GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error)

	// ScanTuples yields every tuple under scope: a namespace, a
	// schema, or a record path. A record read is a scan of one path.
	//
	// Tuples belonging to the same record are yielded contiguously;
	// attr order within a record is unspecified.
	ScanTuples(ctx context.Context, scope *core.URI) iter.Seq2[*core.Tuple, error]
}

// TupleWriter applies tuple mutations to storage.
type TupleWriter interface {
	// Apply executes one mutation atomically. It returns
	// [core.ErrAlreadyExists] for [OpCreate] on an existing record.
	// Drivers return errors without [MutationError] wrappers; the facade
	// adds batch sequencing and error attribution. Middleware handles
	// schema validation and version checks.
	Apply(ctx context.Context, m Mutation) error
}

// SchemaReader reads schema definitions, stored verbatim.
type SchemaReader interface {
	// GetSchema retrieves a definition by URI (ns + schema).
	// Returns [core.ErrNotFound] if absent.
	GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)

	// ScanSchemas yields definitions under scope: nil for all, or a
	// namespace URI.
	ScanSchemas(ctx context.Context, scope *core.URI) iter.Seq2[*schema.Def, error]
}

// SchemaWriter stores schema definitions verbatim and removes record data.
// Middleware validates definitions and checks revisions before writes.
type SchemaWriter interface {
	// CreateSchema stores a new definition or returns [core.ErrAlreadyExists]
	// if one exists at def.URI. The existence check and write must be atomic.
	CreateSchema(ctx context.Context, def *schema.Def) error

	// PutSchema stores a definition unconditionally (upsert). Used for
	// updates whose policy checks already ran in middleware, and for
	// dynamic-evolve write-backs.
	PutSchema(ctx context.Context, def *schema.Def) error

	// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if
	// absent.
	DeleteSchema(ctx context.Context, uri *core.URI) error

	// DropRecords deletes all record tuples belonging to a
	// schema (DROP TABLE / key sweep / scan+delete). No-op if no
	// records exist.
	DropRecords(ctx context.Context, uri *core.URI) error
}

// Driver is the storage contract implemented by a backend.
// The facade converts between [core.Record] values and tuple operations.
// Use [New] to add schema enforcement and versioning to a Driver.
type Driver interface {
	TupleReader
	TupleWriter
	SchemaReader
	SchemaWriter
}

// TxDriver is an optional capability for drivers with native
// transactions. fn receives a Driver scoped to the transaction; if fn
// returns an error, all changes are rolled back.
//
// The facade detects this once, on the raw driver, at [New] time.
type TxDriver interface {
	Tx(ctx context.Context, fn func(tx Driver) error) error
}

// ErrUnsupportedQuery is returned by [QueryDriver.QueryTuples] for
// queries the driver cannot push down (e.g. namespace-wide scopes).
// The facade falls back to synthesizing the list from ScanTuples.
var ErrUnsupportedQuery = errors.New("store: unsupported query")

// QueryDriver is an optional capability for drivers with native
// filter pushdown (e.g. compiling CEL to SQL). QueryTuples returns
// one page item per matching record: that record's full tuple set.
// Drivers without it get ListRecords synthesized from ScanTuples;
// drivers with it can decline individual queries by returning
// [ErrUnsupportedQuery].
type QueryDriver interface {
	QueryTuples(ctx context.Context, q *Query) (*Page[[]*core.Tuple], error)
}
