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
	return `"kv:` + uri.NS() + `/` + uri.Schema() + `"`
}

// columnTableName returns the quoted table name for a column-strategy schema.
// Format: "t:<ns>/<schema>".
func columnTableName(uri *core.URI) string {
	return `"t:` + uri.NS() + `/` + uri.Schema() + `"`
}

// columnDefs builds sorted [xsql.Column] definitions from a schema.
// Columns are sorted alphabetically for deterministic ordering.
func columnDefs(def *schema.Def) []xsql.Column {
	names := sortedColumns(def)
	cols := make([]xsql.Column, len(names))
	for i, name := range names {
		cols[i] = xsql.Column{
			Name: name,
			Type: xsql.SQLiteTypeName(def.Fields[name].Type.ID().String()),
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

// columnValues builds sorted [xsql.Value] descriptors (Name + Type)
// from a schema for use in read operations.
func columnValues(def *schema.Def) []xsql.Value {
	names := sortedColumns(def)
	vals := make([]xsql.Value, len(names))
	for i, name := range names {
		vals[i] = xsql.Value{
			Name: name,
			Type: def.Fields[name].Type,
		}
	}
	return vals
}

// valuesFromTuples maps tuples onto the schema's full column set in
// alphabetical order. Columns without a tuple get a nil Val (NULL).
// Tuples for attrs the schema does not declare are dropped — a column
// table physically cannot hold them.
func valuesFromTuples(def *schema.Def, tuples []*core.Tuple) []xsql.Value {
	byAttr := make(map[string]*core.Tuple, len(tuples))
	for _, tuple := range tuples {
		byAttr[tuple.Attr()] = tuple
	}

	cols := sortedColumns(def)
	vals := make([]xsql.Value, len(cols))
	for i, col := range cols {
		v := xsql.Value{Name: col}
		if tuple, ok := byAttr[col]; ok {
			v.Val = tuple.Value()
		}
		vals[i] = v
	}
	return vals
}

// kvValuesFromTuples maps tuples to KV rows, one per tuple.
func kvValuesFromTuples(tuples []*core.Tuple) []xsql.Value {
	vals := make([]xsql.Value, len(tuples))
	for i, tuple := range tuples {
		vals[i] = xsql.Value{
			Name: tuple.Attr(),
			Val:  tuple.Value(),
		}
	}
	return vals
}

// tuplesFromValues builds tuples at path from row values, skipping
// NULL columns and the _id pseudo-column — a NULL column is not a
// tuple.
func tuplesFromValues(path *core.URI, vals []xsql.Value) []*core.Tuple {
	tuples := make([]*core.Tuple, 0, len(vals))
	for _, v := range vals {
		if v.Val == nil || v.Name == "_id" {
			continue
		}
		tuples = append(tuples, core.NewTuple(path.Path(), v.Name, v.Val))
	}
	return tuples
}
