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
	// GetTuples returns the tuples at the given attr-level URIs
	// (point reads). Absent attrs are omitted — batch reads don't
	// error on absence; the facade maps absence to errors where
	// singular.
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
	// Apply executes one mutation atomically, returning bare sentinel
	// errors ([core.ErrAlreadyExists] for [OpCreate] on an existing
	// path). Batching, sequencing, and error attribution
	// ([MutationError]) are the facade's job; whole-batch atomicity is
	// orchestrated by the facade on [TxDriver]s.
	//
	// No validation, no policy — that is middleware's job. Semantics
	// per op are fixed by the driver conformance suite.
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

// SchemaWriter writes schema definitions verbatim and drops record-space.
//
// Definitions are stored verbatim: no Validate, no ValidateUpdate, no
// revision checks — middleware's job. The create/put split exists so
// exists-semantics stay data-driven, mirroring [OpCreate]/[OpPut].
type SchemaWriter interface {
	// CreateSchema stores a new definition. Returns
	// [core.ErrAlreadyExists] if one exists at def.URI. MUST be
	// atomic — no check-then-write.
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

// Driver is what a backend implements: pure storage, composed from the
// four storage roles. Note what is absent — [core.Record] never crosses
// this boundary. Records are assembled by the facade from tuple reads
// and compiled by the facade into mutations on writes.
//
// A Driver is not a [Store]: it performs no validation, no mode
// enforcement, no revision CAS. [New] is the only way to obtain a
// Store, and it installs the enforcement middleware unconditionally.
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
