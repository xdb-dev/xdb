package xdbsqlite

import (
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// kvTableName returns the quoted table name for a KV-strategy schema.
// Format: "kv:<ns>/<schema>".
func kvTableName(uri *core.URI) string {
	return `"kv:` + uri.NS().String() + `/` + uri.Schema().String() + `"`
}

// columnTableName returns the quoted table name for a column-strategy schema.
// Format: "t:<ns>/<schema>".
func columnTableName(uri *core.URI) string {
	return `"t:` + uri.NS().String() + `/` + uri.Schema().String() + `"`
}

// columnDefs builds sorted [xsql.Column] definitions from a schema.
// Columns are sorted alphabetically for deterministic ordering.
func columnDefs(def *schema.Def) []xsql.Column { //nolint:unused // used once implementations land
	names := sortedColumns(def)
	cols := make([]xsql.Column, len(names))
	for i, name := range names {
		cols[i] = xsql.Column{
			Name: name,
			Type: xsql.SQLiteTypeName(string(def.Fields[name].Type)),
		}
	}
	return cols
}

// sortedColumns returns alphabetically sorted column names from a schema.
func sortedColumns(def *schema.Def) []string {
	names := make([]string, 0, len(def.Fields))
	for name := range def.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// columnValues builds sorted [xsql.Value] descriptors (Name + Type) from a schema
// for use in read operations. Columns are sorted alphabetically.
func columnValues(def *schema.Def) []xsql.Value {
	names := sortedColumns(def)
	vals := make([]xsql.Value, len(names))
	for i, name := range names {
		vals[i] = xsql.Value{
			Name: name,
			Type: core.NewType(def.Fields[name].Type),
		}
	}
	return vals
}

// recordToValues extracts column values from a [core.Record] as [xsql.Value]
// in alphabetical column order matching the schema. Missing columns produce
// nil Val fields.
func recordToValues(def *schema.Def, record *core.Record) []xsql.Value { //nolint:unused // used once implementations land
	cols := sortedColumns(def)
	vals := make([]xsql.Value, len(cols))
	for i, col := range cols {
		v := xsql.Value{Name: col}
		if t := record.Get(col); t != nil {
			v.Val = t.Value()
		}
		vals[i] = v
	}
	return vals
}
