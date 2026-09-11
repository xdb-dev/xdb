package xdbsqlite

import (
	"errors"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/filter/sqlgen"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// compileWhere compiles a CEL filter to a SQL WHERE clause. An empty
// filter yields no clause.
//
// Two sqlgen faults are not the caller's. A field absent from the schema's
// current column set (dynamic mode, a field not yet written) gives
// [sqlgen.ErrUnknownColumn]. A valid CEL construct with no SQL form, such
// as matches(), gives [sqlgen.ErrUnsupportedExpr]. Both map to
// [store.ErrUnsupportedQuery], so the facade falls back to a scan and
// evaluates the filter in memory.
//
// Any other sqlgen fault is bad input, so it is wrapped as
// [core.ErrInvalidFilter] and reaches the caller as an argument error, not
// an internal one. Strict-mode rejection of unknown filter fields already
// happened in [filter.Compile] and never reaches sqlgen.
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
		if errors.Is(err, sqlgen.ErrUnknownColumn) || errors.Is(err, sqlgen.ErrUnsupportedExpr) {
			return "", nil, store.ErrUnsupportedQuery
		}

		return "", nil, fmt.Errorf("%w: %w", core.ErrInvalidFilter, err)
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
