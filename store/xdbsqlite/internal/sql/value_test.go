package sql_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

var testTime = time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)

func TestValue_Value(t *testing.T) {
	tests := []struct {
		name string
		val  *core.Value
		want any
	}{
		{"nil", nil, nil},
		{"string", core.StringVal("hello"), "hello"},
		{"int", core.IntVal(42), int64(42)},
		{"float", core.FloatVal(3.14), float64(3.14)},
		{"bool true", core.BoolVal(true), int64(1)},
		{"bool false", core.BoolVal(false), int64(0)},
		{"unsigned", core.UintVal(99), int64(99)},
		{"time", core.TimeVal(testTime), testTime.UnixMilli()},
		{"json", core.JSONVal(json.RawMessage(`{"a":1}`)), `{"a":1}`},
		{"bytes", core.BytesVal([]byte{0xDE, 0xAD}), []byte{0xDE, 0xAD}},
		{
			"array of strings",
			core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b")),
			`["a","b"]`,
		},
		{
			"array of ints",
			core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2)),
			`[1,2]`,
		},
		{"empty array", core.ArrayVal(core.TIDString), `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := xsql.Value{Val: tt.val}
			got, err := v.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValue_Scan(t *testing.T) {
	tests := []struct {
		name    string
		typ     core.Type
		src     any
		wantVal *core.Value
	}{
		{"nil src", core.TypeString, nil, nil},
		{"string", core.TypeString, "hello", core.StringVal("hello")},
		{"int from int64", core.TypeInt, int64(42), core.IntVal(42)},
		{"int from string", core.TypeInt, "99", core.IntVal(99)},
		{"float from float64", core.TypeFloat, float64(3.14), core.FloatVal(3.14)},
		{"float from int64", core.TypeFloat, int64(3), core.FloatVal(3)},
		{"float from string", core.TypeFloat, "2.5", core.FloatVal(2.5)},
		{"bool from int64 true", core.TypeBool, int64(1), core.BoolVal(true)},
		{"bool from int64 false", core.TypeBool, int64(0), core.BoolVal(false)},
		{"bool from native", core.TypeBool, true, core.BoolVal(true)},
		{"bool from string", core.TypeBool, "true", core.BoolVal(true)},
		{"unsigned from int64", core.TypeUnsigned, int64(7), core.UintVal(7)},
		{"unsigned from string", core.TypeUnsigned, "42", core.UintVal(42)},
		{"time from int64", core.TypeTime, testTime.UnixMilli(), core.TimeVal(testTime)},
		{"time from time.Time", core.TypeTime, testTime, core.TimeVal(testTime)},
		{"json from string", core.TypeJSON, `{"a":1}`, core.JSONVal(json.RawMessage(`{"a":1}`))},
		{"json from bytes", core.TypeJSON, []byte(`{"a":1}`), core.JSONVal(json.RawMessage(`{"a":1}`))},
		{"bytes", core.TypeBytes, []byte{0xCA, 0xFE}, core.BytesVal([]byte{0xCA, 0xFE})},
		{
			"array of strings from string",
			core.NewArrayType(core.TIDString),
			`["a","b"]`,
			core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b")),
		},
		{
			"array of ints from string",
			core.NewArrayType(core.TIDInteger),
			`[1,2]`,
			core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2)),
		},
		{
			"array of strings from bytes",
			core.NewArrayType(core.TIDString),
			[]byte(`["x"]`),
			core.ArrayVal(core.TIDString, core.StringVal("x")),
		},
		{
			"empty array",
			core.NewArrayType(core.TIDString),
			`[]`,
			core.ArrayVal(core.TIDString),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := xsql.Value{Type: tt.typ}
			require.NoError(t, v.Scan(tt.src))
			if tt.wantVal == nil {
				assert.Nil(t, v.Val)
			} else {
				require.NotNil(t, v.Val)
				assertValueEqual(t, tt.wantVal, v.Val)
			}
		})
	}
}

// assertValueEqual compares two Values, treating nil and empty slices as equal for arrays.
func assertValueEqual(t *testing.T, want, got *core.Value) {
	t.Helper()
	if want.Type().ID() == core.TIDArray {
		wantArr, _ := want.AsArray()
		gotArr, _ := got.AsArray()
		assert.Equal(t, len(wantArr), len(gotArr))
		for i := range wantArr {
			assert.Equal(t, wantArr[i].Unwrap(), gotArr[i].Unwrap())
		}
		return
	}
	assert.Equal(t, want.Unwrap(), got.Unwrap())
}

func TestValue_MarshalBytes(t *testing.T) {
	tests := []struct {
		name string
		val  *core.Value
		want string
	}{
		{"nil", nil, ""},
		{"string", core.StringVal("hello"), "hello"},
		{"int", core.IntVal(42), "42"},
		{"unsigned", core.UintVal(99), "99"},
		{"float", core.FloatVal(3.14), "3.14"},
		{"bool true", core.BoolVal(true), "true"},
		{"bool false", core.BoolVal(false), "false"},
		{"time", core.TimeVal(testTime), fmt.Sprintf("%d", testTime.UnixMilli())},
		{"json", core.JSONVal(json.RawMessage(`{"a":1}`)), `{"a":1}`},
		{
			"array of strings",
			core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b")),
			`["a","b"]`,
		},
		{
			"array of ints",
			core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2)),
			`[1,2]`,
		},
		{"empty array", core.ArrayVal(core.TIDString), `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := xsql.Value{Val: tt.val}
			got, err := v.MarshalBytes()
			require.NoError(t, err)
			if tt.val == nil {
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}

func TestValue_UnmarshalBytes(t *testing.T) {
	tests := []struct {
		name    string
		typ     core.Type
		data    string
		wantVal *core.Value
	}{
		{"string", core.TypeString, "hello", core.StringVal("hello")},
		{"int", core.TypeInt, "42", core.IntVal(42)},
		{"unsigned", core.TypeUnsigned, "99", core.UintVal(99)},
		{"float", core.TypeFloat, "3.14", core.FloatVal(3.14)},
		{"bool true", core.TypeBool, "true", core.BoolVal(true)},
		{"bool false", core.TypeBool, "false", core.BoolVal(false)},
		{"time", core.TypeTime, fmt.Sprintf("%d", testTime.UnixMilli()), core.TimeVal(testTime)},
		{"json", core.TypeJSON, `{"a":1}`, core.JSONVal(json.RawMessage(`{"a":1}`))},
		{"bytes", core.TypeBytes, "raw", core.BytesVal([]byte("raw"))},
		{
			"array of strings",
			core.NewArrayType(core.TIDString),
			`["a","b"]`,
			core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b")),
		},
		{
			"array of ints",
			core.NewArrayType(core.TIDInteger),
			`[1,2]`,
			core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2)),
		},
		{
			"empty array",
			core.NewArrayType(core.TIDString),
			`[]`,
			core.ArrayVal(core.TIDString),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v xsql.Value
			require.NoError(t, v.UnmarshalBytes(tt.typ, []byte(tt.data)))
			require.NotNil(t, v.Val)
			assertValueEqual(t, tt.wantVal, v.Val)
		})
	}
}

func TestValue_MarshalUnmarshalBytes_Roundtrip(t *testing.T) {
	tests := []struct {
		name string
		typ  core.Type
		val  *core.Value
	}{
		{"string", core.TypeString, core.StringVal("hello")},
		{"int", core.TypeInt, core.IntVal(-42)},
		{"unsigned", core.TypeUnsigned, core.UintVal(999)},
		{"float", core.TypeFloat, core.FloatVal(2.718)},
		{"bool", core.TypeBool, core.BoolVal(true)},
		{"time", core.TypeTime, core.TimeVal(testTime)},
		{"json", core.TypeJSON, core.JSONVal(json.RawMessage(`[1,2,3]`))},
		{"bytes", core.TypeBytes, core.BytesVal([]byte{0x01, 0x02})},
		{
			"array of strings",
			core.NewArrayType(core.TIDString),
			core.ArrayVal(core.TIDString, core.StringVal("x"), core.StringVal("y")),
		},
		{
			"array of ints",
			core.NewArrayType(core.TIDInteger),
			core.ArrayVal(core.TIDInteger, core.IntVal(10), core.IntVal(20)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := xsql.Value{Val: tt.val}
			data, err := orig.MarshalBytes()
			require.NoError(t, err)

			var restored xsql.Value
			require.NoError(t, restored.UnmarshalBytes(tt.typ, data))
			assertValueEqual(t, tt.val, restored.Val)
		})
	}
}
