package xdbsqlite

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// recordScanBatch is the page size for full-table scans.
const recordScanBatch = 1000

// isNoTable reports whether err is SQLite's missing-table error. KV
// tables are created lazily on first write, so reads treat a missing
// table as an empty one.
func isNoTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

// recordTuples reads one record's tuples. Returns nil when the record
// does not exist.
func recordTuples(
	ctx context.Context,
	q *xsql.Queries,
	path *core.URI,
	def *schema.Def,
) ([]*core.Tuple, error) {
	if useColumnTable(def) {
		vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
			Table:   columnTableName(def.URI),
			ID:      path.ID(),
			Columns: columnValues(def),
		})
		if isNoTable(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if vals == nil {
			return nil, nil
		}
		return tuplesFromValues(path, vals), nil
	}

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{
		Table: kvTableName(path.SchemaURI()),
		ID:    path.ID(),
	})
	if isNoTable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if vals == nil {
		return nil, nil
	}
	return tuplesFromValues(path, vals), nil
}

// getTuples resolves attr-level point reads, omitting absences and
// preserving request order. Records are read at most once per call.
func getTuples(
	ctx context.Context,
	q *xsql.Queries,
	uris []*core.URI,
) ([]*core.Tuple, error) {
	defs := make(map[string]*schema.Def)
	records := make(map[string]map[string]*core.Tuple)

	var got []*core.Tuple
	for _, uri := range uris {
		recordPath := uri.RecordPath()

		attrs, ok := records[recordPath]
		if !ok {
			schemaPath := uri.SchemaURI().Path()
			def, defLoaded := defs[schemaPath]
			if !defLoaded {
				var err error
				def, err = getSchemaRaw(ctx, q, uri.NS(), uri.Schema())
				if err != nil {
					return nil, err
				}
				defs[schemaPath] = def
			}

			tuples, err := recordTuples(ctx, q, uri, def)
			if err != nil {
				return nil, err
			}

			attrs = make(map[string]*core.Tuple, len(tuples))
			for _, tuple := range tuples {
				attrs[tuple.Attr()] = tuple
			}
			records[recordPath] = attrs
		}

		if tuple, ok := attrs[uri.Attr()]; ok {
			got = append(got, tuple)
		}
	}

	return got, nil
}

// tableRef identifies one backing table discovered in sqlite_master.
type tableRef struct {
	name   string // unquoted sqlite_master name
	ns     string
	schema string
	kv     bool
}

// quoted returns the identifier-quoted table name for queries.
func (t tableRef) quoted() string {
	return `"` + t.name + `"`
}

// parseTableRef parses a backing-table name ("kv:ns/schema" or
// "t:ns/schema"). Returns false for tables of neither kind.
func parseTableRef(name string) (tableRef, bool) {
	var rest string
	var kv bool

	switch {
	case strings.HasPrefix(name, "kv:"):
		rest, kv = name[len("kv:"):], true
	case strings.HasPrefix(name, "t:"):
		rest, kv = name[len("t:"):], false
	default:
		return tableRef{}, false
	}

	ns, schemaName, ok := strings.Cut(rest, "/")
	if !ok {
		return tableRef{}, false
	}

	return tableRef{
		name:   name,
		ns:     ns,
		schema: schemaName,
		kv:     kv,
	}, true
}

// listScopeTables discovers the backing tables under a namespace or
// schema scope, sorted by (ns, schema, kind) for deterministic scans.
func listScopeTables(
	ctx context.Context,
	q *xsql.Queries,
	scope *core.URI,
) ([]tableRef, error) {
	suffix := scope.NS() + "/*"
	if scope.Schema() != "" {
		suffix = scope.NS() + "/" + scope.Schema()
	}

	var refs []tableRef
	for _, prefix := range []string{"kv:", "t:"} {
		names, err := q.ListTables(ctx, xsql.ListTablesParams{
			Pattern: prefix + suffix,
		})
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if ref, ok := parseTableRef(name); ok {
				refs = append(refs, ref)
			}
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].ns != refs[j].ns {
			return refs[i].ns < refs[j].ns
		}
		if refs[i].schema != refs[j].schema {
			return refs[i].schema < refs[j].schema
		}
		return refs[i].kv && !refs[j].kv
	})

	return refs, nil
}

// scanTuples yields every tuple under scope. Records are contiguous:
// each table is paged in _id order, one record's tuples yielded
// together.
func scanTuples(
	ctx context.Context,
	q *xsql.Queries,
	scope *core.URI,
) iter.Seq2[*core.Tuple, error] {
	return func(yield func(*core.Tuple, error) bool) {
		if scope != nil && scope.ID() != "" {
			scanOneRecord(ctx, q, scope, yield)
			return
		}

		refs, err := listScopeTables(ctx, q, scope)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, ref := range refs {
			if !scanTable(ctx, q, ref, yield) {
				return
			}
		}
	}
}

// scanOneRecord yields the tuples of a single record path.
func scanOneRecord(
	ctx context.Context,
	q *xsql.Queries,
	path *core.URI,
	yield func(*core.Tuple, error) bool,
) {
	def, err := schemaForRecord(ctx, q, path)
	if err != nil {
		yield(nil, err)
		return
	}

	tuples, err := recordTuples(ctx, q, path, def)
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

// scanTable yields every tuple in one backing table, paged by record.
// Returns false if the consumer stopped the iteration.
func scanTable(
	ctx context.Context,
	q *xsql.Queries,
	ref tableRef,
	yield func(*core.Tuple, error) bool,
) bool {
	if ref.kv {
		return scanKVTable(ctx, q, ref, yield)
	}
	return scanColumnTable(ctx, q, ref, yield)
}

func scanKVTable(
	ctx context.Context,
	q *xsql.Queries,
	ref tableRef,
	yield func(*core.Tuple, error) bool,
) bool {
	for offset := 0; ; offset += recordScanBatch {
		rows, err := q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
			Table:  ref.quoted(),
			Limit:  recordScanBatch,
			Offset: offset,
		})
		if err != nil {
			return yield(nil, err)
		}

		for _, row := range rows {
			path := core.MustNewURI(ref.ns, ref.schema, row.ID)
			for _, tuple := range tuplesFromValues(path, row.Values) {
				if !yield(tuple, nil) {
					return false
				}
			}
		}

		if len(rows) < recordScanBatch {
			return true
		}
	}
}

func scanColumnTable(
	ctx context.Context,
	q *xsql.Queries,
	ref tableRef,
	yield func(*core.Tuple, error) bool,
) bool {
	// A column table is unreadable without its def (column layout).
	// Defs deleted out from under their tables leave the data
	// unreachable until re-created — same posture as the old store.
	def, err := getSchemaRaw(ctx, q, ref.ns, ref.schema)
	if err != nil {
		return yield(nil, err)
	}
	if !useColumnTable(def) {
		return true
	}

	columns := columnValues(def)
	for offset := 0; ; offset += recordScanBatch {
		rows, err := q.ListRecords(ctx, xsql.ListRecordsParams{
			Table:   ref.quoted(),
			Columns: columns,
			Limit:   recordScanBatch,
			Offset:  offset,
		})
		if err != nil {
			return yield(nil, err)
		}

		for _, row := range rows {
			id, idErr := row[0].Val.AsStr()
			if idErr != nil {
				return yield(nil, idErr)
			}
			path := core.MustNewURI(ref.ns, ref.schema, id)
			for _, tuple := range tuplesFromValues(path, row[1:]) {
				if !yield(tuple, nil) {
					return false
				}
			}
		}

		if len(rows) < recordScanBatch {
			return true
		}
	}
}

// --- Mutations ---

// applyMutation routes one mutation to its backing table by the
// stored def's mode: strict/dynamic → column table, flexible or
// schema-less → KV table.
func applyMutation(
	ctx context.Context,
	q *xsql.Queries,
	m store.Mutation,
) error {
	def, err := schemaForRecord(ctx, q, m.Path)
	if err != nil {
		return err
	}

	if useColumnTable(def) {
		return applyColumn(ctx, q, def, m)
	}
	return applyKV(ctx, q, m)
}

// applyOps runs one mutation's op against a record, given whether it
// currently exists, a reader for its current tuples, and a full-set
// writer (an empty set removes the record). The four-op semantics are
// identical for column and KV tables — only exists/read/writeFull
// differ — so both storage strategies share this body.
func applyOps(
	m store.Mutation,
	exists bool,
	read func() ([]*core.Tuple, error),
	writeFull func([]*core.Tuple) error,
) error {
	switch m.Op {
	case store.OpPatch:
		if !exists {
			return writeFull(m.Tuples)
		}
		current, err := read()
		if err != nil {
			return err
		}
		return writeFull(store.MergeTuples(current, m.Tuples))

	case store.OpCreate:
		if exists {
			return core.ErrAlreadyExists
		}
		return writeFull(m.Tuples)

	case store.OpPut:
		return writeFull(m.Tuples)

	case store.OpDelete:
		if len(m.Attrs) == 0 {
			return writeFull(nil)
		}
		if !exists {
			return nil
		}
		current, err := read()
		if err != nil {
			return err
		}
		return writeFull(store.RemoveAttrs(current, m.Attrs))

	default:
		return fmt.Errorf("xdbsqlite: unknown op %s", m.Op)
	}
}

func applyColumn(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	m store.Mutation,
) error {
	table := columnTableName(def.URI)
	id := m.Path.ID()

	exists, err := q.RecordExists(ctx, xsql.RecordExistsParams{
		Table: table,
		ID:    id,
	})
	if err != nil {
		return err
	}

	read := func() ([]*core.Tuple, error) {
		return recordTuples(ctx, q, m.Path, def)
	}
	// writeFull replaces the record's full tuple set; an empty set
	// means the record ceases to exist (row removed), preserving the
	// row-exists ⇔ has-tuples invariant.
	writeFull := func(tuples []*core.Tuple) error {
		if len(tuples) == 0 {
			return q.DeleteRecord(ctx, xsql.DeleteRecordParams{
				Table: table,
				ID:    id,
			})
		}
		return q.UpsertRecord(ctx, xsql.UpsertRecordParams{
			Table:  table,
			ID:     id,
			Values: valuesFromTuples(def, tuples),
		})
	}

	return applyOps(m, exists, read, writeFull)
}

func applyKV(
	ctx context.Context,
	q *xsql.Queries,
	m store.Mutation,
) error {
	table := kvTableName(m.Path.SchemaURI())
	id := m.Path.ID()

	// KV tables are created lazily on first write.
	if err := q.CreateKVTable(ctx, xsql.CreateKVTableParams{Table: table}); err != nil {
		return err
	}

	exists, err := q.KVRecordExists(ctx, xsql.KVRecordExistsParams{
		Table: table,
		ID:    id,
	})
	if err != nil {
		return err
	}

	read := func() ([]*core.Tuple, error) {
		return recordTuples(ctx, q, m.Path, nil)
	}
	writeFull := func(tuples []*core.Tuple) error {
		if len(tuples) == 0 {
			return q.DeleteKVRecord(ctx, xsql.DeleteKVRecordParams{
				Table: table,
				ID:    id,
			})
		}
		return q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
			Table:  table,
			ID:     id,
			Values: kvValuesFromTuples(tuples),
		})
	}

	return applyOps(m, exists, read, writeFull)
}
