package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
)

func TestNewValue_Primitives(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected types.TypeID
	}{
		{
			name:     "string",
			value:    "hello",
			expected: types.TypeIDString,
		},
		{
			name:     "int",
			value:    1,
			expected: types.TypeIDInteger,
		},
		{
			name:     "float",
			value:    1.0,
			expected: types.TypeIDFloat,
		},
		{
			name:     "bool",
			value:    true,
			expected: types.TypeIDBoolean,
		},
		{
			name:     "bytes",
			value:    []byte("hello"),
			expected: types.TypeIDBytes,
		},
		{
			name:     "time",
			value:    time.Now(),
			expected: types.TypeIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := types.NewValue(tc.value)

			got := value.Type().ID()
			assert.Equal(t, tc.expected, got)
			tests.AssertEqualValues(t, tc.value, value)
		})
	}
}

func TestNewValue_Arrays(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected types.TypeID
	}{
		{
			name:     "string",
			value:    []string{"hello", "world"},
			expected: types.TypeIDString,
		},
		{
			name:     "int",
			value:    []int64{1, 2, 3},
			expected: types.TypeIDInteger,
		},
		{
			name:     "float",
			value:    []float64{1.0, 2.0, 3.0},
			expected: types.TypeIDFloat,
		},
		{
			name:     "bool",
			value:    []bool{true, false, true},
			expected: types.TypeIDBoolean,
		},
		{
			name:     "bytes",
			value:    [][]byte{[]byte("hello"), []byte("world")},
			expected: types.TypeIDBytes,
		},
		{
			name:     "time",
			value:    []time.Time{time.Now(), time.Now().Add(time.Hour)},
			expected: types.TypeIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := types.NewValue(tc.value)

			at := value.Type()
			assert.Equal(t, tc.expected, at.ValueType())
			tests.AssertEqualValues(t, tc.value, value)
		})
	}
}
