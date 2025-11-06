package core_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

var id = core.NewID("Test", "1")

func TestValue_ToBool(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected bool
	}{
		{
			name:     "bool true",
			value:    true,
			expected: true,
		},
		{
			name:     "bool false",
			value:    false,
			expected: false,
		},
		{
			name:     "string true",
			value:    "true",
			expected: true,
		},
		{
			name:     "int 1",
			value:    1,
			expected: true,
		},
		{
			name:     "int 0",
			value:    0,
			expected: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToBool())
			assert.Equal(t, tc.expected, tuple.Value().ToBool())
		})
	}
}

func TestValue_ToInt(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected int64
	}{
		{
			name:     "int",
			value:    1,
			expected: 1,
		},
		{
			name:     "string to int",
			value:    "123",
			expected: 123,
		},
		{
			name:     "float to int",
			value:    123.456,
			expected: 123,
		},
		{
			name:     "bool to int",
			value:    true,
			expected: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToInt())
			assert.Equal(t, tc.expected, tuple.Value().ToInt())
		})
	}
}

func TestValue_ToUint(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected uint64
	}{
		{
			name:     "uint",
			value:    1,
			expected: 1,
		},
		{
			name:     "string to uint",
			value:    "123",
			expected: 123,
		},

		{
			name:     "float to uint",
			value:    123.456,
			expected: 123,
		},
		{
			name:     "bool to uint",
			value:    true,
			expected: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToUint())
			assert.Equal(t, tc.expected, tuple.Value().ToUint())
		})
	}
}

func TestValue_ToFloat(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected float64
	}{
		{
			name:     "float",
			value:    123.456,
			expected: 123.456,
		},
		{
			name:     "string to float",
			value:    "123.456",
			expected: 123.456,
		},
		{
			name:     "bool to float",
			value:    true,
			expected: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToFloat())
			assert.Equal(t, tc.expected, tuple.Value().ToFloat())
		})
	}
}

func TestValue_ToString(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "string",
			value:    "hello",
			expected: "hello",
		},
		{
			name:     "int to string",
			value:    123,
			expected: "123",
		},
		{
			name:     "float to string",
			value:    123.456,
			expected: "123.456",
		},
		{
			name:     "bool to string",
			value:    true,
			expected: "true",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToString())
			assert.Equal(t, tc.expected, tuple.Value().ToString())
		})
	}
}

func TestValue_ToBytes(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected []byte
	}{
		{
			name:     "bytes",
			value:    []byte("hello"),
			expected: []byte("hello"),
		},
		{
			name:     "string to bytes",
			value:    "hello",
			expected: []byte("hello"),
		},
		{
			name:     "int to bytes",
			value:    123,
			expected: []byte("123"),
		},
		{
			name:     "float to bytes",
			value:    123.456,
			expected: []byte("123.456"),
		},
		{
			name:     "bool to bytes",
			value:    true,
			expected: []byte("true"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToBytes())
			assert.Equal(t, tc.expected, tuple.Value().ToBytes())
		})
	}
}

func TestValue_ToTime(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		value    any
		expected time.Time
	}{
		{
			name:     "time",
			value:    time.Date(2025, 6, 10, 1, 2, 3, 4, time.UTC),
			expected: time.Date(2025, 6, 10, 1, 2, 3, 4, time.UTC),
		},
		{
			name:     "string to time",
			value:    "2025-06-10T01:02:03.000Z",
			expected: time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC),
		},
		{
			name:     "bytes to time",
			value:    []byte("2025-06-10T01:02:03.000Z"),
			expected: time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC),
		},
		{
			name:     "int64 to time",
			value:    int64(1749517323000),
			expected: time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC),
		},
		{
			name:     "uint64 to time",
			value:    uint64(1749517323000),
			expected: time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC),
		},
		{
			name:     "float64 to time",
			value:    float64(1749517323000),
			expected: time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := core.NewTuple("test.repo", id, "attr", tc.value)
			assert.Equal(t, tc.expected.Compare(tuple.ToTime()), 0)
			assert.Equal(t, tc.expected.Compare(tuple.Value().ToTime()), 0)
		})
	}
}

func TestValue_Slices(t *testing.T) {
	t.Parallel()

	t.Run("IntArray", func(t *testing.T) {
		value := []int64{1, 2, 3}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToIntArray())
		assert.EqualValues(t, value, tuple.Value().ToIntArray())
	})

	t.Run("FloatArray", func(t *testing.T) {
		value := []float64{1.1, 2.2, 3.3}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToFloatArray())
		assert.EqualValues(t, value, tuple.Value().ToFloatArray())
	})

	t.Run("StringArray", func(t *testing.T) {
		value := []string{"a", "b", "c"}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToStringArray())
		assert.EqualValues(t, value, tuple.Value().ToStringArray())
	})

	t.Run("BoolArray", func(t *testing.T) {
		value := []bool{true, false, true}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToBoolArray())
		assert.EqualValues(t, value, tuple.Value().ToBoolArray())
	})

	t.Run("BytesArray", func(t *testing.T) {
		value := [][]byte{[]byte("a"), []byte("b")}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToBytesArray())
		assert.EqualValues(t, value, tuple.Value().ToBytesArray())
	})

	t.Run("UintArray", func(t *testing.T) {
		value := []uint64{1, 2, 3}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToUintArray())
		assert.EqualValues(t, value, tuple.Value().ToUintArray())
	})

	t.Run("TimeArray", func(t *testing.T) {
		t1 := time.Date(2025, 6, 10, 1, 2, 3, 0, time.UTC)
		t2 := time.Date(2026, 7, 11, 4, 5, 6, 0, time.UTC)
		value := []time.Time{t1, t2}
		tuple := core.NewTuple("test.repo", id, "attr", value)
		assert.EqualValues(t, value, tuple.ToTimeArray())
		assert.EqualValues(t, value, tuple.Value().ToTimeArray())
	})
}

func TestCast_InvalidConversions(t *testing.T) {
	t.Parallel()

	t.Run("String to Int - Invalid", func(t *testing.T) {
		value := core.NewValue("not-a-number")
		// The cast library is lenient and returns 0 for invalid strings
		assert.Equal(t, int64(0), value.ToInt())
	})

	t.Run("String to Float - Invalid", func(t *testing.T) {
		value := core.NewValue("not-a-float")
		// The cast library is lenient and returns 0 for invalid strings
		assert.Equal(t, float64(0), value.ToFloat())
	})

	t.Run("String to Uint - Invalid", func(t *testing.T) {
		value := core.NewValue("not-a-uint")
		// The cast library is lenient and returns 0 for invalid strings
		assert.Equal(t, uint64(0), value.ToUint())
	})

	t.Run("String to Bool - Invalid", func(t *testing.T) {
		value := core.NewValue("not-a-bool")
		// The cast library is lenient and returns false for invalid strings
		assert.Equal(t, false, value.ToBool())
	})

	t.Run("String to Time - Invalid", func(t *testing.T) {
		value := core.NewValue("not-a-time")
		// This should panic as time.Parse returns an error
		assert.Panics(t, func() {
			value.ToTime()
		})
	})
}

func TestCast_BoundaryValues(t *testing.T) {
	t.Parallel()

	t.Run("Max Int64", func(t *testing.T) {
		value := core.NewValue(int64(9223372036854775807))
		assert.Equal(t, int64(9223372036854775807), value.ToInt())
	})

	t.Run("Min Int64", func(t *testing.T) {
		value := core.NewValue(int64(-9223372036854775808))
		assert.Equal(t, int64(-9223372036854775808), value.ToInt())
	})

	t.Run("Max Uint64", func(t *testing.T) {
		value := core.NewValue(uint64(18446744073709551615))
		assert.Equal(t, uint64(18446744073709551615), value.ToUint())
	})

	t.Run("Float Precision", func(t *testing.T) {
		value := core.NewValue(3.141592653589793)
		assert.Equal(t, 3.141592653589793, value.ToFloat())
	})
}

func TestCast_ArrayCastingFailures(t *testing.T) {
	t.Parallel()

	t.Run("Array with Invalid Elements", func(t *testing.T) {
		// Create an array with mixed types that can't be converted to int
		value := core.NewValue([]interface{}{"valid", "invalid-number"})
		// The cast library is lenient and converts invalid strings to 0
		result := value.ToIntArray()
		assert.Equal(t, []int64{0, 0}, result)
	})

	t.Run("Array with Nil Elements", func(t *testing.T) {
		// This should work as nil values are handled
		value := core.NewValue([]interface{}{"1", nil, "3"})
		// The cast library is lenient and converts nil to 0
		result := value.ToIntArray()
		assert.Equal(t, []int64{1, 0, 3}, result)
	})
}

func TestCast_NilValueCasting(t *testing.T) {
	t.Parallel()

	var value *core.Value

	t.Run("Nil to Bool", func(t *testing.T) {
		assert.Equal(t, false, value.ToBool())
	})

	t.Run("Nil to Int", func(t *testing.T) {
		assert.Equal(t, int64(0), value.ToInt())
	})

	t.Run("Nil to Uint", func(t *testing.T) {
		assert.Equal(t, uint64(0), value.ToUint())
	})

	t.Run("Nil to Float", func(t *testing.T) {
		assert.Equal(t, float64(0), value.ToFloat())
	})

	t.Run("Nil to String", func(t *testing.T) {
		assert.Equal(t, "", value.ToString())
	})

	t.Run("Nil to Bytes", func(t *testing.T) {
		assert.Nil(t, value.ToBytes())
	})

	t.Run("Nil to Time", func(t *testing.T) {
		assert.Equal(t, time.Time{}, value.ToTime())
	})

	t.Run("Nil to Arrays", func(t *testing.T) {
		assert.Nil(t, value.ToBoolArray())
		assert.Nil(t, value.ToIntArray())
		assert.Nil(t, value.ToUintArray())
		assert.Nil(t, value.ToFloatArray())
		assert.Nil(t, value.ToStringArray())
		assert.Nil(t, value.ToBytesArray())
		assert.Nil(t, value.ToTimeArray())
	})
}

func TestCast_TypeMismatchCasting(t *testing.T) {
	t.Parallel()

	t.Run("Array to Scalar", func(t *testing.T) {
		value := core.NewValue([]string{"a", "b"})
		assert.Panics(t, func() {
			value.ToInt()
		})
	})

	t.Run("Map to Scalar", func(t *testing.T) {
		value := core.NewValue(map[string]int{"a": 1})
		assert.Panics(t, func() {
			value.ToString()
		})
	})

	t.Run("Scalar to Array", func(t *testing.T) {
		value := core.NewValue("not-an-array")
		assert.Panics(t, func() {
			value.ToIntArray()
		})
	})
}

func TestCast_EdgeCaseConversions(t *testing.T) {
	t.Parallel()

	t.Run("Zero Values", func(t *testing.T) {
		t.Run("Zero Int", func(t *testing.T) {
			value := core.NewValue(0)
			assert.Equal(t, false, value.ToBool())
			assert.Equal(t, int64(0), value.ToInt())
			assert.Equal(t, uint64(0), value.ToUint())
			assert.Equal(t, float64(0), value.ToFloat())
		})

		t.Run("Zero Float", func(t *testing.T) {
			value := core.NewValue(0.0)
			assert.Equal(t, false, value.ToBool())
			assert.Equal(t, int64(0), value.ToInt())
			assert.Equal(t, uint64(0), value.ToUint())
			assert.Equal(t, float64(0), value.ToFloat())
		})

		t.Run("Empty String", func(t *testing.T) {
			value := core.NewValue("")
			assert.Equal(t, false, value.ToBool())
			assert.Equal(t, int64(0), value.ToInt())
			assert.Equal(t, uint64(0), value.ToUint())
			assert.Equal(t, float64(0), value.ToFloat())
		})
	})

	t.Run("Negative Numbers", func(t *testing.T) {
		value := core.NewValue(-123)
		assert.Equal(t, true, value.ToBool()) // Non-zero is true
		assert.Equal(t, int64(-123), value.ToInt())
		assert.Equal(t, uint64(18446744073709551493), value.ToUint()) // Wraps around
		assert.Equal(t, float64(-123), value.ToFloat())
	})
}

func TestCast_StringConversionEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Special Float Values", func(t *testing.T) {
		value := core.NewValue(math.Inf(1))
		assert.Equal(t, "+Inf", value.ToString())
	})

	t.Run("NaN Values", func(t *testing.T) {
		value := core.NewValue(math.NaN())
		assert.Equal(t, "NaN", value.ToString())
	})
}
