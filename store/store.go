package store

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("[xdb/store] not found")

	// ErrAlreadyExists is returned when creating a resource that already exists.
	ErrAlreadyExists = errors.New("[xdb/store] already exists")

	// ErrSchemaViolation is returned when data violates a schema constraint.
	ErrSchemaViolation = errors.New("[xdb/store] schema violation")
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
	Fields []string
	Limit  int
	Offset int
}

// RecordReader reads records from the store.
type RecordReader interface {
	// GetRecord retrieves a single record by URI.
	// The URI must contain ns, schema, and id components.
	// Returns [ErrNotFound] if the record does not exist.
	GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error)

	// ListRecords lists records matching the given query.
	// Query.URI determines the scope: ns-only lists all records in the namespace,
	// ns+schema lists records for that schema.
	ListRecords(ctx context.Context, q *Query) (*Page[*core.Record], error)
}

// RecordWriter writes records to the store.
type RecordWriter interface {
	// CreateRecord creates a new record.
	// Returns [ErrAlreadyExists] if a record with the same URI exists.
	CreateRecord(ctx context.Context, record *core.Record) error

	// UpdateRecord updates an existing record.
	// Returns [ErrNotFound] if the record does not exist.
	UpdateRecord(ctx context.Context, record *core.Record) error

	// UpsertRecord creates or updates a record.
	UpsertRecord(ctx context.Context, record *core.Record) error

	// DeleteRecord deletes a record by URI.
	// Returns [ErrNotFound] if the record does not exist.
	DeleteRecord(ctx context.Context, uri *core.URI) error
}

// RecordStore combines read and write access for records.
type RecordStore interface {
	RecordReader
	RecordWriter
}

// SchemaReader reads schema definitions from the store.
type SchemaReader interface {
	// GetSchema retrieves a schema definition by URI (ns + schema).
	// Returns [ErrNotFound] if the schema does not exist.
	GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error)

	// ListSchemas lists schemas matching the given query.
	// Query.URI scopes the listing by namespace.
	ListSchemas(ctx context.Context, q *Query) (*Page[*schema.Def], error)
}

// SchemaWriter writes schema definitions to the store.
type SchemaWriter interface {
	// CreateSchema creates a new schema definition.
	// Returns [ErrAlreadyExists] if the schema already exists.
	CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error

	// UpdateSchema updates an existing schema definition.
	// Returns [ErrNotFound] if the schema does not exist.
	UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error

	// DeleteSchema deletes a schema by URI.
	// Returns [ErrNotFound] if the schema does not exist.
	DeleteSchema(ctx context.Context, uri *core.URI) error
}

// SchemaStore combines read and write access for schemas.
type SchemaStore interface {
	SchemaReader
	SchemaWriter
}

// NamespaceReader reads namespaces from the store.
// Namespaces are derived from schemas — there is no writer interface.
type NamespaceReader interface {
	// GetNamespace retrieves namespace metadata by URI.
	// Returns [ErrNotFound] if the namespace does not exist.
	GetNamespace(ctx context.Context, uri *core.URI) (*core.NS, error)

	// ListNamespaces lists all known namespaces.
	ListNamespaces(ctx context.Context, q *Query) (*Page[*core.NS], error)
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
}

// HealthChecker is an optional interface stores can implement
// for health reporting. Stores that do not implement this
// interface are assumed healthy.
type HealthChecker interface {
	// Health returns nil if the store is healthy, or an error
	// describing the issue. Should complete quickly (< 1 second).
	Health(ctx context.Context) error
}

// BatchExecutor is an optional interface for stores that support
// atomic batch operations. The service layer falls back to
// sequential execution if the store does not implement this.
type BatchExecutor interface {
	// ExecuteBatch runs fn within a transaction.
	// If fn returns an error, all changes are rolled back.
	ExecuteBatch(ctx context.Context, fn func(tx Store) error) error
}
