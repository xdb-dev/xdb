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
func columnDefs(def *schema.Def) []xsql.Column {
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
func recordToValues(def *schema.Def, record *core.Record) []xsql.Value {
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

// kvRecordFromValues builds a [*core.Record] from KV values, filtering out
// the sentinel attribute used for empty records.
func kvRecordFromValues(uri *core.URI, values []xsql.Value) *core.Record {
	record := core.NewRecord(uri.NS().String(), uri.Schema().String(), uri.ID().String())
	for _, v := range values {
		if v.Name == "_" {
			continue
		}
		record.Set(v.Name, v.Val)
	}
	return record
}

// kvValues extracts all tuple values from a [core.Record] as [xsql.Value]
// for KV table writes. Empty records get a sentinel attribute so the record
// ID is always trackable via KVRecordExists and CountKVRecords.
func kvValues(record *core.Record) []xsql.Value {
	tuples := record.Tuples()
	if len(tuples) == 0 {
		return []xsql.Value{{
			Name: "_",
			Val:  core.BoolVal(true),
		}}
	}
	vals := make([]xsql.Value, len(tuples))
	for i, t := range tuples {
		vals[i] = xsql.Value{
			Name: t.Attr().String(),
			Val:  t.Value(),
		}
	}
	return vals
}
