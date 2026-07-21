package xdbsqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestKVTableName(t *testing.T) {
	uri := core.MustNewURI("myns", "posts", "abc")
	assert.Equal(t, `"kv:myns/posts"`, kvTableName(uri))
}

func TestColumnTableName(t *testing.T) {
	uri := core.MustNewURI("myns", "posts", "abc")
	assert.Equal(t, `"t:myns/posts"`, columnTableName(uri))
}

func TestSQLiteTypeName(t *testing.T) {
	tests := []struct {
		tid  core.TID
		want string
	}{
		{core.TIDInteger, "INTEGER"},
		{core.TIDFloat, "REAL"},
		{core.TIDBoolean, "INTEGER"},
		{core.TIDUnsigned, "INTEGER"},
		{core.TIDTime, "INTEGER"},
		{core.TIDString, "TEXT"},
		{core.TIDBytes, "BLOB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, xsql.SQLiteTypeName(string(tt.tid)), "SQLiteTypeName(%v)", tt.tid)
	}
}

func TestColumnDefs(t *testing.T) {
	uri := core.MustNewURI("myns", "posts")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"title":  {Type: core.TypeString},
			"count":  {Type: core.TypeInt},
			"active": {Type: core.TypeBool},
		},
		Mode: schema.ModeStrict,
	}

	cols := columnDefs(def)

	require.Len(t, cols, 3)
	assert.Equal(t, "active", cols[0].Name)
	assert.Equal(t, "INTEGER", cols[0].Type)
	assert.Equal(t, "count", cols[1].Name)
	assert.Equal(t, "INTEGER", cols[1].Type)
	assert.Equal(t, "title", cols[2].Name)
	assert.Equal(t, "TEXT", cols[2].Type)
}

func TestSortedColumns(t *testing.T) {
	uri := core.MustNewURI("myns", "posts")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"zebra": {Type: core.TypeString},
			"alpha": {Type: core.TypeString},
			"mid":   {Type: core.TypeString},
		},
	}

	cols := sortedColumns(def)
	require.Equal(t, []string{"alpha", "mid", "zebra"}, cols)
}

func TestValuesFromTuples(t *testing.T) {
	uri := core.MustNewURI("myns", "posts")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"count": {Type: core.TypeInt},
		},
		Mode: schema.ModeStrict,
	}

	tuples := []*core.Tuple{
		core.NewTuple("myns/posts/abc", "title", "hello"),
		core.NewTuple("myns/posts/abc", "count", 42),
	}

	vals := valuesFromTuples(def, tuples)

	require.Len(t, vals, 2)
	assert.Equal(t, "count", vals[0].Name)
	count, err := vals[0].Val.AsInt()
	require.NoError(t, err)
	assert.Equal(t, int64(42), count)
	assert.Equal(t, "title", vals[1].Name)
	title, err := vals[1].Val.AsStr()
	require.NoError(t, err)
	assert.Equal(t, "hello", title)
}

func TestValuesFromTuples_MissingColumn(t *testing.T) {
	uri := core.MustNewURI("myns", "posts")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"title":   {Type: core.TypeString},
			"missing": {Type: core.TypeString},
		},
		Mode: schema.ModeStrict,
	}

	tuples := []*core.Tuple{
		core.NewTuple("myns/posts/abc", "title", "hello"),
	}

	vals := valuesFromTuples(def, tuples)

	require.Len(t, vals, 2)
	assert.Equal(t, "missing", vals[0].Name)
	assert.Nil(t, vals[0].Val)
	assert.Equal(t, "title", vals[1].Name)
	title, err := vals[1].Val.AsStr()
	require.NoError(t, err)
	assert.Equal(t, "hello", title)
}
