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

func TestEncodeDecodeTuple(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		value   any
		flatkey string
	}{
		{
			name:    "boolean",
			value:   true,
			flatkey: "Test:1:boolean",
		},
		{
			name:    "integer",
			value:   int64(42),
			flatkey: "Test:1:integer",
		},
		{
			name:    "string",
			value:   "hello world",
			flatkey: "Test:1:string",
		},
		{
			name:    "float",
			value:   float64(3.14),
			flatkey: "Test:1:float",
		},
		{
			name:    "uint64",
			value:   uint64(123),
			flatkey: "Test:1:uint64",
		},
		{
			name:    "bytes",
			value:   []byte("hello world"),
			flatkey: "Test:1:bytes",
		},
		{
			name:    "time",
			value:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			flatkey: "Test:1:time",
		},
		{
			name:    "boolean_array",
			value:   []bool{true, false, true},
			flatkey: "Test:1:boolean_array",
		},
		{
			name:    "integer_array",
			value:   []int64{1, 2, 3},
			flatkey: "Test:1:integer_array",
		},
		{
			name:    "unsigned_array",
			value:   []uint64{1, 2, 3},
			flatkey: "Test:1:unsigned_array",
		},
		{
			name:    "float_array",
			value:   []float64{1.1, 2.2, 3.3},
			flatkey: "Test:1:float_array",
		},
		{
			name:    "string_array",
			value:   []string{"value1", "value2", "value3"},
			flatkey: "Test:1:string_array",
		},
		{
			name:    "bytes_array",
			value:   [][]byte{[]byte("hello"), []byte("world")},
			flatkey: "Test:1:bytes_array",
		},
		{
			name: "time_array",
			value: []time.Time{
				time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			flatkey: "Test:1:time_array",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tuple := types.NewTuple("Test", "1", tc.name, tc.value)

			encodedKey, encodedValue, err := xdbkv.EncodeTuple(tuple)
			require.NoError(t, err)
			assert.Equal(t, tc.flatkey, string(encodedKey))

			decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
			require.NoError(t, err)
			tests.AssertEqualTuple(t, tuple, decodedTuple)
		})
	}
}
