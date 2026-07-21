package xdbsqlite

import (
	"context"
	"encoding/json"
	"iter"
	"sort"

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

// schemaForRecord returns the definition governing a record path, or nil.
func schemaForRecord(
	ctx context.Context,
	q *xsql.Queries,
	path *core.URI,
) (*schema.Def, error) {
	return getSchemaRaw(ctx, q, path.NS(), path.Schema())
}

// useColumnTable reports whether records of this definition live in a
// column table. Flexible and schema-less records use KV tables.
func useColumnTable(def *schema.Def) bool {
	return def != nil && def.Mode != schema.ModeFlexible
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

// createBackingTable creates the table records of this definition
// live in, per its mode.
func createBackingTable(
	ctx context.Context,
	q *xsql.Queries,
	def *schema.Def,
) error {
	if useColumnTable(def) {
		return q.CreateTable(ctx, xsql.CreateTableParams{
			Table:   columnTableName(def.URI),
			Columns: columnDefs(def),
		})
	}
	return q.CreateKVTable(ctx, xsql.CreateKVTableParams{
		Table: kvTableName(def.URI),
	})
}

// createSchema stores a new definition and creates its backing table.
func createSchema(ctx context.Context, q *xsql.Queries, def *schema.Def) error {
	exists, err := q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: def.URI.NS(),
		Schema:    def.URI.Schema(),
	})
	if err != nil {
		return err
	}
	if exists {
		return core.ErrAlreadyExists
	}

	if err := putSchemaRow(ctx, q, def); err != nil {
		return err
	}

	return createBackingTable(ctx, q, def)
}

// putSchema upserts a definition, creating or evolving its backing table
// (DDL for added and removed fields on column tables).
func putSchema(ctx context.Context, q *xsql.Queries, def *schema.Def) error {
	old, err := getSchemaRaw(ctx, q, def.URI.NS(), def.URI.Schema())
	if err != nil {
		return err
	}

	if err := putSchemaRow(ctx, q, def); err != nil {
		return err
	}

	if old == nil {
		return createBackingTable(ctx, q, def)
	}

	return evolveBackingTable(ctx, q, old, def)
}

// evolveBackingTable alters a column table to match the new field
// set: added fields become columns, removed fields drop theirs. KV
// tables need no DDL. Type changes are middleware's job to reject
// (schema.ValidateUpdate) — the driver stores what it is given.
func evolveBackingTable(
	ctx context.Context,
	q *xsql.Queries,
	old, updated *schema.Def,
) error {
	if !useColumnTable(updated) {
		return nil
	}

	table := columnTableName(updated.URI)

	added := make([]string, 0, len(updated.Fields))
	for name := range updated.Fields {
		if _, ok := old.Fields[name]; !ok {
			added = append(added, name)
		}
	}
	sort.Strings(added)

	for _, name := range added {
		err := q.AddColumn(ctx, xsql.AddColumnParams{
			Table: table,
			Column: xsql.Column{
				Name: name,
				Type: xsql.SQLiteTypeName(updated.Fields[name].Type.ID().String()),
			},
		})
		if err != nil {
			return err
		}
	}

	removed := make([]string, 0, len(old.Fields))
	for name := range old.Fields {
		if _, ok := updated.Fields[name]; !ok {
			removed = append(removed, name)
		}
	}
	sort.Strings(removed)

	for _, name := range removed {
		err := q.DropColumn(ctx, xsql.DropColumnParams{
			Table:  table,
			Column: name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteSchema removes the definition row. Backing tables are left in
// place; DropRecords owns record cleanup.
func deleteSchema(ctx context.Context, q *xsql.Queries, uri *core.URI) error {
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

// deleteSchemaRecords drops the schema's backing tables. Dropping
// (rather than clearing) also disposes of column layouts, so a
// re-created schema starts from a clean table.
func deleteSchemaRecords(ctx context.Context, q *xsql.Queries, uri *core.URI) error {
	if err := q.DropTable(ctx, xsql.DropTableParams{Table: kvTableName(uri)}); err != nil {
		return err
	}
	return q.DropTable(ctx, xsql.DropTableParams{Table: columnTableName(uri)})
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
