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

func TestCreateKVRecord_NativeRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		val  *core.Value
	}{
		{"string", core.StringVal("hello")},
		{"int", core.IntVal(-42)},
		{"unsigned", core.UintVal(999)},
		{"float", core.FloatVal(2.718)},
		{"bool", core.BoolVal(true)},
		{"time", core.TimeVal(testTime)},
		{"json", core.JSONVal([]byte(`{"a":1}`))},
		{"bytes", core.BytesVal([]byte{0x01, 0x02})},
		{"array of ints", core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2))},
		{"array of strings", core.ArrayVal(core.TIDString, core.StringVal("x"), core.StringVal("y"))},
		{"array of floats", core.ArrayVal(core.TIDFloat, core.FloatVal(1.5), core.FloatVal(2.5))},
		{"array of bools", core.ArrayVal(core.TIDBoolean, core.BoolVal(true), core.BoolVal(false))},
		{"array of unsigned", core.ArrayVal(core.TIDUnsigned, core.UintVal(7), core.UintVal(8))},
		{"array of times", core.ArrayVal(core.TIDTime, core.TimeVal(testTime))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, q := testDB(t)
			ctx := context.Background()
			table := `"kv:test/t"`
			createKVTable(t, q, table)

			require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
				Table: table, ID: "id1",
				Values: []xsql.Value{{Name: "v", Val: tt.val}},
			}))

			vals, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: table, ID: "id1"})
			require.NoError(t, err)
			require.Len(t, vals, 1)
			assert.Equal(t, tt.val.Type().ID(), vals[0].Val.Type().ID())
			assert.Equal(t, tt.val.Type().ElemTypeID(), vals[0].Val.Type().ElemTypeID())
			assert.Equal(t, tt.val.Unwrap(), vals[0].Val.Unwrap())
		})
	}
}

func TestGetKVRecord_NoTable(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	_, err := q.GetKVRecord(ctx, xsql.GetKVRecordParams{Table: `"kv:test/missing"`, ID: "id1"})
	assert.ErrorIs(t, err, xsql.ErrNoTable)
}

func TestKVRecord_NativeNumericOrdering(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()
	table := `"kv:test/t"`
	createKVTable(t, q, table)

	// 9 sorts after 10 lexically but before it numerically; native
	// INTEGER storage makes the SQL comparison numeric.
	for id, age := range map[string]int64{"a": 9, "b": 10} {
		require.NoError(t, q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
			Table: table, ID: id,
			Values: []xsql.Value{{Name: "age", Val: core.IntVal(age)}},
		}))
	}

	records, err := q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
		Table:     table,
		Where:     "_id IN (SELECT _id FROM " + table + " WHERE _attr = ? AND _val > ?)",
		WhereArgs: []any{"age", int64(9)},
		Limit:     100,
	})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "b", records[0].ID)
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
