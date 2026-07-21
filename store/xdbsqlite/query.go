package xdbsqlite

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/filter/sqlgen"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// queryTuples pushes a schema-scoped list query down to SQL: the CEL
// filter compiles to a WHERE clause (column-strategy for column
// tables, KV-strategy for KV tables), counting and pagination happen
// in the database. Namespace-scoped queries span heterogeneous
// schemas and are declined with [store.ErrUnsupportedQuery] — the
// facade synthesizes them from a scan, which keeps filter semantics
// identical across backends.
func queryTuples(
	ctx context.Context,
	q *xsql.Queries,
	sq *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	uri := sq.URI
	if uri == nil || uri.Schema() == "" {
		return nil, store.ErrUnsupportedQuery
	}

	def, err := getSchemaRaw(ctx, q, uri.NS(), uri.Schema())
	if err != nil {
		return nil, err
	}

	if useColumnTable(def) {
		return queryColumnTable(ctx, q, def, sq)
	}
	return queryKVTable(ctx, q, def, sq)
}

func queryColumnTable(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	sq *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	table := columnTableName(def.URI)

	whereSQL, whereArgs, err := compileWhere(sq.Filter, def, sqlgen.ColumnStrategy, table)
	if err != nil {
		return nil, err
	}

	total, err := q.CountRecords(ctx, xsql.CountRecordsParams{
		Table:     table,
		Where:     whereSQL,
		WhereArgs: whereArgs,
	})
	if isNoTable(err) {
		return emptyPage(), nil
	}
	if err != nil {
		return nil, err
	}

	limit, offset := paginationParams(sq)

	rows, err := q.ListRecords(ctx, xsql.ListRecordsParams{
		Table:     table,
		Columns:   columnValues(def),
		Where:     whereSQL,
		WhereArgs: whereArgs,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([][]*core.Tuple, 0, len(rows))
	for _, row := range rows {
		id, idErr := row[0].Val.AsStr()
		if idErr != nil {
			return nil, idErr
		}
		path := core.MustNewURI(def.URI.NS(), def.URI.Schema(), id)
		items = append(items, tuplesFromValues(path, row[1:]))
	}

	return buildPage(total, offset, items), nil
}

func queryKVTable(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
	sq *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	uri := sq.URI
	table := kvTableName(uri)

	whereSQL, whereArgs, err := compileWhere(sq.Filter, def, sqlgen.KVStrategy, table)
	if err != nil {
		return nil, err
	}

	total, err := q.CountKVRecords(ctx, xsql.CountKVRecordsParams{
		Table:     table,
		Where:     whereSQL,
		WhereArgs: whereArgs,
	})
	if isNoTable(err) {
		return emptyPage(), nil
	}
	if err != nil {
		return nil, err
	}

	limit, offset := paginationParams(sq)

	rows, err := q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
		Table:     table,
		Where:     whereSQL,
		WhereArgs: whereArgs,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([][]*core.Tuple, 0, len(rows))
	for _, row := range rows {
		path := core.MustNewURI(uri.NS(), uri.Schema(), row.ID)
		items = append(items, tuplesFromValues(path, row.Values))
	}

	return buildPage(total, offset, items), nil
}

// compileWhere compiles a CEL filter to a SQL WHERE clause. An empty
// filter yields no clause.
func compileWhere(
	filterExpr string,
	def *schema.Def,
	strategy sqlgen.Strategy,
	table string,
) (string, []any, error) {
	if filterExpr == "" {
		return "", nil, nil
	}

	f, err := filter.Compile(filterExpr, def)
	if err != nil {
		return "", nil, err
	}

	wc, err := sqlgen.Generate(f, strategy, table)
	if err != nil {
		return "", nil, err
	}

	return wc.SQL, wc.Params, nil
}

func emptyPage() *store.Page[[]*core.Tuple] {
	return &store.Page[[]*core.Tuple]{}
}

// buildPage assembles a tuple page from a query's total, request
// offset, and assembled items — shared by the column and KV paths.
func buildPage(total, offset int, items [][]*core.Tuple) *store.Page[[]*core.Tuple] {
	return &store.Page[[]*core.Tuple]{
		Items:      items,
		Total:      total,
		NextOffset: nextOffset(offset, len(items), total),
	}
}

// paginationParams extracts limit/offset from a query, defaulting to
// [store.DefaultLimit] and capping at [store.MaxLimit].
func paginationParams(q *store.Query) (limit, offset int) {
	if q == nil {
		return store.DefaultLimit, 0
	}
	limit = q.Limit
	if limit <= 0 {
		limit = store.DefaultLimit
	}
	if limit > store.MaxLimit {
		limit = store.MaxLimit
	}
	return limit, q.Offset
}

// nextOffset computes the next page offset, 0 when no more pages.
func nextOffset(offset, fetched, total int) int {
	next := offset + fetched
	if next >= total {
		return 0
	}
	return next
}
