package xdbsqlite

import (
	"context"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
	"github.com/xdb-dev/xdb/x"
)

// session is the full [store.Driver] implementation bound to one query
// handle — a database for lock-free reads, or a transaction for
// writes. The public Driver owns resources and atomicity; session owns
// the routing. Every method is identical inside and outside a
// transaction because the query handle is the only thing that differs.
type session struct {
	q *xsql.Queries
}

// --- Tuple reads ---

// GetTuples resolves attr-level point reads, omitting absences and
// preserving request order. Each record and each def is read at most
// once per call.
func (s *session) GetTuples(
	ctx context.Context,
	uris ...*core.URI,
) ([]*core.Tuple, error) {
	defs := make(map[string]*schema.Def)
	records := make(map[string]map[string]*core.Tuple)

	var got []*core.Tuple
	for _, uri := range uris {
		recordPath := uri.RecordPath()

		attrs, ok := records[recordPath]
		if !ok {
			schemaPath := uri.SchemaURI().Path()
			def, loaded := defs[schemaPath]
			if !loaded {
				var err error
				def, err = getSchemaRaw(ctx, s.q, uri.NS(), uri.Schema())
				if err != nil {
					return nil, err
				}
				defs[schemaPath] = def
			}

			tuples, err := engineFor(s.q, def).readRecord(ctx, uri.RecordURI())
			if err != nil {
				return nil, err
			}
			attrs = x.Index(tuples, (*core.Tuple).Attr)
			records[recordPath] = attrs
		}

		if tuple, ok := attrs[uri.Attr()]; ok {
			got = append(got, tuple)
		}
	}

	return got, nil
}

// ScanTuples yields every tuple under scope. Records are contiguous:
// each schema is paged in _id order, one record's tuples yielded
// together.
func (s *session) ScanTuples(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return func(yield func(*core.Tuple, error) bool) {
		if scope != nil && scope.ID() != "" {
			s.scanOneRecord(ctx, scope, yield)
			return
		}

		targets, err := scanTargets(ctx, s.q, scope)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, tgt := range targets {
			eng := engineFor(s.q, tgt.def)
			for tuples, err := range eng.scanRecords(ctx, tgt.uri) {
				if err != nil {
					yield(nil, err)
					return
				}
				for _, tuple := range tuples {
					if !yield(tuple, nil) {
						return
					}
				}
			}
		}
	}
}

// scanOneRecord yields the tuples of a single record path.
func (s *session) scanOneRecord(
	ctx context.Context,
	path *core.URI,
	yield func(*core.Tuple, error) bool,
) {
	def, err := getSchemaRaw(ctx, s.q, path.NS(), path.Schema())
	if err != nil {
		yield(nil, err)
		return
	}

	tuples, err := engineFor(s.q, def).readRecord(ctx, path.RecordURI())
	if err != nil {
		yield(nil, err)
		return
	}

	for _, tuple := range tuples {
		if !yield(tuple, nil) {
			return
		}
	}
}

// --- Tuple writes ---

// Apply routes one mutation to the engine chosen by its schema's def
// and executes the op there.
func (s *session) Apply(ctx context.Context, m store.Mutation) error {
	def, err := getSchemaRaw(ctx, s.q, m.Path.NS(), m.Path.Schema())
	if err != nil {
		return err
	}
	return runMutation(ctx, engineFor(s.q, def), m)
}

// --- Definition reads ---

// GetSchema retrieves a definition by URI. Returns [core.ErrNotFound] if absent.
func (s *session) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	def, err := getSchemaRaw(ctx, s.q, uri.NS(), uri.Schema())
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
func (s *session) ScanSchemas(
	ctx context.Context,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return scanSchemas(ctx, s.q, scope)
}

// --- Definition writes ---

// CreateSchema stores a new definition verbatim and creates its backing
// storage. Returns [core.ErrAlreadyExists] if one exists.
func (s *session) CreateSchema(ctx context.Context, def *schema.Def) error {
	exists, err := s.q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: def.URI.NS(),
		Schema:    def.URI.Schema(),
	})
	if err != nil {
		return err
	}
	if exists {
		return core.ErrAlreadyExists
	}

	if err := putSchemaRow(ctx, s.q, def); err != nil {
		return err
	}

	return engineFor(s.q, def).ensure(ctx, def.URI)
}

// PutSchema stores a definition verbatim (upsert), creating or evolving
// its backing storage to match the field set.
func (s *session) PutSchema(ctx context.Context, def *schema.Def) error {
	old, err := getSchemaRaw(ctx, s.q, def.URI.NS(), def.URI.Schema())
	if err != nil {
		return err
	}

	if err := putSchemaRow(ctx, s.q, def); err != nil {
		return err
	}

	eng := engineFor(s.q, def)
	if old == nil {
		return eng.ensure(ctx, def.URI)
	}
	return eng.evolve(ctx, old)
}

// DeleteSchema deletes a definition. Returns [core.ErrNotFound] if absent.
// Record data is left in place — DropRecords owns record cleanup.
func (s *session) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return deleteSchemaRow(ctx, s.q, uri)
}

// DropRecords deletes all record tuples belonging to a schema by
// dropping both possible backing tables. It is deliberately unrouted:
// cleanup must remove whichever layout the schema ever used.
func (s *session) DropRecords(ctx context.Context, uri *core.URI) error {
	if err := (&kvEngine{q: s.q}).drop(ctx, uri); err != nil {
		return err
	}
	return (&tableEngine{q: s.q}).drop(ctx, uri)
}

// --- Query pushdown ---

// QueryTuples pushes a schema-scoped query down to SQL. Namespace-
// scoped queries return [store.ErrUnsupportedQuery] so the facade
// synthesizes them from a scan.
func (s *session) QueryTuples(
	ctx context.Context,
	q *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	if q.URI == nil || q.URI.Schema() == "" {
		return nil, store.ErrUnsupportedQuery
	}

	def, err := getSchemaRaw(ctx, s.q, q.URI.NS(), q.URI.Schema())
	if err != nil {
		return nil, err
	}

	return engineFor(s.q, def).queryRecords(ctx, q)
}
