package xdbsqlite

import (
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
	"github.com/xdb-dev/xdb/x"
)

// columnDefs builds sorted [xsql.Column] definitions from a schema.
// Columns are sorted alphabetically for deterministic ordering.
func columnDefs(def *schema.Def) []xsql.Column {
	return x.Map(sortedColumns(def), func(name string) xsql.Column {
		return xsql.Column{
			Name: name,
			Type: xsql.SQLiteTypeName(def.Fields[name].Type.ID().String()),
		}
	})
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
	return x.Map(sortedColumns(def), func(name string) xsql.Value {
		return xsql.Value{
			Name: name,
			Type: def.Fields[name].Type,
		}
	})
}

// valuesFromTuples maps tuples onto the schema's full column set in
// alphabetical order. Columns without a tuple get a nil Val (NULL).
// Tuples for attrs the schema does not declare are dropped — a column
// table physically cannot hold them.
func valuesFromTuples(def *schema.Def, tuples []*core.Tuple) []xsql.Value {
	byAttr := x.Index(tuples, (*core.Tuple).Attr)

	return x.Map(sortedColumns(def), func(col string) xsql.Value {
		v := xsql.Value{Name: col}
		if tuple, ok := byAttr[col]; ok {
			v.Val = tuple.Value()
		}
		return v
	})
}

// kvValuesFromTuples maps tuples to KV rows, one per tuple.
func kvValuesFromTuples(tuples []*core.Tuple) []xsql.Value {
	return x.Map(tuples, func(tuple *core.Tuple) xsql.Value {
		return xsql.Value{
			Name: tuple.Attr(),
			Val:  tuple.Value(),
		}
	})
}

// tuplesFromValues builds tuples at path from row values, skipping
// NULL columns and the _id pseudo-column — a NULL column is not a
// tuple. This is the hottest converter (one call per record on every
// read), so it stays a single-pass loop rather than filter-then-map.
func tuplesFromValues(path *core.URI, vals []xsql.Value) []*core.Tuple {
	p := path.Path()
	tuples := make([]*core.Tuple, 0, len(vals))
	for _, v := range vals {
		if v.Val == nil || v.Name == "_id" {
			continue
		}
		tuples = append(tuples, core.NewTuple(p, v.Name, v.Val))
	}
	return tuples
}
