package sql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func createColumnTable(t *testing.T, q *xsql.Queries, table string) {
	t.Helper()
	require.NoError(t, q.CreateTable(context.Background(), xsql.CreateTableParams{
		Table: table,
		Columns: []xsql.Column{
			{Name: "title", Type: "TEXT"},
			{Name: "count", Type: "INTEGER"},
			{Name: "score", Type: "REAL"},
		},
	}))
}

var testColumns = []xsql.ColumnType{
	{Name: "title", Type: core.TypeString},
	{Name: "count", Type: core.TypeInt},
	{Name: "score", Type: core.TypeFloat},
}

func TestCreateRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	err := q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table,
		ID:    "id1",
		Values: []xsql.Value{
			{Column: "title", Val: core.StringVal("hello")},
			{Column: "count", Val: core.IntVal(42)},
			{Column: "score", Val: core.FloatVal(3.14)},
		},
	})
	require.NoError(t, err)

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1", Columns: testColumns,
	})
	require.NoError(t, err)
	require.Len(t, vals, 3)

	assert.Equal(t, "title", vals[0].Column)
	assert.Equal(t, "hello", vals[0].Val.Unwrap())
	assert.Equal(t, "count", vals[1].Column)
	assert.Equal(t, int64(42), vals[1].Val.Unwrap())
	assert.Equal(t, "score", vals[2].Column)
	assert.Equal(t, float64(3.14), vals[2].Val.Unwrap())
}

func TestCreateRecord_Duplicate(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	params := xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Column: "title", Val: core.StringVal("x")}},
	}

	require.NoError(t, q.CreateRecord(ctx, params))
	err := q.CreateRecord(ctx, params)
	assert.Error(t, err)
}

func TestUpdateRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{
			{Column: "title", Val: core.StringVal("old")},
			{Column: "count", Val: core.IntVal(1)},
		},
	}))

	cols := []xsql.ColumnType{
		{Name: "title", Type: core.TypeString},
		{Name: "count", Type: core.TypeInt},
	}

	err := q.UpdateRecord(ctx, xsql.UpdateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{
			{Column: "title", Val: core.StringVal("new")},
			{Column: "count", Val: core.IntVal(2)},
		},
	})
	require.NoError(t, err)

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1", Columns: cols,
	})
	require.NoError(t, err)
	assert.Equal(t, "new", vals[0].Val.Unwrap())
	assert.Equal(t, int64(2), vals[1].Val.Unwrap())
}

func TestUpsertRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	titleCol := []xsql.ColumnType{{Name: "title", Type: core.TypeString}}

	t.Run("creates when missing", func(t *testing.T) {
		err := q.UpsertRecord(ctx, xsql.UpsertRecordParams{
			Table: table, ID: "id1",
			Values: []xsql.Value{{Column: "title", Val: core.StringVal("v1")}},
		})
		require.NoError(t, err)

		vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
			Table: table, ID: "id1", Columns: titleCol,
		})
		require.NoError(t, err)
		assert.Equal(t, "v1", vals[0].Val.Unwrap())
	})

	t.Run("updates when present", func(t *testing.T) {
		err := q.UpsertRecord(ctx, xsql.UpsertRecordParams{
			Table: table, ID: "id1",
			Values: []xsql.Value{{Column: "title", Val: core.StringVal("v2")}},
		})
		require.NoError(t, err)

		vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
			Table: table, ID: "id1", Columns: titleCol,
		})
		require.NoError(t, err)
		assert.Equal(t, "v2", vals[0].Val.Unwrap())
	})
}

func TestGetRecord_Missing(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "nope",
		Columns: []xsql.ColumnType{{Name: "title", Type: core.TypeString}},
	})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestListRecords(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	for _, id := range []string{"c", "a", "b"} {
		require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
			Table: table, ID: id,
			Values: []xsql.Value{{Column: "title", Val: core.StringVal("t-" + id)}},
		}))
	}

	cols := []xsql.ColumnType{{Name: "title", Type: core.TypeString}}

	t.Run("ordered by _id", func(t *testing.T) {
		rows, err := q.ListRecords(ctx, xsql.ListRecordsParams{
			Table: table, Columns: cols, Limit: 100,
		})
		require.NoError(t, err)
		require.Len(t, rows, 3)

		assert.Equal(t, "_id", rows[0][0].Column)
		assert.Equal(t, "a", rows[0][0].Val.Unwrap())
		assert.Equal(t, "t-a", rows[0][1].Val.Unwrap())
		assert.Equal(t, "b", rows[1][0].Val.Unwrap())
		assert.Equal(t, "c", rows[2][0].Val.Unwrap())
	})

	t.Run("paginated", func(t *testing.T) {
		rows, err := q.ListRecords(ctx, xsql.ListRecordsParams{
			Table: table, Columns: cols, Limit: 2, Offset: 0,
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)

		rows, err = q.ListRecords(ctx, xsql.ListRecordsParams{
			Table: table, Columns: cols, Limit: 2, Offset: 2,
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
	})
}

func TestDeleteRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Column: "title", Val: core.StringVal("x")}},
	}))

	require.NoError(t, q.DeleteRecord(ctx, xsql.DeleteRecordParams{Table: table, ID: "id1"}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.ColumnType{{Name: "title", Type: core.TypeString}},
	})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestRecordExists(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	exists, err := q.RecordExists(ctx, xsql.RecordExistsParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Column: "title", Val: core.StringVal("x")}},
	}))

	exists, err = q.RecordExists(ctx, xsql.RecordExistsParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRecord_NullValue(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{
			{Column: "title"},
			{Column: "count", Val: core.IntVal(5)},
		},
	}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.ColumnType{
			{Name: "title", Type: core.TypeString},
			{Name: "count", Type: core.TypeInt},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, vals[0].Val)
	assert.Equal(t, int64(5), vals[1].Val.Unwrap())
}

func TestRecord_BoolCoercion(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/bool"`

	require.NoError(t, q.CreateTable(ctx, xsql.CreateTableParams{
		Table:   table,
		Columns: []xsql.Column{{Name: "active", Type: "INTEGER"}},
	}))

	require.NoError(t, q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Column: "active", Val: core.BoolVal(true)}},
	}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.ColumnType{{Name: "active", Type: core.TypeBool}},
	})
	require.NoError(t, err)
	assert.Equal(t, true, vals[0].Val.Unwrap())
}
