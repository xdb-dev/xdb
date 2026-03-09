package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Typed constructors + utility methods ---

func TestBoolValConstructor(t *testing.T) {
	v := BoolVal(true)
	require.NotNil(t, v)
	assert.Equal(t, TIDBoolean, v.Type().ID())
	assert.Equal(t, true, v.Unwrap())
}

func TestIntValConstructor(t *testing.T) {
	v := IntVal(42)
	require.NotNil(t, v)
	assert.Equal(t, TIDInteger, v.Type().ID())
	assert.Equal(t, int64(42), v.Unwrap())
}

func TestUintValConstructor(t *testing.T) {
	v := UintVal(42)
	require.NotNil(t, v)
	assert.Equal(t, TIDUnsigned, v.Type().ID())
	assert.Equal(t, uint64(42), v.Unwrap())
}

func TestFloatValConstructor(t *testing.T) {
	v := FloatVal(3.14)
	require.NotNil(t, v)
	assert.Equal(t, TIDFloat, v.Type().ID())
	assert.Equal(t, 3.14, v.Unwrap())
}

func TestStringValConstructor(t *testing.T) {
	v := StringVal("hello")
	require.NotNil(t, v)
	assert.Equal(t, TIDString, v.Type().ID())
	assert.Equal(t, "hello", v.Unwrap())
}

func TestBytesValConstructor(t *testing.T) {
	v := BytesVal([]byte("hello"))
	require.NotNil(t, v)
	assert.Equal(t, TIDBytes, v.Type().ID())
	assert.Equal(t, []byte("hello"), v.Unwrap())
}

func TestTimeValConstructor(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	v := TimeVal(now)
	require.NotNil(t, v)
	assert.Equal(t, TIDTime, v.Type().ID())
	assert.Equal(t, now, v.Unwrap())
}

func TestJSONValConstructor(t *testing.T) {
	raw := json.RawMessage(`{"key":"value"}`)
	v := JSONVal(raw)
	require.NotNil(t, v)
	assert.Equal(t, TIDJSON, v.Type().ID())
	assert.Equal(t, raw, v.Unwrap())
}

func TestArrayValConstructor(t *testing.T) {
	v := ArrayVal(
		TIDString,
		StringVal("a"),
		StringVal("b"),
	)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDString, v.Type().ElemTypeID())

	elems := v.Unwrap().([]*Value)
	require.Len(t, elems, 2)
	assert.Equal(t, "a", elems[0].Unwrap())
	assert.Equal(t, "b", elems[1].Unwrap())
}

func TestArrayValConstructorEmpty(t *testing.T) {
	v := ArrayVal(TIDInteger)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())

	elems := v.Unwrap().([]*Value)
	assert.Empty(t, elems)
}

func TestValueIsNil(t *testing.T) {
	tests := []struct {
		name string
		v    *Value
		want bool
	}{
		{"nil pointer", nil, true},
		{"zero value", &Value{}, true},
		{"non-nil", BoolVal(true), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.v.IsNil())
		})
	}
}

func TestValueString(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")

	tests := []struct {
		name string
		v    *Value
		want string
	}{
		{"nil", nil, "nil"},
		{"bool true", BoolVal(true), "true"},
		{"bool false", BoolVal(false), "false"},
		{"int", IntVal(42), "42"},
		{"int negative", IntVal(-7), "-7"},
		{"uint", UintVal(100), "100"},
		{"float", FloatVal(3.14), "3.14"},
		{"string", StringVal("hello"), "hello"},
		{"bytes", BytesVal([]byte("abc")), "abc"},
		{"time", TimeVal(now), "2024-01-15T10:30:00Z"},
		{"json", JSONVal(json.RawMessage(`{"k":"v"}`)), `{"k":"v"}`},
		{
			"array",
			ArrayVal(TIDString, StringVal("a"), StringVal("b")),
			"[a, b]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.v.String())
		})
	}
}

func TestValueGoString(t *testing.T) {
	v := BoolVal(true)
	assert.Equal(t, "Value(BOOLEAN, true)", v.GoString())
}

// --- Safe As-prefixed extractors ---

func TestAsBool(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := BoolVal(true).AsBool()
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := IntVal(42).AsBool()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})

	t.Run("nil value", func(t *testing.T) {
		got, err := (*Value)(nil).AsBool()
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestAsInt(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := IntVal(42).AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(42), got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := BoolVal(true).AsInt()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})

	t.Run("nil value", func(t *testing.T) {
		got, err := (*Value)(nil).AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(0), got)
	})
}

func TestAsUint(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := UintVal(42).AsUint()
		require.NoError(t, err)
		assert.Equal(t, uint64(42), got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := IntVal(42).AsUint()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsFloat(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := FloatVal(3.14).AsFloat()
		require.NoError(t, err)
		assert.InDelta(t, 3.14, got, 0.001)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := IntVal(42).AsFloat()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsStr(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := StringVal("hello").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := IntVal(42).AsStr()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsBytes(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		got, err := BytesVal([]byte("hello")).AsBytes()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := StringVal("hello").AsBytes()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)

	t.Run("correct type", func(t *testing.T) {
		got, err := TimeVal(now).AsTime()
		require.NoError(t, err)
		assert.Equal(t, now, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := IntVal(42).AsTime()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsJSON(t *testing.T) {
	raw := json.RawMessage(`{"k":"v"}`)

	t.Run("correct type", func(t *testing.T) {
		got, err := JSONVal(raw).AsJSON()
		require.NoError(t, err)
		assert.Equal(t, raw, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := StringVal("hello").AsJSON()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

func TestAsArray(t *testing.T) {
	t.Run("correct type", func(t *testing.T) {
		v := ArrayVal(TIDString, StringVal("a"))
		got, err := v.AsArray()
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "a", got[0].Unwrap())
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := StringVal("hello").AsArray()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

// --- Must extractors ---

func TestMustBool(t *testing.T) {
	assert.True(t, BoolVal(true).MustBool())
	assert.Panics(t, func() { IntVal(42).MustBool() })
}

func TestMustInt(t *testing.T) {
	assert.Equal(t, int64(42), IntVal(42).MustInt())
	assert.Panics(t, func() { BoolVal(true).MustInt() })
}

func TestMustUint(t *testing.T) {
	assert.Equal(t, uint64(42), UintVal(42).MustUint())
	assert.Panics(t, func() { BoolVal(true).MustUint() })
}

func TestMustFloat(t *testing.T) {
	assert.InDelta(t, 3.14, FloatVal(3.14).MustFloat(), 0.001)
	assert.Panics(t, func() { BoolVal(true).MustFloat() })
}

func TestMustStr(t *testing.T) {
	assert.Equal(t, "hello", StringVal("hello").MustStr())
	assert.Panics(t, func() { IntVal(42).MustStr() })
}

func TestMustBytes(t *testing.T) {
	assert.Equal(t, []byte("hello"), BytesVal([]byte("hello")).MustBytes())
	assert.Panics(t, func() { IntVal(42).MustBytes() })
}

func TestMustTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	assert.Equal(t, now, TimeVal(now).MustTime())
	assert.Panics(t, func() { IntVal(42).MustTime() })
}

func TestMustJSON(t *testing.T) {
	raw := json.RawMessage(`{"k":"v"}`)
	assert.Equal(t, raw, JSONVal(raw).MustJSON())
	assert.Panics(t, func() { IntVal(42).MustJSON() })
}

func TestMustArray(t *testing.T) {
	v := ArrayVal(TIDString, StringVal("a"))
	got := v.MustArray()
	require.Len(t, got, 1)
	assert.Panics(t, func() { IntVal(42).MustArray() })
}

// --- NewValue / NewSafeValue ---

func TestNewSafeValueNil(t *testing.T) {
	v, err := NewSafeValue(nil)
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNewSafeValuePassThrough(t *testing.T) {
	orig := BoolVal(true)
	v, err := NewSafeValue(orig)
	require.NoError(t, err)
	assert.Same(t, orig, v)
}

func TestNewSafeValueBool(t *testing.T) {
	v, err := NewSafeValue(true)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDBoolean, v.Type().ID())
	assert.Equal(t, true, v.Unwrap())
}

func TestNewSafeValueInts(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int64
	}{
		{"int", int(7), 7},
		{"int8", int8(7), 7},
		{"int16", int16(7), 7},
		{"int32", int32(7), 7},
		{"int64", int64(7), 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewSafeValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDInteger, v.Type().ID())
			assert.Equal(t, tt.want, v.Unwrap())
		})
	}
}

func TestNewSafeValueUints(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  uint64
	}{
		{"uint", uint(7), 7},
		{"uint16", uint16(7), 7},
		{"uint32", uint32(7), 7},
		{"uint64", uint64(7), 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewSafeValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDUnsigned, v.Type().ID())
			assert.Equal(t, tt.want, v.Unwrap())
		})
	}
}

func TestNewSafeValueUint8IsByte(t *testing.T) {
	v, err := NewSafeValue(uint8(7))
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDUnsigned, v.Type().ID())
}

func TestNewSafeValueFloats(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float64
	}{
		{"float32", float32(3.14), float64(float32(3.14))},
		{"float64", float64(3.14), 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewSafeValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDFloat, v.Type().ID())
			assert.InDelta(t, tt.want, v.Unwrap(), 0.001)
		})
	}
}

func TestNewSafeValueString(t *testing.T) {
	v, err := NewSafeValue("hello")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDString, v.Type().ID())
	assert.Equal(t, "hello", v.Unwrap())
}

func TestNewSafeValueBytes(t *testing.T) {
	v, err := NewSafeValue([]byte("hello"))
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDBytes, v.Type().ID())
	assert.Equal(t, []byte("hello"), v.Unwrap())
}

func TestNewSafeValueTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	v, err := NewSafeValue(now)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDTime, v.Type().ID())
	assert.Equal(t, now, v.Unwrap())
}

func TestNewSafeValueJSONRawMessage(t *testing.T) {
	raw := json.RawMessage(`{"key":"value"}`)
	v, err := NewSafeValue(raw)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDJSON, v.Type().ID())
	assert.Equal(t, raw, v.Unwrap())
}

func TestNewSafeValuePointer(t *testing.T) {
	s := "hello"
	v, err := NewSafeValue(&s)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDString, v.Type().ID())
	assert.Equal(t, "hello", v.Unwrap())
}

func TestNewSafeValueNilPointer(t *testing.T) {
	var s *string
	v, err := NewSafeValue(s)
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNewSafeValueSlice(t *testing.T) {
	v, err := NewSafeValue([]string{"a", "b", "c"})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDString, v.Type().ElemTypeID())

	elems := v.Unwrap().([]*Value)
	require.Len(t, elems, 3)
	assert.Equal(t, "a", elems[0].Unwrap())
}

func TestNewSafeValueSliceInt(t *testing.T) {
	v, err := NewSafeValue([]int64{1, 2, 3})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())
}

func TestNewSafeValueEmptySlice(t *testing.T) {
	v, err := NewSafeValue([]string{})
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNewSafeValueUnsupported(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"struct", struct{}{}},
		{"map", map[string]string{"a": "b"}},
		{"channel", make(chan int)},
		{"func", func() {}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSafeValue(tt.input)
			assert.ErrorIs(t, err, ErrUnsupportedValue)
		})
	}
}

func TestNewValuePanicsOnUnsupported(t *testing.T) {
	assert.Panics(t, func() { NewValue(struct{}{}) })
}

func TestNewValuePrimitives(t *testing.T) {
	assert.Equal(t, TIDBoolean, NewValue(true).Type().ID())
	assert.Equal(t, TIDInteger, NewValue(42).Type().ID())
	assert.Equal(t, TIDString, NewValue("hello").Type().ID())
}

func TestNewValueNilReturnsNil(t *testing.T) {
	assert.Nil(t, NewValue(nil))
}
