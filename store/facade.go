package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/schema"
)

// Option configures the store facade created by [New].
type Option func(*options)

type options struct {
	logger *slog.Logger
	cache  bool
}

// WithLogger enables write logging on the driver stack.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithSchemaCache caches schema definitions in memory, avoiding repeated
// driver reads during validation on non-transactional backends.
// Transactions bypass the cache to read their own schema changes and
// invalidate it on success. Writes on memory and SQLite backends therefore
// do not use the cache.
func WithSchemaCache() Option {
	return func(o *options) {
		o.cache = true
	}
}

// New creates a [Store] with versioning and schema enforcement over d.
// It panics if d is nil. Options enable write logging and schema caching.
//
// [TxDriver] and [QueryDriver] are detected on d before middleware is added.
// If d implements TxDriver, the returned Store also implements [TX] and
// runs writes in transactions. [Closer] and [HealthChecker] are checked on d
// when Close or Health is called.
func New(d Driver, opts ...Option) Store {
	if d == nil {
		panic("store: New requires a non-nil Driver")
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	f := &facade{
		raw:    d,
		logger: o.logger,
	}
	if txd, ok := d.(TxDriver); ok {
		f.txd = txd
	}
	if qd, ok := d.(QueryDriver); ok {
		f.qd = qd
	}

	stack := versioned(d)
	if o.cache {
		f.cache = newDefCache(stack)
		stack = f.cache
	}
	stack = enforce(stack)
	if o.logger != nil {
		stack = newLoggingDriver(o.logger, stack)
	}
	f.stack = stack

	if f.txd != nil {
		return &txStore{facade: f}
	}
	return f
}

// facade implements [Store] over a [Driver]: record verbs compile to
// mutations, records assemble from tuple scans, namespaces derive
// from schema scans, and transactions are orchestrated around the
// middleware stack.
type facade struct {
	stack  Driver         // logging(enforce(cache(versioned(raw))))
	raw    Driver         // capabilities detected here at New() time
	txd    TxDriver       // nil when unsupported
	qd     QueryDriver    // nil when unsupported
	cache  *cachingDriver // nil when disabled
	logger *slog.Logger   // for rebuilding the stack inside transactions
}

// withWriteTx runs fn against the enforced driver stack, inside a
// transaction when the driver supports one.
func (f *facade) withWriteTx(ctx context.Context, fn func(Driver) error) error {
	if f.txd == nil {
		return fn(f.stack)
	}

	return f.inTx(ctx, func(td Driver) error {
		return fn(f.wrapTx(td))
	})
}

// inTx runs fn inside a native transaction. Transactional writes
// bypass the schema cache, so it is invalidated after a commit.
func (f *facade) inTx(ctx context.Context, fn func(td Driver) error) error {
	err := f.txd.Tx(ctx, fn)
	if err == nil && f.cache != nil {
		f.cache.invalidateAll()
	}
	return err
}

// wrapTx rebuilds the middleware stack over a tx-scoped driver. The
// schema cache is deliberately skipped: a tx that writes a Def must
// read its own write, not a cached one.
func (f *facade) wrapTx(td Driver) Driver {
	stack := enforce(versioned(td))
	if f.logger != nil {
		stack = newLoggingDriver(f.logger, stack)
	}
	return stack
}

// Close releases driver resources, if the driver holds any.
func (f *facade) Close() error {
	if c, ok := f.raw.(Closer); ok {
		return c.Close()
	}
	return nil
}

// Health reports driver health. Drivers without a [HealthChecker] are
// assumed healthy.
func (f *facade) Health(ctx context.Context) error {
	if h, ok := f.raw.(HealthChecker); ok {
		return h.Health(ctx)
	}
	return nil
}

// --- Records ---

// GetRecord retrieves a record by URI, assembled from a tuple scan of
// its path. Returns [core.ErrNotFound] if the path holds no tuples.
func (f *facade) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	records, err := scanRecords(ctx, f.stack, uri)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, core.ErrNotFound
	}
	return records[0], nil
}

// ListRecords lists records matching the query. Drivers with native
// filter pushdown ([QueryDriver]) handle the query themselves; for
// the rest, the list is synthesized from a scan.
func (f *facade) ListRecords(
	ctx context.Context,
	q *Query,
) (*Page[*core.Record], error) {
	if f.qd != nil {
		page, err := f.queryRecords(ctx, q)
		if !errors.Is(err, ErrUnsupportedQuery) {
			return page, err
		}
		// Fall through to the synthesized path.
	}

	records, err := scanRecords(ctx, f.stack, q.URI)
	if err != nil {
		return nil, err
	}

	if q.Filter != "" {
		def, err := f.filterDef(ctx, q.URI)
		if err != nil {
			return nil, err
		}
		flt, err := filter.Compile(q.Filter, def)
		if err != nil {
			return nil, err
		}
		records, err = filter.Records(flt, records)
		if err != nil {
			return nil, err
		}
	}

	slices.SortFunc(records, func(a, b *core.Record) int {
		return strings.Compare(a.URI().Path(), b.URI().Path())
	})
	return Paginate(records, q), nil
}

// queryRecords delegates the query to the driver's pushdown and
// assembles the returned tuple pages into records.
func (f *facade) queryRecords(
	ctx context.Context,
	q *Query,
) (*Page[*core.Record], error) {
	page, err := f.qd.QueryTuples(ctx, q)
	if err != nil {
		return nil, err
	}

	items := make([]*core.Record, 0, len(page.Items))
	for _, tuples := range page.Items {
		if len(tuples) == 0 {
			continue
		}
		items = append(items,
			projectID(core.NewRecordFromTuples(tuples[0].Path(), tuples)),
		)
	}

	return &Page[*core.Record]{
		Items:      items,
		Total:      page.Total,
		NextOffset: page.NextOffset,
	}, nil
}

// filterDef fetches the schema definition for a schema-scoped query URI, for
// [filter.Compile] to type-check and strict-mode-validate against. A
// namespace-scoped query (no schema component) or a schema that no longer
// exists compiles with a nil def (schema-free, with dynamic typing).
func (f *facade) filterDef(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	if uri == nil || uri.Schema() == "" {
		return nil, nil
	}

	def, err := f.stack.GetSchema(ctx, uri.SchemaURI())
	if errors.Is(err, core.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return def, nil
}

// CreateRecord creates a new record.
func (f *facade) CreateRecord(ctx context.Context, record *core.Record) error {
	return f.applyRecord(ctx, record, OpCreate)
}

// UpsertRecord creates or replaces a record.
func (f *facade) UpsertRecord(ctx context.Context, record *core.Record) error {
	return f.applyRecord(ctx, record, OpPut)
}

// applyRecord compiles a record write into its mutation.
func (f *facade) applyRecord(
	ctx context.Context,
	record *core.Record,
	op Op,
) error {
	return f.withWriteTx(ctx, func(d Driver) error {
		return d.Apply(ctx, Mutation{
			Path:   record.URI(),
			Op:     op,
			Tuples: record.Tuples(),
		})
	})
}

// ValidateRecord checks the record against its schema exactly as the
// write with the given op would, without writing anything.
func (f *facade) ValidateRecord(
	ctx context.Context,
	record *core.Record,
	op Op,
) error {
	_, err := checkMutation(ctx, f.stack, Mutation{
		Path:   record.URI(),
		Op:     op,
		Tuples: record.Tuples(),
	})

	return err
}

// ValidateDeleteRecord checks a record or attr-level delete without
// deleting anything. Deleting a required attr is a schema violation.
func (f *facade) ValidateDeleteRecord(ctx context.Context, uri *core.URI) error {
	m := Mutation{
		Path: uri.RecordURI(),
		Op:   OpDelete,
	}
	if uri.Attr() != "" {
		m.Attrs = []string{uri.Attr()}
	}

	_, err := checkMutation(ctx, f.stack, m)

	return err
}

// DeleteRecord deletes a record by URI.
// Returns [core.ErrNotFound] if the record does not exist.
func (f *facade) DeleteRecord(ctx context.Context, uri *core.URI) error {
	return f.withWriteTx(ctx, func(d Driver) error {
		exists, err := recordExists(ctx, d, uri)
		if err != nil {
			return err
		}
		if !exists {
			return core.ErrNotFound
		}
		return d.Apply(ctx, Mutation{
			Path: uri,
			Op:   OpDelete,
		})
	})
}

// --- Tuples ---

// GetTuple retrieves a single tuple by attr-level URI.
// Returns [core.ErrNotFound] if the record or the attr is absent.
func (f *facade) GetTuple(ctx context.Context, uri *core.URI) (*core.Tuple, error) {
	if err := requireAttrURI(uri, "store: GetTuple requires an attr-level URI, got %s"); err != nil {
		return nil, err
	}

	tuples, err := f.stack.GetTuples(ctx, uri)
	if err != nil {
		return nil, err
	}
	if len(tuples) == 0 {
		return nil, core.ErrNotFound
	}
	return tuples[0], nil
}

// GetTuples retrieves tuples by attr-level URIs, omitting absences.
func (f *facade) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	return f.stack.GetTuples(ctx, uris...)
}

// PutTuples patches tuples into their records. Compiles to one
// [OpPatch] mutation per record path, in first-appearance order.
func (f *facade) PutTuples(ctx context.Context, tuples ...*core.Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	muts := groupByPath(tuples, OpPatch,
		func(t *core.Tuple) *core.URI { return t.Path() },
		func(m *Mutation, t *core.Tuple) { m.Tuples = append(m.Tuples, t) },
	)

	return f.withWriteTx(ctx, func(d Driver) error {
		return applyAll(ctx, d, muts)
	})
}

// DeleteTuples removes the tuples at the given attr-level URIs.
// Compiles to one [OpDelete] mutation per record path.
func (f *facade) DeleteTuples(ctx context.Context, uris ...*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	for _, uri := range uris {
		if err := requireAttrURI(uri,
			"store: DeleteTuples requires attr-level URIs, got %s (use DeleteRecord)",
		); err != nil {
			return err
		}
	}

	muts := groupByPath(uris, OpDelete,
		func(u *core.URI) *core.URI { return u.RecordURI() },
		func(m *Mutation, u *core.URI) { m.Attrs = append(m.Attrs, u.Attr()) },
	)

	return f.withWriteTx(ctx, func(d Driver) error {
		return applyAll(ctx, d, muts)
	})
}

// --- Schemas ---

// GetSchema retrieves a schema definition by URI.
func (f *facade) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	return f.stack.GetSchema(ctx, uri)
}

// ListSchemas lists schemas, scoped by namespace when the query URI
// is set.
func (f *facade) ListSchemas(
	ctx context.Context,
	q *Query,
) (*Page[*schema.Def], error) {
	var scope *core.URI
	if q != nil {
		scope = q.URI
	}

	var defs []*schema.Def
	for def, err := range f.stack.ScanSchemas(ctx, scope) {
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}

	slices.SortFunc(defs, func(a, b *schema.Def) int {
		return strings.Compare(a.URI.Path(), b.URI.Path())
	})
	return Paginate(defs, q), nil
}

// CreateSchema creates a new schema definition.
func (f *facade) CreateSchema(
	ctx context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	def.URI = uri
	return f.withWriteTx(ctx, func(d Driver) error {
		return d.CreateSchema(ctx, def)
	})
}

// UpdateSchema updates an existing schema definition.
func (f *facade) UpdateSchema(
	ctx context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	def.URI = uri
	return f.withWriteTx(ctx, func(d Driver) error {
		return d.PutSchema(ctx, def)
	})
}

// DeleteSchema deletes a schema by URI.
func (f *facade) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return f.withWriteTx(ctx, func(d Driver) error {
		return d.DeleteSchema(ctx, uri)
	})
}

// DeleteSchemaRecords deletes all records belonging to a schema.
func (f *facade) DeleteSchemaRecords(ctx context.Context, uri *core.URI) error {
	return f.withWriteTx(ctx, func(d Driver) error {
		return d.DropRecords(ctx, uri)
	})
}

// --- Namespaces ---

// GetNamespace checks if any schema exists in the given namespace.
func (f *facade) GetNamespace(ctx context.Context, uri *core.URI) (string, error) {
	for _, err := range f.stack.ScanSchemas(ctx, uri) {
		if err != nil {
			return "", err
		}
		return uri.NS(), nil
	}
	return "", core.ErrNotFound
}

// ListNamespaces lists unique namespaces derived from schemas.
func (f *facade) ListNamespaces(
	ctx context.Context,
	q *Query,
) (*Page[string], error) {
	seen := make(map[string]struct{})
	for def, err := range f.stack.ScanSchemas(ctx, nil) {
		if err != nil {
			return nil, err
		}
		seen[def.URI.NS()] = struct{}{}
	}

	items := slices.Sorted(maps.Keys(seen))
	return Paginate(items, q), nil
}

// --- TX ---

// txStore is the facade over a [TxDriver]; it additionally implements
// [TX].
type txStore struct {
	*facade
}

// Run executes fn within a transaction. The middleware stack is
// rebuilt over the tx-scoped driver so enforcement runs inside the
// transaction; the inner Store has no transaction capability of its
// own (no nesting).
func (t *txStore) Run(ctx context.Context, fn func(tx Store) error) error {
	return t.inTx(ctx, func(td Driver) error {
		inner := &facade{
			stack:  t.wrapTx(td),
			raw:    td,
			logger: t.logger,
		}
		return fn(inner)
	})
}

// --- Helpers ---

// requireAttrURI guards verbs that address tuples: the URI must carry
// an attr fragment (…#attr). format receives the offending URI.
func requireAttrURI(uri *core.URI, format string) error {
	if uri.Attr() == "" {
		return fmt.Errorf(format, uri)
	}
	return nil
}

// applyAll feeds mutations to the driver one at a time, stopping at
// the first failure and attributing it to its mutation. This is the
// only place [MutationError] is constructed.
func applyAll(ctx context.Context, d Driver, muts []Mutation) error {
	for i := range muts {
		if err := d.Apply(ctx, muts[i]); err != nil {
			return &MutationError{
				Index: i,
				Path:  muts[i].Path,
				Err:   err,
			}
		}
	}
	return nil
}

// recordExists reports whether the record path holds any tuples.
func recordExists(ctx context.Context, d Driver, path *core.URI) (bool, error) {
	for _, err := range d.ScanTuples(ctx, path) {
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// groupByPath compiles items into per-record mutations of one op, in
// first-appearance order. pathOf extracts each item's record path;
// add applies the item's contribution (a put tuple or a delete attr)
// to its mutation.
func groupByPath[T any](
	items []T,
	op Op,
	pathOf func(T) *core.URI,
	add func(m *Mutation, item T),
) []Mutation {
	var muts []Mutation
	byPath := make(map[string]int)
	for _, item := range items {
		path := pathOf(item)
		key := path.Path()
		i, ok := byPath[key]
		if !ok {
			i = len(muts)
			byPath[key] = i
			muts = append(muts, Mutation{Path: path, Op: op})
		}
		add(&muts[i], item)
	}
	return muts
}

// scanRecords scans a scope and groups the tuples into records,
// relying on the driver contract's per-record contiguity.
func scanRecords(
	ctx context.Context,
	d Driver,
	scope *core.URI,
) ([]*core.Record, error) {
	var records []*core.Record
	var current *core.Record

	for tuple, err := range d.ScanTuples(ctx, scope) {
		if err != nil {
			return nil, err
		}

		path := tuple.Path()
		if current == nil || current.URI().Path() != path.Path() {
			current = core.NewRecord(path.NS(), path.Schema(), path.ID())
			projectID(current)
			records = append(records, current)
		}
		current.Set(tuple.Attr(), tuple.Value())
	}

	return records, nil
}

// projectID sets the record's virtual [schema.FieldID]. It is never
// stored: every backend already holds the id as the record's addressing
// key, so it is projected from the path on the way out.
func projectID(record *core.Record) *core.Record {
	return record.Set(schema.FieldID, record.URI().ID())
}
