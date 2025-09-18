package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/core"
)

func TestNewValue_Primitives(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected core.TypeID
	}{
		{
			name:     "string",
			value:    "hello",
			expected: core.TypeIDString,
		},
		{
			name:     "int",
			value:    1,
			expected: core.TypeIDInteger,
		},
		{
			name:     "float",
			value:    1.0,
			expected: core.TypeIDFloat,
		},
		{
			name:     "bool",
			value:    true,
			expected: core.TypeIDBoolean,
		},
		{
			name:     "bytes",
			value:    []byte("hello"),
			expected: core.TypeIDBytes,
		},
		{
			name:     "time",
			value:    time.Now(),
			expected: core.TypeIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := core.NewValue(tc.value)

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
		expected core.TypeID
	}{
		{
			name:     "string",
			value:    []string{"hello", "world"},
			expected: core.TypeIDString,
		},
		{
			name:     "int",
			value:    []int64{1, 2, 3},
			expected: core.TypeIDInteger,
		},
		{
			name:     "float",
			value:    []float64{1.0, 2.0, 3.0},
			expected: core.TypeIDFloat,
		},
		{
			name:     "bool",
			value:    []bool{true, false, true},
			expected: core.TypeIDBoolean,
		},
		{
			name:     "bytes",
			value:    [][]byte{[]byte("hello"), []byte("world")},
			expected: core.TypeIDBytes,
		},
		{
			name:     "time",
			value:    []time.Time{time.Now(), time.Now().Add(time.Hour)},
			expected: core.TypeIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := core.NewValue(tc.value)

			at := value.Type()
			assert.Equal(t, tc.expected, at.ValueType())
			tests.AssertEqualValues(t, tc.value, value)
		})
	}
}
