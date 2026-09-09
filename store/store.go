package store

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Page is a paginated list of items.
type Page[T any] struct {
	Items      []T
	Total      int
	NextOffset int // 0 means no more pages
}

// Query holds scope, filtering, and pagination parameters for list operations.
type Query struct {
	URI    *core.URI // scope: ns-only or ns+schema
	Filter string
	Limit  int
	Offset int
}

// RecordStore combines read and write access for records.
type RecordStore interface {
	// GetRecord retrieves a single record by URI.
	// The URI must contain ns, schema, and id components.
	// Returns [core.ErrNotFound] if the record does not exist.
	GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error)

	// ListRecords lists records matching the given query.
	// Query.URI determines the scope: ns-only lists all records in the namespace,
	// ns+schema lists records for that schema.
	ListRecords(ctx context.Context, q *Query) (*Page[*core.Record], error)

	// CreateRecord creates a new record.
	// Returns [core.ErrAlreadyExists] if a record with the same URI exists.
	CreateRecord(ctx context.Context, record *core.Record) error

	// UpsertRecord creates or replaces a record (full-record put).
	UpsertRecord(ctx context.Context, record *core.Record) error

	// DeleteRecord deletes a record by URI.
	// Returns [core.ErrNotFound] if the record does not exist.
	DeleteRecord(ctx context.Context, uri *core.URI) error
}

// SchemaStore combines read and write access for schemas.
type SchemaStore interface {
	// GetSchema retrieves a schema definition by URI (ns + schema).
	// Returns [core.ErrNotFound] if the schema does not exist.
	GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)

	// ListSchemas lists schemas matching the given query.
	// Query.URI scopes the listing by namespace.
	ListSchemas(ctx context.Context, q *Query) (*Page[*schema.Def], error)

	// CreateSchema creates a new schema definition.
	// Returns [core.ErrAlreadyExists] if the schema already exists.
	CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error

	// UpdateSchema updates an existing schema definition.
	// Returns [core.ErrNotFound] if the schema does not exist.
	UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error

	// DeleteSchema deletes a schema by URI.
	// Returns [core.ErrNotFound] if the schema does not exist.
	DeleteSchema(ctx context.Context, uri *core.URI) error

	// DeleteSchemaRecords deletes all records belonging to a schema.
	// This includes dropping any backing tables or storage associated
	// with the schema's records. It is a no-op if no records exist.
	DeleteSchemaRecords(ctx context.Context, uri *core.URI) error
}

// NamespaceReader reads namespaces from the store.
// Namespaces are derived from schemas — there is no writer interface.
type NamespaceReader interface {
	// NamespaceExists reports whether the namespace holds any schema.
	// An absent namespace is (false, nil), not an error.
	NamespaceExists(ctx context.Context, uri *core.URI) (bool, error)

	// ListNamespaces lists all known namespace names.
	ListNamespaces(ctx context.Context, q *Query) (*Page[string], error)
}

// TupleStore is attr-level access on a store: every record attribute
// is addressable as xdb://ns/schema/id#attr. Writes are patches — a
// record springs into existence when its first tuples are put, and
// disappears when its last tuple is deleted.
type TupleStore interface {
	// GetTuple retrieves a single tuple by attr-level URI.
	// Returns [core.ErrNotFound] if the record or the attr is absent.
	GetTuple(ctx context.Context, uri *core.URI) (*core.Tuple, error)

	// GetTuples retrieves tuples by attr-level URIs. Absent attrs
	// are omitted — batch reads don't error on absence.
	GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error)

	// PutTuples patches tuples into their records, leaving other
	// attrs untouched. Tuples can span records.
	PutTuples(ctx context.Context, tuples ...*core.Tuple) error

	// DeleteTuples removes the tuples at the given attr-level URIs.
	// Idempotent: absent tuples are not an error. Removing a
	// record's last tuple removes the record.
	DeleteTuples(ctx context.Context, uris ...*core.URI) error
}

// Closer is implemented by stores that hold resources requiring cleanup.
type Closer interface {
	// Close releases any resources held by the store.
	Close() error
}

// Store is the full store interface consumed by the service layer.
type Store interface {
	Closer
	RecordStore
	SchemaStore
	NamespaceReader
	TupleStore
}

// HealthChecker is an optional interface stores can implement
// for health reporting. Stores that do not implement this
// interface are assumed healthy.
type HealthChecker interface {
	// Health returns nil if the store is healthy, or an error
	// describing the issue. Should complete quickly (< 1 second).
	Health(ctx context.Context) error
}

// TX is an optional interface for stores that support atomic
// transactions. Without it, the service layer runs each update's
// read and write-back sequentially, and batch.execute refuses to run
// unless the request sets non_atomic.
type TX interface {
	// Run executes fn within a transaction.
	// If fn returns an error, all changes are rolled back.
	Run(ctx context.Context, fn func(tx Store) error) error
}

// Validator is an optional interface for validating would-be writes
// against schema policy without writing. Facade-built stores always
// implement it.
type Validator interface {
	// ValidateRecord checks the record exactly as the write with the
	// given op would, performing no writes. Dynamic-mode evolution is
	// computed and discarded.
	ValidateRecord(ctx context.Context, record *core.Record, op Op) error

	// ValidateDeleteRecord checks a record or attr-level delete;
	// deleting a required attr is a schema violation.
	ValidateDeleteRecord(ctx context.Context, uri *core.URI) error
}
