package sql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func createKVTable(t *testing.T, q *xsql.Queries, table string) {
	t.Helper()
	require.NoError(t, q.CreateKVTable(context.Background(), xsql.CreateKVTableParams{Table: table}))
}

func TestCreateKVRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	err := q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table,
		ID:    "id1",
		Values: []xsql.Value{
			{Name: "name", Val: core.StringVal("alice")},
			{Name: "age", Val: core.IntVal(30)},
		},
	})
	require.NoError(t, err)

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	require.Len(t, vals, 2)

	// Sorted by _attr: age, name
	assert.Equal(t, "age", vals[0].Name)
	assert.Equal(t, int64(30), vals[0].Val.Unwrap())
	assert.Equal(t, core.TIDInteger, vals[0].Val.Type().ID())

	assert.Equal(t, "name", vals[1].Name)
	assert.Equal(t, "alice", vals[1].Val.Unwrap())
	assert.Equal(t, core.TIDString, vals[1].Val.Type().ID())
}

func TestCreateKVRecord_ReplaceSemantics(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "v", Val: core.IntVal(1)}},
	}))

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "v", Val: core.IntVal(2)}},
	}))

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	require.Len(t, vals, 1)
	assert.Equal(t, int64(2), vals[0].Val.Unwrap())
}

func TestCreateKVRecord_EmptyValues(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "v", Val: core.IntVal(1)}},
	}))

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1", Values: nil,
	}))

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestGetKVRecord_Missing(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "nope"})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestListKVRecordIDs(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	for _, id := range []string{"c", "a", "b"} {
		require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
			Table: table, ID: id,
			Values: []xsql.Value{{Name: "x", Val: core.StringVal("v")}},
		}))
	}

	ids, err := q.ListKVRecordIDs(ctx, xsql.ListKVRecordIDsParams{Table: table, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, ids)
}

func TestListKVRecords(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{
			{Name: "a", Val: core.StringVal("1")},
			{Name: "b", Val: core.IntVal(2)},
		},
	}))
	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id2",
		Values: []xsql.Value{
			{Name: "c", Val: core.BoolVal(true)},
		},
	}))

	records, err := q.ListKVRecords(ctx, xsql.ListKVRecordsParams{Table: table, Limit: 100})
	require.NoError(t, err)
	require.Len(t, records, 2)

	// Ordered by ID.
	assert.Equal(t, "id1", records[0].ID)
	require.Len(t, records[0].Values, 2)
	assert.Equal(t, "a", records[0].Values[0].Name)
	assert.Equal(t, "b", records[0].Values[1].Name)

	assert.Equal(t, "id2", records[1].ID)
	require.Len(t, records[1].Values, 1)
	assert.Equal(t, "c", records[1].Values[0].Name)
	assert.Equal(t, true, records[1].Values[0].Val.Unwrap())
}

func TestDeleteKVRecord(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "x", Val: core.StringVal("v")}},
	}))

	require.NoError(t, q.DeleteKVRecord(ctx, xsql.DeleteKVRecordParams{Table: table, ID: "id1"}))

	vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.Nil(t, vals)
}

func TestKVRecordExists(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	exists, err := q.KVRecordExists(ctx, xsql.KVRecordExistsParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table: table, ID: "id1",
		Values: []xsql.Value{{Name: "x", Val: core.StringVal("v")}},
	}))

	exists, err = q.KVRecordExists(ctx, xsql.KVRecordExistsParams{Table: table, ID: "id1"})
	require.NoError(t, err)
	assert.True(t, exists)
}
