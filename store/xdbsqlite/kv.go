package xdbsqlite

import (
	"context"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter/sqlgen"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// kvEngine stores a schema's records in a per-schema KV table, one row
// per attribute. It backs flexible and schema-less records, so def may
// be nil.
type kvEngine struct {
	q   *xsql.Queries
	def *schema.Def
}

func (e *kvEngine) readRecord(ctx context.Context, path *core.URI) ([]*core.Tuple, error) {
	vals, err := e.q.GetKVRecord(ctx, xsql.GetKVRecordParams{
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

func (e *kvEngine) exists(ctx context.Context, path *core.URI) (bool, error) {
	exists, err := e.q.KVRecordExists(ctx, xsql.KVRecordExistsParams{
		Table: kvTableName(path.SchemaURI()),
		ID:    path.ID(),
	})
	if isNoTable(err) {
		return false, nil
	}
	return exists, err
}

func (e *kvEngine) writeRecord(ctx context.Context, path *core.URI, tuples []*core.Tuple) error {
	table := kvTableName(path.SchemaURI())
	if len(tuples) == 0 {
		return e.q.DeleteKVRecord(ctx, xsql.DeleteKVRecordParams{
			Table: table,
			ID:    path.ID(),
		})
	}
	return e.q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table:  table,
		ID:     path.ID(),
		Values: kvValuesFromTuples(tuples),
	})
}

func (e *kvEngine) scanRecords(
	ctx context.Context,
	schemaURI *core.URI,
) iter.Seq2[[]*core.Tuple, error] {
	table := kvTableName(schemaURI)
	return func(yield func([]*core.Tuple, error) bool) {
		for offset := 0; ; offset += recordScanBatch {
			rows, err := e.q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
				Table:  table,
				Limit:  recordScanBatch,
				Offset: offset,
			})
			if isNoTable(err) {
				return
			}
			if err != nil {
				yield(nil, err)
				return
			}

			for _, row := range rows {
				path := core.MustNewURI(schemaURI.NS(), schemaURI.Schema(), row.ID)
				if !yield(tuplesFromValues(path, row.Values), nil) {
					return
				}
			}

			if len(rows) < recordScanBatch {
				return
			}
		}
	}
}

func (e *kvEngine) queryRecords(
	ctx context.Context,
	q *store.Query,
) (*store.Page[[]*core.Tuple], error) {
	uri := q.URI
	table := kvTableName(uri)

	whereSQL, whereArgs, err := compileWhere(q.Filter, e.def, sqlgen.KVStrategy, table)
	if err != nil {
		return nil, err
	}

	total, err := e.q.CountKVRecords(ctx, xsql.CountKVRecordsParams{
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

	rows, err := e.q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
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

func (e *kvEngine) ensure(ctx context.Context, schemaURI *core.URI) error {
	if err := e.q.CreateKVTable(ctx, xsql.CreateKVTableParams{
		Table: kvTableName(schemaURI),
	}); err != nil {
		return err
	}
	return e.q.CreateIndex(ctx, xsql.CreateIndexParams{
		Table: kvTableName(schemaURI),
		Index: xsql.Index{
			Name:    kvIndexName(schemaURI),
			Columns: []string{"_attr", "_val"},
		},
	})
}

// evolve is a no-op for KV tables: they carry no per-field columns, so
// a field-set change needs no DDL.
func (e *kvEngine) evolve(context.Context, *schema.Def) error {
	return nil
}

func (e *kvEngine) drop(ctx context.Context, schemaURI *core.URI) error {
	return e.q.DropTable(ctx, xsql.DropTableParams{Table: kvTableName(schemaURI)})
}

// kvIndexName returns the quoted name of a KV table's (_attr,_val)
// index. Format: "ix:kv:<ns>/<schema>".
func kvIndexName(uri *core.URI) string {
	return `"ix:kv:` + uri.NS() + `/` + uri.Schema() + `"`
}
