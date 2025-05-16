package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

func TestNewValue(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected types.TypeID
		repeated bool
	}{
		{
			name:     "string",
			value:    "hello",
			expected: types.TypeString,
			repeated: false,
		},
		{
			name:     "string slice",
			value:    []string{"hello", "world"},
			expected: types.TypeString,
			repeated: true,
		},
		{
			name:     "int",
			value:    1,
			expected: types.TypeInteger,
			repeated: false,
		},
		{
			name:     "int slice",
			value:    []int{1, 2, 3},
			expected: types.TypeInteger,
			repeated: true,
		},
		{
			name:     "float",
			value:    1.0,
			expected: types.TypeFloat,
			repeated: false,
		},
		{
			name:     "float slice",
			value:    []float64{1.0, 2.0, 3.0},
			expected: types.TypeFloat,
			repeated: true,
		},
		{
			name:     "bool",
			value:    true,
			expected: types.TypeBoolean,
			repeated: false,
		},
		{
			name:     "bool slice",
			value:    []bool{true, false, true},
			expected: types.TypeBoolean,
			repeated: true,
		},
		{
			name:     "bytes",
			value:    []byte("hello"),
			expected: types.TypeBytes,
			repeated: false,
		},
		{
			name:     "bytes slice",
			value:    [][]byte{[]byte("hello"), []byte("world")},
			expected: types.TypeBytes,
			repeated: true,
		},
		{
			name:     "time",
			value:    time.Now(),
			expected: types.TypeTime,
			repeated: false,
		},
		{
			name:     "time slice",
			value:    []time.Time{time.Now(), time.Now().Add(time.Hour)},
			expected: types.TypeTime,
			repeated: true,
		},
		{
			name:     "point",
			value:    types.Point{Lat: 1.0, Long: 2.0},
			expected: types.TypePoint,
			repeated: false,
		},
		{
			name: "point slice",
			value: []types.Point{
				{Lat: 1.0, Long: 2.0},
				{Lat: 3.0, Long: 4.0},
			},
			expected: types.TypePoint,
			repeated: true,
		},
		{
			name:     "unknown",
			value:    struct{ A int }{A: 1},
			expected: types.TypeUnknown,
			repeated: false,
		},
		{
			name:     "map",
			value:    map[string]any{"a": 1},
			expected: types.TypeUnknown,
			repeated: false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			value := types.NewValue(testcase.value)
			assert.Equal(t, testcase.expected, value.TypeID())
			assert.Equal(t, testcase.repeated, value.Repeated())
			assert.EqualValues(t, testcase.value, value.Unwrap())
		})
	}
}
