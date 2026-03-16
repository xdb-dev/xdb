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
	uri := core.New().NS("myns").Schema("posts").ID("abc").MustURI()
	assert.Equal(t, `"kv:myns/posts"`, kvTableName(uri))
}

func TestColumnTableName(t *testing.T) {
	uri := core.New().NS("myns").Schema("posts").ID("abc").MustURI()
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
	uri := core.New().NS("myns").Schema("posts").MustURI()
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.FieldDef{
			"title":  {Type: core.TIDString},
			"count":  {Type: core.TIDInteger},
			"active": {Type: core.TIDBoolean},
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
	uri := core.New().NS("myns").Schema("posts").MustURI()
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.FieldDef{
			"zebra": {Type: core.TIDString},
			"alpha": {Type: core.TIDString},
			"mid":   {Type: core.TIDString},
		},
	}

	cols := sortedColumns(def)
	require.Equal(t, []string{"alpha", "mid", "zebra"}, cols)
}

func TestRecordToValues(t *testing.T) {
	uri := core.New().NS("myns").Schema("posts").MustURI()
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
			"count": {Type: core.TIDInteger},
		},
		Mode: schema.ModeStrict,
	}

	record := core.NewRecord("myns", "posts", "abc")
	record.Set("title", "hello")
	record.Set("count", 42)

	vals := recordToValues(def, record)

	require.Len(t, vals, 2)
	assert.Equal(t, "count", vals[0].Name)
	assert.Equal(t, int64(42), vals[0].Val.Unwrap())
	assert.Equal(t, "title", vals[1].Name)
	assert.Equal(t, "hello", vals[1].Val.Unwrap())
}

func TestRecordToValues_MissingColumn(t *testing.T) {
	uri := core.New().NS("myns").Schema("posts").MustURI()
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.FieldDef{
			"title":   {Type: core.TIDString},
			"missing": {Type: core.TIDString},
		},
		Mode: schema.ModeStrict,
	}

	record := core.NewRecord("myns", "posts", "abc")
	record.Set("title", "hello")

	vals := recordToValues(def, record)

	require.Len(t, vals, 2)
	assert.Equal(t, "missing", vals[0].Name)
	assert.Nil(t, vals[0].Val)
	assert.Equal(t, "title", vals[1].Name)
	assert.Equal(t, "hello", vals[1].Val.Unwrap())
}
