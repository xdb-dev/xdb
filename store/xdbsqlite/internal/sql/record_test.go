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

func TestUpsertRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	titleCol := []xsql.Value{{Name: "title", Type: core.TypeString}}

	t.Run("creates when missing", func(t *testing.T) {
		err := q.UpsertRecord(ctx, xsql.UpsertRecordParams{
			Table: table, ID: "id1",
			Values: []xsql.Value{{Name: "title", Val: core.StringVal("v1")}},
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
			Values: []xsql.Value{{Name: "title", Val: core.StringVal("v2")}},
		})
		require.NoError(t, err)

		vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
			Table: table, ID: "id1", Columns: titleCol,
		})
		require.NoError(t, err)
		assert.Equal(t, "v2", vals[0].Val.Unwrap())
	})
}

func TestUpsertRecord_ZeroColumns(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	// A record carrying only _id has no columns to update on conflict;
	// the upsert must not generate a malformed "DO UPDATE SET ".
	up := xsql.UpsertRecordParams{Table: table, ID: "id1"}
	require.NoError(t, q.UpsertRecord(ctx, up))
	require.NoError(t, q.UpsertRecord(ctx, up))

	exists, err := q.RecordExists(ctx, xsql.RecordExistsParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestGetRecord_Missing(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "nope",
		Columns: []xsql.Value{{Name: "title", Type: core.TypeString}},
	})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestGetRecord_NoTable(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	_, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: `"t:test/missing"`, ID: "id1",
		Columns: []xsql.Value{{Name: "title", Type: core.TypeString}},
	})
	assert.ErrorIs(t, err, xsql.ErrNoTable)
}

func TestListRecords(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"t:test/t"`
	createColumnTable(t, q, table)

	for _, id := range []string{"c", "a", "b"} {
		require.NoError(t, q.UpsertRecord(ctx, xsql.UpsertRecordParams{
			Table: table, ID: id,
			Values: []xsql.Value{{Name: "title", Val: core.StringVal("t-" + id)}},
		}))
	}

	cols := []xsql.Value{{Name: "title", Type: core.TypeString}}

	t.Run("ordered by _id", func(t *testing.T) {
		rows, err := q.ListRecords(ctx, xsql.ListRecordsParams{
			Table: table, Columns: cols, Limit: 100,
		})
		require.NoError(t, err)
		require.Len(t, rows, 3)

		assert.Equal(t, "_id", rows[0][0].Name)
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

	require.NoError(t, q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "title", Val: core.StringVal("x")}},
	}))

	require.NoError(t, q.DeleteRecord(ctx, xsql.DeleteRecordParams{Table: table, ID: "id1"}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.Value{{Name: "title", Type: core.TypeString}},
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

	require.NoError(t, q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "title", Val: core.StringVal("x")}},
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

	require.NoError(t, q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{
			{Name: "title"},
			{Name: "count", Val: core.IntVal(5)},
		},
	}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.Value{
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

	require.NoError(t, q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "active", Val: core.BoolVal(true)}},
	}))

	vals, err := q.GetRecord(ctx, xsql.GetRecordParams{
		Table: table, ID: "id1",
		Columns: []xsql.Value{{Name: "active", Type: core.TypeBool}},
	})
	require.NoError(t, err)
	assert.Equal(t, true, vals[0].Val.Unwrap())
}
