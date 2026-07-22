package xdbsqlite

import (
	"context"
	"iter"
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter/sqlgen"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// tableEngine stores a schema's records in a per-schema column table,
// one row per record and one column per field. It backs strict and
// dynamic schemas, so def is never nil.
type tableEngine struct {
	q   *xsql.Queries
	def *schema.Def
}

func (e *tableEngine) readRecord(ctx context.Context, path *core.URI) ([]*core.Tuple, error) {
	vals, err := e.q.GetRecord(ctx, xsql.GetRecordParams{
		Table:   columnTableName(e.def.URI),
		ID:      path.ID(),
		Columns: columnValues(e.def),
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

func (e *tableEngine) exists(ctx context.Context, path *core.URI) (bool, error) {
	exists, err := e.q.RecordExists(ctx, xsql.RecordExistsParams{
		Table: columnTableName(e.def.URI),
		ID:    path.ID(),
	})
	if isNoTable(err) {
		return false, nil
	}
	return exists, err
}

func (e *tableEngine) writeRecord(ctx context.Context, path *core.URI, tuples []*core.Tuple) error {
	table := columnTableName(e.def.URI)
	if len(tuples) == 0 {
		return e.q.DeleteRecord(ctx, xsql.DeleteRecordParams{
			Table: table,
			ID:    path.ID(),
		})
	}
	return e.q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table:  table,
		ID:     path.ID(),
		Values: valuesFromTuples(e.def, tuples),
	})
}

func (e *tableEngine) scanRecords(
	ctx context.Context,
	schemaURI *core.URI,
) iter.Seq2[[]*core.Tuple, error] {
	table := columnTableName(e.def.URI)
	columns := columnValues(e.def)
	return func(yield func([]*core.Tuple, error) bool) {
		for offset := 0; ; offset += recordScanBatch {
			rows, err := e.q.ListRecords(ctx, xsql.ListRecordsParams{
				Table:   table,
				Columns: columns,
				Limit:   recordScanBatch,
				Offset:  offset,
			})
			if isNoTable(err) {
				return
			}
			if err != nil {
				yield(nil, err)
				return
			}

			for _, row := range rows {
				id, idErr := row[0].Val.AsStr()
				if idErr != nil {
					yield(nil, idErr)
					return
				}
				path := core.MustNewURI(schemaURI.NS(), schemaURI.Schema(), id)
				if !yield(tuplesFromValues(path, row[1:]), nil) {
					return
				}
			}

			if len(rows) < recordScanBatch {
				return
			}
		}
	}
}

func (e *tableEngine) queryRecords(
	ctx context.Context,
	q *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	table := columnTableName(e.def.URI)

	whereSQL, whereArgs, err := compileWhere(q.Filter, e.def, sqlgen.ColumnStrategy, table)
	if err != nil {
		return nil, err
	}

	total, err := e.q.CountRecords(ctx, xsql.CountRecordsParams{
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

	limit, offset := paginationParams(q)

	rows, err := e.q.ListRecords(ctx, xsql.ListRecordsParams{
		Table:     table,
		Columns:   columnValues(e.def),
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
		path := core.MustNewURI(e.def.URI.NS(), e.def.URI.Schema(), id)
		items = append(items, tuplesFromValues(path, row[1:]))
	}

	return buildPage(total, offset, items), nil
}

func (e *tableEngine) ensure(ctx context.Context, _ *core.URI) error {
	return e.q.CreateTable(ctx, xsql.CreateTableParams{
		Table:   columnTableName(e.def.URI),
		Columns: columnDefs(e.def),
	})
}

// evolve alters the column table to match the new field set: added
// fields become columns, removed fields drop theirs. Type changes are
// middleware's job to reject; the driver stores what it is given.
func (e *tableEngine) evolve(ctx context.Context, old *schema.Def) error {
	table := columnTableName(e.def.URI)

	added := make([]string, 0, len(e.def.Fields))
	for name := range e.def.Fields {
		if _, ok := old.Fields[name]; !ok {
			added = append(added, name)
		}
	}
	sort.Strings(added)

	for _, name := range added {
		err := e.q.AddColumn(ctx, xsql.AddColumnParams{
			Table: table,
			Column: xsql.Column{
				Name: name,
				Type: xsql.SQLiteTypeName(e.def.Fields[name].Type.ID().String()),
			},
		})
		if err != nil {
			return err
		}
	}

	removed := make([]string, 0, len(old.Fields))
	for name := range old.Fields {
		if _, ok := e.def.Fields[name]; !ok {
			removed = append(removed, name)
		}
	}
	sort.Strings(removed)

	for _, name := range removed {
		err := e.q.DropColumn(ctx, xsql.DropColumnParams{
			Table:  table,
			Column: name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *tableEngine) drop(ctx context.Context, schemaURI *core.URI) error {
	return e.q.DropTable(ctx, xsql.DropTableParams{Table: columnTableName(schemaURI)})
}
