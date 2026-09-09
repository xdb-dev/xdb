package xdbsqlite

import (
	"context"
	"database/sql"
	"iter"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// Driver is a SQLite-backed implementation of [store.Driver].
// Reads use the database without acquiring the driver's mutex. Writes hold
// the mutex and run in SQL transactions. [session] implements storage
// operations for both database and transaction handles.
type Driver struct {
	db *sql.DB
	mu sync.Mutex
}

// Option is reserved for future [Driver] configuration. No options
// exist yet.
type Option func(*Driver)

// NewDriver creates a new SQLite driver backed by the given [*sql.DB].
// The caller opens the connection; the driver takes ownership and
// closes it on Close.
func NewDriver(db *sql.DB, opts ...Option) (*Driver, error) {
	d := &Driver{db: db}
	for _, opt := range opts {
		opt(d)
	}

	if err := xsql.NewQueries(db).Bootstrap(context.Background()); err != nil {
		return nil, err
	}

	return d, nil
}

// Close closes the underlying database connection.
func (d *Driver) Close() error {
	return d.db.Close()
}

// Health pings the database.
func (d *Driver) Health(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// reader returns a lock-free session over the database.
func (d *Driver) reader() *session {
	return &session{q: xsql.NewQueries(d.db)}
}

// write serializes fn behind the mutex and runs it in one transaction,
// committing on success and rolling back on error. It is the single
// place transactions are opened for the whole driver.
func (d *Driver) write(ctx context.Context, fn func(s *session) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(&session{q: xsql.NewQueries(tx)}); err != nil {
		return err
	}

	return tx.Commit()
}

// --- Tuple reads ---

// GetTuples returns the tuples at the given attr-level URIs.
func (d *Driver) GetTuples(ctx context.Context, uris ...*core.URI) ([]*core.Tuple, error) {
	return d.reader().GetTuples(ctx, uris...)
}

// ScanTuples yields every tuple under scope, per-record contiguous.
func (d *Driver) ScanTuples(ctx context.Context, scope *core.URI) iter.Seq2[*core.Tuple, error] {
	return d.reader().ScanTuples(ctx, scope)
}

// --- Tuple writes ---

// Apply executes one mutation atomically in its own SQL transaction.
func (d *Driver) Apply(ctx context.Context, m store.Mutation) error {
	return d.write(ctx, func(s *session) error {
		return s.Apply(ctx, m)
	})
}

// --- Definition reads ---

// GetSchema retrieves a definition by URI. Returns [core.ErrNotFound] if absent.
func (d *Driver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	return d.reader().GetSchema(ctx, uri)
}

// ScanSchemas yields definitions under scope (nil = all, or a namespace URI).
func (d *Driver) ScanSchemas(ctx context.Context, scope *core.URI) iter.Seq2[*schema.Def, error] {
	return d.reader().ScanSchemas(ctx, scope)
}

// --- Definition writes ---

// CreateSchema stores a new definition and creates its backing storage.
func (d *Driver) CreateSchema(ctx context.Context, def *schema.Def) error {
	return d.write(ctx, func(s *session) error {
		return s.CreateSchema(ctx, def)
	})
}

// PutSchema stores a definition (upsert), creating or evolving its backing
// storage to match the field set.
func (d *Driver) PutSchema(ctx context.Context, def *schema.Def) error {
	return d.write(ctx, func(s *session) error {
		return s.PutSchema(ctx, def)
	})
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if absent.
func (d *Driver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return d.write(ctx, func(s *session) error {
		return s.DeleteSchema(ctx, uri)
	})
}

// DropRecords deletes all record tuples belonging to a schema.
func (d *Driver) DropRecords(ctx context.Context, uri *core.URI) error {
	return d.write(ctx, func(s *session) error {
		return s.DropRecords(ctx, uri)
	})
}

// --- TxDriver ---

// Tx executes fn against a transaction-scoped driver view. On error,
// all changes are rolled back.
func (d *Driver) Tx(ctx context.Context, fn func(tx store.Driver) error) error {
	return d.write(ctx, func(s *session) error {
		return fn(s)
	})
}

// --- QueryDriver ---

// QueryTuples pushes a schema-scoped query down to SQL. Namespace-
// scoped queries return [store.ErrUnsupportedQuery].
func (d *Driver) QueryTuples(ctx context.Context, q *store.Query) (*store.Page[[]*core.Tuple], error) {
	return d.reader().QueryTuples(ctx, q)
}
