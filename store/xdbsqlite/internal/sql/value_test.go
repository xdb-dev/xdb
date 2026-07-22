package sql_test

import (
	"encoding/json"
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

func TestTypeFrom(t *testing.T) {
	tests := []struct {
		name string
		tid  string
		elem string
		want core.Type
	}{
		{"scalar", string(core.TIDString), "", core.TypeString},
		{"int", string(core.TIDInteger), "", core.TypeInt},
		{"array of ints", string(core.TIDArray), string(core.TIDInteger), core.NewArrayType(core.TIDInteger)},
		{"array of strings", string(core.TIDArray), string(core.TIDString), core.NewArrayType(core.TIDString)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xsql.TypeFrom(tt.tid, tt.elem)
			assert.Equal(t, tt.want.ID(), got.ID())
			assert.Equal(t, tt.want.ElemTypeID(), got.ElemTypeID())
		})
	}
}

func TestValue_ElemTID(t *testing.T) {
	tests := []struct {
		name string
		val  *core.Value
		want string
	}{
		{"nil", nil, ""},
		{"scalar", core.IntVal(1), ""},
		{"array of ints", core.ArrayVal(core.TIDInteger, core.IntVal(1)), string(core.TIDInteger)},
		{"array of strings", core.ArrayVal(core.TIDString, core.StringVal("a")), string(core.TIDString)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, xsql.Value{Val: tt.val}.ElemTID())
		})
	}
}
