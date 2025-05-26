package xdbkv_test

import (
	"testing"

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
			name:    "String",
			tuple:   types.NewTuple("Test", "1", "string", "value"),
			flatkey: "Test:1:string",
		},
		{
			name:    "String Array",
			tuple:   types.NewTuple("Test", "1", "string_array", []string{"value1", "value2"}),
			flatkey: "Test:1:string_array",
		},
		{
			name:    "Integer",
			tuple:   types.NewTuple("Test", "1", "integer", int64(42)),
			flatkey: "Test:1:integer",
		},
		{
			name:    "Integer Array",
			tuple:   types.NewTuple("Test", "1", "integer_array", []int64{1, 2}),
			flatkey: "Test:1:integer_array",
		},
		{
			name:    "Float",
			tuple:   types.NewTuple("Test", "1", "float", 3.14),
			flatkey: "Test:1:float",
		},
		{
			name:    "Float Array",
			tuple:   types.NewTuple("Test", "1", "float_array", []float64{1.1, 2.2}),
			flatkey: "Test:1:float_array",
		},
		{
			name:    "Boolean",
			tuple:   types.NewTuple("Test", "1", "boolean", true),
			flatkey: "Test:1:boolean",
		},
		{
			name:    "Boolean Array",
			tuple:   types.NewTuple("Test", "1", "boolean_array", []bool{true, false, true}),
			flatkey: "Test:1:boolean_array",
		},
		{
			name:    "Bytes",
			tuple:   types.NewTuple("Test", "1", "bytes", []byte("hello world")),
			flatkey: "Test:1:bytes",
		},
		{
			name:    "Bytes Array",
			tuple:   types.NewTuple("Test", "1", "bytes_array", [][]byte{[]byte("hello"), []byte("world")}),
			flatkey: "Test:1:bytes_array",
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
