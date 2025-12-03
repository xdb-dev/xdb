package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestNewValue_Primitives(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected core.TID
	}{
		{
			name:     "string",
			value:    "hello",
			expected: core.TIDString,
		},
		{
			name:     "int",
			value:    1,
			expected: core.TIDInteger,
		},
		{
			name:     "float",
			value:    1.0,
			expected: core.TIDFloat,
		},
		{
			name:     "bool",
			value:    true,
			expected: core.TIDBoolean,
		},
		{
			name:     "bytes",
			value:    []byte("hello"),
			expected: core.TIDBytes,
		},
		{
			name:     "time",
			value:    time.Now(),
			expected: core.TIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := core.NewValue(tc.value)

			got := value.Type().ID()
			assert.Equal(t, tc.expected, got)
			assert.EqualValues(t, tc.value, value.Unwrap())
		})
	}
}

func TestNewValue_Arrays(t *testing.T) {
	testcases := []struct {
		name     string
		value    any
		expected core.TID
	}{
		{
			name:     "string",
			value:    []string{"hello", "world"},
			expected: core.TIDString,
		},
		{
			name:     "int",
			value:    []int64{1, 2, 3},
			expected: core.TIDInteger,
		},
		{
			name:     "float",
			value:    []float64{1.0, 2.0, 3.0},
			expected: core.TIDFloat,
		},
		{
			name:     "bool",
			value:    []bool{true, false, true},
			expected: core.TIDBoolean,
		},
		{
			name:     "bytes",
			value:    [][]byte{[]byte("hello"), []byte("world")},
			expected: core.TIDBytes,
		},
		{
			name:     "time",
			value:    []time.Time{time.Now(), time.Now().Add(time.Hour)},
			expected: core.TIDTime,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			value := core.NewValue(tc.value)

			at := value.Type()
			assert.Equal(t, core.TIDArray, at.ID())
			assert.Equal(t, tc.expected, at.ValueTypeID())
		})
	}
}

func TestValue_NilValues(t *testing.T) {
	t.Parallel()

	t.Run("Direct Nil", func(t *testing.T) {
		value, err := core.NewSafeValue(nil)
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Nil Pointer", func(t *testing.T) {
		var ptr *string
		value, err := core.NewSafeValue(ptr)
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Nil Interface", func(t *testing.T) {
		var iface interface{}
		value, err := core.NewSafeValue(iface)
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestValue_EmptyArrays(t *testing.T) {
	t.Parallel()

	t.Run("Empty String Array", func(t *testing.T) {
		value, err := core.NewSafeValue([]string{})
		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("Empty Int Array", func(t *testing.T) {
		value, err := core.NewSafeValue([]int{})
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestValue_EmptyMaps(t *testing.T) {
	t.Parallel()

	t.Run("Empty String Map", func(t *testing.T) {
		value, err := core.NewSafeValue(map[string]string{})
		assert.NoError(t, err)
		assert.Nil(t, value)
	})
}

func TestValue_UnsupportedTypes(t *testing.T) {
	t.Parallel()

	t.Run("Struct Type", func(t *testing.T) {
		type unsupported struct {
			Field string
		}
		_, err := core.NewSafeValue(unsupported{Field: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported value")
	})

	t.Run("Channel Type", func(t *testing.T) {
		ch := make(chan int)
		_, err := core.NewSafeValue(ch)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported value")
	})

	t.Run("Function Type", func(t *testing.T) {
		fn := func() {}
		_, err := core.NewSafeValue(fn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported value")
	})
}

func TestValue_PanicUnsupportedTypes(t *testing.T) {
	t.Parallel()

	t.Run("Struct Type Panic", func(t *testing.T) {
		type unsupported struct {
			Field string
		}
		assert.Panics(t, func() {
			core.NewValue(unsupported{Field: "test"})
		})
	})
}

func TestValue_MethodsOnNil(t *testing.T) {
	t.Parallel()

	value := core.NewValue(nil)
	assert.Nil(t, value)
}

func TestValue_MixedTypes(t *testing.T) {
	t.Parallel()

	path := "com.example/test/test-id"

	t.Run("Array with Mixed Types", func(t *testing.T) {
		// This should work as each element is converted individually
		value := []any{"string", 123, true, 45.67}
		tuple := core.NewTuple(path, "attr", value)

		assert.NotNil(t, tuple)
		assert.Equal(t, core.TIDArray, tuple.Value().Type().ID())
	})

	t.Run("Map with Mixed Types", func(t *testing.T) {
		// This should work as keys and values are converted individually
		value := map[string]any{
			"string": "value",
			"number": 123,
			"bool":   true,
		}
		tuple := core.NewTuple(path, "attr", value)

		assert.NotNil(t, tuple)
		assert.Equal(t, core.TIDMap, tuple.Value().Type().ID())
	})
}

func TestValue_PointerDereferencing(t *testing.T) {
	t.Parallel()

	str := "hello"
	ptr := &str
	ptrPtr := &ptr

	value, err := core.NewSafeValue(ptrPtr)
	assert.NoError(t, err)
	assert.Equal(t, "hello", value.ToString())
}

func TestValue_AlreadyValueType(t *testing.T) {
	t.Parallel()

	original := core.NewValue("test")
	value, err := core.NewSafeValue(original)
	assert.NoError(t, err)
	assert.Equal(t, original, value)
}

func TestValue_TypeInformation(t *testing.T) {
	t.Parallel()

	t.Run("Boolean Type", func(t *testing.T) {
		value := core.NewValue(true)
		assert.Equal(t, core.TIDBoolean, value.Type().ID())
		assert.Equal(t, "BOOLEAN", value.Type().String())
	})

	t.Run("Array Type", func(t *testing.T) {
		value := core.NewValue([]string{"a", "b"})
		assert.Equal(t, core.TIDArray, value.Type().ID())
		assert.Equal(t, core.TIDString, value.Type().ValueTypeID())
	})

	t.Run("Map Type", func(t *testing.T) {
		value := core.NewValue(map[string]int{"a": 1})
		assert.Equal(t, core.TIDMap, value.Type().ID())
		assert.Equal(t, core.TIDString, value.Type().KeyTypeID())
		assert.Equal(t, core.TIDInteger, value.Type().ValueTypeID())
	})
}
