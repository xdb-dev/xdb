package xdbsqlite

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// schemaScanLimit bounds def listings; schemas are metadata and far
// fewer than records.
const schemaScanLimit = 10000

// getSchemaRaw returns the stored definition, or nil if absent.
func getSchemaRaw(
	ctx context.Context,
	q *xsql.Queries,
	ns, name string,
) (*schema.Def, error) {
	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{
		Namespace: ns,
		Schema:    name,
	})
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// putSchemaRow stores the definition JSON verbatim.
func putSchemaRow(ctx context.Context, q *xsql.Queries, def *schema.Def) error {
	data, err := json.Marshal(def)
	if err != nil {
		return err
	}
	return q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: def.URI.NS(),
		Schema:    def.URI.Schema(),
		Data:      data,
	})
}

// deleteSchemaRow removes the definition row. Backing tables are left
// in place; DropRecords owns record cleanup.
func deleteSchemaRow(ctx context.Context, q *xsql.Queries, uri *core.URI) error {
	exists, err := q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: uri.NS(),
		Schema:    uri.Schema(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return core.ErrNotFound
	}

	return q.DeleteSchema(ctx, xsql.DeleteSchemaParams{
		Namespace: uri.NS(),
		Schema:    uri.Schema(),
	})
}

// scanSchemas yields definitions under scope, ordered by (ns, schema).
func scanSchemas(
	ctx context.Context,
	q *xsql.Queries,
	scope *core.URI,
) iter.Seq2[*schema.Def, error] {
	return func(yield func(*schema.Def, error) bool) {
		var ns *string
		if scope != nil {
			n := scope.NS()
			ns = &n
		}

		rows, err := q.ListSchemas(ctx, xsql.ListSchemasParams{
			Namespace: ns,
			Limit:     schemaScanLimit,
		})
		if err != nil {
			yield(nil, err)
			return
		}

		for _, row := range rows {
			var def schema.Def
			if err := json.Unmarshal(row.Data, &def); err != nil {
				yield(nil, err)
				return
			}
			if !yield(&def, nil) {
				return
			}
		}
	}
}
