package sql

import (
	"context"
	"fmt"
	"strings"
)

// Column describes a single table column for DDL operations.
type Column struct {
	Name    string
	Type    string
	Default string
	NotNull bool
	Unique  bool
}

// Index describes a table index for DDL operations.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// CreateKVTableParams are the arguments for [Queries.CreateKVTable].
type CreateKVTableParams struct {
	Table string
}

// CreateKVTable creates a KV table for flexible/schema-free records.
// One row per attribute: _type/_elem record the [core.TID] (and array
// element id) so values decode; _val is ANY so each value keeps its
// native storage class (INTEGER/REAL/TEXT/BLOB) for correct SQL
// comparisons.
func (q *Queries) CreateKVTable(ctx context.Context, arg CreateKVTableParams) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			_id   TEXT NOT NULL,
			_attr TEXT NOT NULL,
			_type TEXT NOT NULL,
			_elem TEXT,
			_val  ANY,
			PRIMARY KEY (_id, _attr)
		) STRICT
	`, arg.Table)

	_, err := q.db.ExecContext(ctx, query)
	return err
}

// CreateTableParams are the arguments for [Queries.CreateTable].
type CreateTableParams struct {
	Table      string
	Columns    []Column
	PrimaryKey []string
}

// CreateTable creates a column table for strict/dynamic schemas.
func (q *Queries) CreateTable(ctx context.Context, arg CreateTableParams) error {
	var parts []string
	parts = append(parts, "_id TEXT NOT NULL")

	for _, col := range arg.Columns {
		def := col.Name + " " + col.Type
		if col.NotNull {
			def += " NOT NULL"
		}
		if col.Unique {
			def += " UNIQUE"
		}
		if col.Default != "" {
			def += " DEFAULT " + col.Default
		}
		parts = append(parts, def)
	}

	pk := []string{"_id"}
	if len(arg.PrimaryKey) > 0 {
		pk = arg.PrimaryKey
	}
	parts = append(parts, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pk, ", ")))

	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n\t%s\n)",
		arg.Table,
		strings.Join(parts, ",\n\t"),
	)

	_, err := q.db.ExecContext(ctx, query)
	return err
}

// AddColumnParams are the arguments for [Queries.AddColumn].
type AddColumnParams struct {
	Table  string
	Column Column
}

// AddColumn adds a column to an existing table.
func (q *Queries) AddColumn(ctx context.Context, arg AddColumnParams) error {
	def := arg.Column.Name + " " + arg.Column.Type
	if arg.Column.NotNull {
		def += " NOT NULL"
	}
	if arg.Column.Default != "" {
		def += " DEFAULT " + arg.Column.Default
	}

	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", arg.Table, def)
	_, err := q.db.ExecContext(ctx, query)
	return err
}

// DropColumnParams are the arguments for [Queries.DropColumn].
type DropColumnParams struct {
	Table  string
	Column string
}

// DropColumn drops a column from an existing table.
func (q *Queries) DropColumn(ctx context.Context, arg DropColumnParams) error {
	query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", arg.Table, arg.Column)
	_, err := q.db.ExecContext(ctx, query)
	return err
}

// CreateIndexParams are the arguments for [Queries.CreateIndex].
type CreateIndexParams struct {
	Table string
	Index Index
}

// CreateIndex creates an index on a table.
func (q *Queries) CreateIndex(ctx context.Context, arg CreateIndexParams) error {
	unique := ""
	if arg.Index.Unique {
		unique = "UNIQUE "
	}

	query := fmt.Sprintf(
		"CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)",
		unique,
		arg.Index.Name,
		arg.Table,
		strings.Join(arg.Index.Columns, ", "),
	)

	_, err := q.db.ExecContext(ctx, query)
	return mapErr(err)
}

// DropIndexParams are the arguments for [Queries.DropIndex].
type DropIndexParams struct {
	Name string
}

// DropIndex drops an index by name. Used when a schema update removes an
// indexed field: the index must go before the column, or SQLite refuses
// to drop a column an index still references.
func (q *Queries) DropIndex(ctx context.Context, arg DropIndexParams) error {
	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", arg.Name)
	_, err := q.db.ExecContext(ctx, query)
	return err
}

// DropTableParams are the arguments for [Queries.DropTable].
type DropTableParams struct {
	Table string
}

// DropTable drops a table by name.
func (q *Queries) DropTable(ctx context.Context, arg DropTableParams) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", arg.Table)
	_, err := q.db.ExecContext(ctx, query)
	return err
}
