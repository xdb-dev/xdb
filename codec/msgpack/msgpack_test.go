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

	testCases := []struct {
		name     string
		value    any
		flatid   string
		flatattr string
	}{
		{
			name:     "boolean",
			value:    true,
			flatid:   "Test/1",
			flatattr: "boolean",
		},
		{
			name:     "integer",
			value:    int64(42),
			flatid:   "Test/1",
			flatattr: "integer",
		},
		{
			name:     "string",
			value:    "hello world",
			flatid:   "Test/1",
			flatattr: "string",
		},
		{
			name:     "float",
			value:    float64(3.14),
			flatid:   "Test/1",
			flatattr: "float",
		},
		{
			name:     "uint64",
			value:    uint64(123),
			flatid:   "Test/1",
			flatattr: "uint64",
		},
		{
			name:     "bytes",
			value:    []byte("hello world"),
			flatid:   "Test/1",
			flatattr: "bytes",
		},
		{
			name:     "time",
			value:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			flatid:   "Test/1",
			flatattr: "time",
		},
		{
			name:     "boolean_array",
			value:    []bool{true, false, true},
			flatid:   "Test/1",
			flatattr: "boolean_array",
		},
		{
			name:     "integer_array",
			value:    []int64{1, 2, 3},
			flatid:   "Test/1",
			flatattr: "integer_array",
		},
		{
			name:     "unsigned_array",
			value:    []uint64{1, 2, 3},
			flatid:   "Test/1",
			flatattr: "unsigned_array",
		},
		{
			name:     "float_array",
			value:    []float64{1.1, 2.2, 3.3},
			flatid:   "Test/1",
			flatattr: "float_array",
		},
		{
			name:     "string_array",
			value:    []string{"value1", "value2", "value3"},
			flatid:   "Test/1",
			flatattr: "string_array",
		},
		{
			name:     "bytes_array",
			value:    [][]byte{[]byte("hello"), []byte("world")},
			flatid:   "Test/1",
			flatattr: "bytes_array",
		},
		{
			name: "time_array",
			value: []time.Time{
				time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			flatid:   "Test/1",
			flatattr: "time_array",
		},
	}

	codec := msgpack.New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := core.NewID("Test", "1")
			attr := core.NewAttr(tc.name)

			tuple := core.NewTuple(id, attr, tc.value)

			encodedID, err := codec.EncodeID(tuple.ID())
			require.NoError(t, err)
			assert.Equal(t, tc.flatid, string(encodedID))

			encodedAttr, err := codec.EncodeAttr(tuple.Attr())
			require.NoError(t, err)
			assert.Equal(t, tc.flatattr, string(encodedAttr))

			encodedValue, err := codec.EncodeValue(tuple.Value())
			require.NoError(t, err)

			decodedID, err := codec.DecodeID(encodedID)
			require.NoError(t, err)
			assert.EqualValues(t, tuple.ID(), decodedID)

			decodedAttr, err := codec.DecodeAttr(encodedAttr)
			require.NoError(t, err)
			assert.EqualValues(t, tuple.Attr(), decodedAttr)

			decodedValue, err := codec.DecodeValue(encodedValue)
			require.NoError(t, err)
			tests.AssertEqualValues(t, tuple.Value(), decodedValue)
		})
	}
}
