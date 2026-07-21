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
//
// Writes are serialized with an in-process mutex — SQLite is
// single-writer anyway — which makes the check-then-write op
// (OpCreate) atomic without relying on busy-timeout retries. Each
// mutation additionally runs in its own SQL transaction for crash
// atomicity.
type Driver struct {
	db *sql.DB
	mu sync.Mutex
}

// Option configures a [Driver].
type Option func(*Driver)

// NewDriver creates a new SQLite driver backed by the given [*sql.DB].
// The caller is responsible for opening the database connection; the
// driver takes ownership and closes it on Close.
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

// --- Tuple reads ---

// GetTuples returns the tuples at the given attr-level URIs.
// Absent attrs are omitted.
func (d *Driver) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	return getTuples(ctx, xsql.NewQueries(d.db), uris)
}

// ScanTuples yields every tuple under scope, per-record contiguous,
// ordered by table then record ID.
func (d *Driver) ScanTuples(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return scanTuples(ctx, xsql.NewQueries(d.db), scope)
}

// --- Tuple writes ---

// Apply executes one mutation atomically in its own SQL transaction.
func (d *Driver) Apply(ctx context.Context, m store.Mutation) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := applyMutation(ctx, xsql.NewQueries(tx), m); err != nil {
		return err
	}

	return tx.Commit()
}

// --- Definition reads ---

// GetSchema retrieves a definition by URI. Returns [core.ErrNotFound] if absent.
func (d *Driver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	def, err := getSchemaRaw(ctx, xsql.NewQueries(d.db), uri.NS(), uri.Schema())
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, core.ErrNotFound
	}
	return def, nil
}

// ScanSchemas yields definitions under scope (nil = all, or a namespace
// URI), ordered by namespace then schema.
func (d *Driver) ScanSchemas(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return scanSchemas(ctx, xsql.NewQueries(d.db), scope)
}

// --- Definition writes ---

// CreateSchema stores a new definition verbatim and creates its backing
// table. Returns [core.ErrAlreadyExists] if one exists.
func (d *Driver) CreateSchema(ctx context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := createSchema(ctx, xsql.NewQueries(tx), def); err != nil {
		return err
	}

	return tx.Commit()
}

// PutSchema stores a definition verbatim (upsert), creating or evolving
// its backing table to match the field set.
func (d *Driver) PutSchema(ctx context.Context, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := putSchema(ctx, xsql.NewQueries(tx), def); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if
// absent. Record data is left in place — [store.Driver] semantics
// separate record cleanup into DropRecords.
func (d *Driver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := deleteSchema(ctx, xsql.NewQueries(tx), uri); err != nil {
		return err
	}

	return tx.Commit()
}

// DropRecords deletes all record tuples belonging to a schema
// by clearing its backing tables. The definition is kept.
func (d *Driver) DropRecords(ctx context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := deleteSchemaRecords(ctx, xsql.NewQueries(tx), uri); err != nil {
		return err
	}

	return tx.Commit()
}

// --- TxDriver ---

// Tx executes fn against a transaction-scoped driver view. On error,
// all changes are rolled back.
func (d *Driver) Tx(ctx context.Context, fn func(tx store.Driver) error) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(&txDriver{q: xsql.NewQueries(tx)}); err != nil {
		return err
	}

	return tx.Commit()
}

// txDriver is a transaction-scoped view of Driver. The parent's
// mutex is already held; all methods delegate to the shared helpers
// over the transaction's queries.
type txDriver struct {
	q *xsql.Queries
}

func (tx *txDriver) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	return getTuples(ctx, tx.q, uris)
}

func (tx *txDriver) ScanTuples(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return scanTuples(ctx, tx.q, scope)
}

func (tx *txDriver) Apply(ctx context.Context, m store.Mutation) error {
	return applyMutation(ctx, tx.q, m)
}

func (tx *txDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	def, err := getSchemaRaw(ctx, tx.q, uri.NS(), uri.Schema())
	if err != nil {
		return nil, err
	}
	if def == nil {
		return nil, core.ErrNotFound
	}
	return def, nil
}

func (tx *txDriver) ScanSchemas(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return scanSchemas(ctx, tx.q, scope)
}

func (tx *txDriver) CreateSchema(ctx context.Context, def *schema.Def) error {
	return createSchema(ctx, tx.q, def)
}

func (tx *txDriver) PutSchema(ctx context.Context, def *schema.Def) error {
	return putSchema(ctx, tx.q, def)
}

func (tx *txDriver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return deleteSchema(ctx, tx.q, uri)
}

func (tx *txDriver) DropRecords(ctx context.Context, uri *core.URI) error {
	return deleteSchemaRecords(ctx, tx.q, uri)
}

// --- QueryDriver ---

// QueryTuples pushes a schema-scoped query down to SQL, compiling the
// CEL filter to a WHERE clause. Namespace-scoped queries return
// [store.ErrUnsupportedQuery] so the facade synthesizes them.
func (d *Driver) QueryTuples(
	ctx context.Context,
	q *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	return queryTuples(ctx, xsql.NewQueries(d.db), q)
}
