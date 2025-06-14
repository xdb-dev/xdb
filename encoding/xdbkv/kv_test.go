package xdbkv_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
)

func TestTuple_X_KV(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		tuple   *types.Tuple
		flatkey string
	}{
		{
			name:    "Boolean",
			tuple:   types.NewTuple("Test", "1", "boolean", true),
			flatkey: "Test:1:boolean",
		},
		{
			name:    "Integer",
			tuple:   types.NewTuple("Test", "1", "integer", int64(42)),
			flatkey: "Test:1:integer",
		},
		{
			name:    "String",
			tuple:   types.NewTuple("Test", "1", "string", "value"),
			flatkey: "Test:1:string",
		},
		{
			name:    "Float",
			tuple:   types.NewTuple("Test", "1", "float", float64(3.14)),
			flatkey: "Test:1:float",
		},
		{
			name:    "Uint64",
			tuple:   types.NewTuple("Test", "1", "uint64", uint64(123)),
			flatkey: "Test:1:uint64",
		},
		{
			name:    "Bytes",
			tuple:   types.NewTuple("Test", "1", "bytes", []byte("hello world")),
			flatkey: "Test:1:bytes",
		},
		{
			name:    "Time",
			tuple:   types.NewTuple("Test", "1", "time", types.Time(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))),
			flatkey: "Test:1:time",
		},
		{
			name:    "Boolean Array",
			tuple:   types.NewTuple("Test", "1", "boolean_array", []bool{true, false, true}),
			flatkey: "Test:1:boolean_array",
		},
		{
			name:    "Integer Array",
			tuple:   types.NewTuple("Test", "1", "integer_array", []int64{1, 2, 3}),
			flatkey: "Test:1:integer_array",
		},
		{
			name:    "Unsigned Array",
			tuple:   types.NewTuple("Test", "1", "unsigned_array", []uint64{1, 2, 3}),
			flatkey: "Test:1:unsigned_array",
		},
		{
			name:    "Float Array",
			tuple:   types.NewTuple("Test", "1", "float_array", []float64{1.1, 2.2, 3.3}),
			flatkey: "Test:1:float_array",
		},
		{
			name:    "String Array",
			tuple:   types.NewTuple("Test", "1", "string_array", []string{"value1", "value2", "value3"}),
			flatkey: "Test:1:string_array",
		},
		{
			name:    "Bytes Array",
			tuple:   types.NewTuple("Test", "1", "bytes_array", [][]byte{[]byte("hello"), []byte("world")}),
			flatkey: "Test:1:bytes_array",
		},
		{
			name:    "Time Array",
			tuple:   types.NewTuple("Test", "1", "time_array", []types.Time{types.Time(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)), types.Time(time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))}),
			flatkey: "Test:1:time_array",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encodedKey, encodedValue, err := xdbkv.EncodeTuple(tc.tuple)
			require.NoError(t, err)
			assert.Equal(t, tc.flatkey, string(encodedKey))

			decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
			require.NoError(t, err)
			tests.AssertEqualTuple(t, tc.tuple, decodedTuple)
		})
	}
}
