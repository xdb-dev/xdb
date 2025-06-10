package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToBool())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToInt())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToUint())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToFloat())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToString())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected, tuple.ToBytes())
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
			tuple := types.NewTuple("Test", "1", "attr", tc.value)
			assert.Equal(t, tc.expected.Compare(tuple.ToTime()), 0)
		})
	}
}

// func TestValue_Slices(t *testing.T) {
// 	t.Parallel()

// 	t.Run("IntSlice", func(t *testing.T) {
// 		value := []int64{1, 2, 3}
// 		tuple := types.NewTuple("Test", "1", "attr", value)
// 		assert.Equal(t, value, tuple.ToIntSlice())
// 	})

// 	t.Run("FloatSlice", func(t *testing.T) {
// 		value := []float64{1.1, 2.2, 3.3}
// 		tuple := types.NewTuple("Test", "1", "attr", value)
// 		assert.Equal(t, value, tuple.ToFloatSlice())
// 		assert.Equal(t, value, tuple.Value().ToFloatSlice())
// 	})

// 	t.Run("StringSlice", func(t *testing.T) {
// 		value := []string{"a", "b", "c"}
// 		tuple := types.NewTuple("Test", "1", "attr", value)
// 		assert.Equal(t, value, tuple.ToStringSlice())
// 		assert.Equal(t, value, tuple.Value().ToStringSlice())
// 	})

// 	t.Run("BoolSlice", func(t *testing.T) {
// 		value := []bool{true, false, true}
// 		tuple := types.NewTuple("Test", "1", "attr", value)
// 		assert.Equal(t, value, tuple.ToBoolSlice())
// 		assert.Equal(t, value, tuple.Value().ToBoolSlice())
// 	})

// 	t.Run("BytesSlice", func(t *testing.T) {
// 		value := [][]byte{[]byte("a"), []byte("b")}
// 		tuple := types.NewTuple("Test", "1", "attr", value)
// 		assert.Equal(t, value, tuple.ToBytesSlice())
// 		assert.Equal(t, value, tuple.Value().ToBytesSlice())
// 	})
// }

// func TestValue_Bytes(t *testing.T) {
// 	t.Parallel()

// 	value := []byte("hello")
// 	tuple := types.NewTuple("Test", "1", "attr", value)
// 	assert.Equal(t, value, tuple.ToBytes())
// 	assert.Equal(t, value, tuple.Value().ToBytes())
// }

// func TestValue_Time(t *testing.T) {
// 	t.Parallel()

// 	now := time.Now()
// 	tuple := types.NewTuple("Test", "1", "attr", now)
// 	assert.Equal(t, now, tuple.ToTime())
// 	assert.Equal(t, now, tuple.Value().ToTime())

// 	t.Run("TimeSlice", func(t *testing.T) {
// 		times := []time.Time{now, now.Add(time.Hour)}
// 		tuple := types.NewTuple("Test", "1", "attr", times)
// 		assert.Equal(t, times, tuple.ToTimeSlice())
// 		assert.Equal(t, times, tuple.Value().ToTimeSlice())
// 	})
// }

// func TestValue_Point(t *testing.T) {
// 	t.Parallel()

// 	point := types.Point{Lat: 1.0, Long: 2.0}
// 	tuple := types.NewTuple("Test", "1", "attr", point)
// 	assert.Equal(t, point, tuple.ToPoint())
// 	assert.Equal(t, point, tuple.Value().ToPoint())

// 	t.Run("PointSlice", func(t *testing.T) {
// 		points := []types.Point{{Lat: 1.0, Long: 2.0}, {Lat: 3.0, Long: 4.0}}
// 		tuple := types.NewTuple("Test", "1", "attr", points)
// 		assert.Equal(t, points, tuple.ToPointSlice())
// 		assert.Equal(t, points, tuple.Value().ToPointSlice())
// 	})
// }
