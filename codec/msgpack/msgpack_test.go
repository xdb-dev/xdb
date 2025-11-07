package msgpack_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/codec/msgpack"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/tests"
)

func TestMsgpackCodec(t *testing.T) {
	t.Parallel()

	repo := "test.repo"
	id := core.NewID("123")

	testCases := []struct {
		name  string
		value any
		uri   string
	}{
		{
			name:  "boolean",
			value: true,
			uri:   "xdb://test.repo/123#boolean",
		},
		{
			name:  "integer",
			value: int64(42),
			uri:   "xdb://test.repo/123#integer",
		},
		{
			name:  "string",
			value: "hello world",
			uri:   "xdb://test.repo/123#string",
		},
		{
			name:  "float",
			value: float64(3.14),
			uri:   "xdb://test.repo/123#float",
		},
		{
			name:  "uint64",
			value: uint64(123),
			uri:   "xdb://test.repo/123#uint64",
		},
		{
			name:  "bytes",
			value: []byte("hello world"),
			uri:   "xdb://test.repo/123#bytes",
		},
		{
			name:  "time",
			value: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			uri:   "xdb://test.repo/123#time",
		},
		{
			name:  "boolean_array",
			value: []bool{true, false, true},
			uri:   "xdb://test.repo/123#boolean_array",
		},
		{
			name:  "integer_array",
			value: []int64{1, 2, 3},
			uri:   "xdb://test.repo/123#integer_array",
		},
		{
			name:  "unsigned_array",
			value: []uint64{1, 2, 3},
			uri:   "xdb://test.repo/123#unsigned_array",
		},
		{
			name:  "float_array",
			value: []float64{1.1, 2.2, 3.3},
			uri:   "xdb://test.repo/123#float_array",
		},
		{
			name:  "string_array",
			value: []string{"value1", "value2", "value3"},
			uri:   "xdb://test.repo/123#string_array",
		},
		{
			name:  "bytes_array",
			value: [][]byte{[]byte("hello"), []byte("world")},
			uri:   "xdb://test.repo/123#bytes_array",
		},
		{
			name: "time_array",
			value: []time.Time{
				time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			uri: "xdb://test.repo/123#time_array",
		},
		{
			name:  "string_map",
			value: map[string]string{"key1": "value1", "key2": "value2"},
			uri:   "xdb://test.repo/123#string_map",
		},
		{
			name:  "integer_map",
			value: map[string]int64{"a": 1, "b": 2, "c": 3},
			uri:   "xdb://test.repo/123#integer_map",
		},
		{
			name:  "boolean_map",
			value: map[string]bool{"flag1": true, "flag2": false},
			uri:   "xdb://test.repo/123#boolean_map",
		},
		{
			name:  "float_map",
			value: map[string]float64{"x": 1.1, "y": 2.2},
			uri:   "xdb://test.repo/123#float_map",
		},
		{
			name:  "int_key_map",
			value: map[int64]string{1: "one", 2: "two", 3: "three"},
			uri:   "xdb://test.repo/123#int_key_map",
		},
	}

	codec := msgpack.New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tuple := core.NewTuple(repo, id, tc.name, tc.value)

			encodedURI, err := codec.EncodeURI(tuple.URI())
			require.NoError(t, err)
			assert.Equal(t, tc.uri, string(encodedURI))

			encodedValue, err := codec.EncodeValue(tuple.Value())
			require.NoError(t, err)

			decodedURI, err := codec.DecodeURI(encodedURI)
			require.NoError(t, err)
			tests.AssertEqualURI(t, tuple.URI(), decodedURI)

			decodedValue, err := codec.DecodeValue(encodedValue)
			require.NoError(t, err)
			tests.AssertEqualValues(t, tuple.Value(), decodedValue)
		})
	}
}
