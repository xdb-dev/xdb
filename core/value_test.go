package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Typed constructors + utility methods ---

// readValue reads v through its typed As* accessor. Tests assert on the
// accessors because those are the contract callers use. Value.Unwrap
// returns the raw tagged-union payload, which carries no such guarantee.
func readValue(t *testing.T, v *Value) any {
	t.Helper()

	var (
		got any
		err error
	)

	switch v.Type().ID() {
	case TIDBoolean:
		got, err = v.AsBool()
	case TIDInteger:
		got, err = v.AsInt()
	case TIDUnsigned:
		got, err = v.AsUint()
	case TIDFloat:
		got, err = v.AsFloat()
	case TIDString:
		got, err = v.AsStr()
	case TIDBytes:
		got, err = v.AsBytes()
	case TIDTime:
		got, err = v.AsTime()
	case TIDJSON:
		got, err = v.AsJSON()
	case TIDArray:
		got, err = v.AsArray()
	default:
		t.Fatalf("readValue: no accessor for %s", v.Type().ID())
	}

	require.NoError(t, err)

	return got
}

// readArray reads v as an array of values.
func readArray(t *testing.T, v *Value) []*Value {
	t.Helper()

	elems, err := v.AsArray()
	require.NoError(t, err)

	return elems
}

// TestTypedConstructors is the table-driven reference for this package.
// Each case reads through the typed As* accessor that CLAUDE.md commits
// to, rather than through Unwrap.
func TestTypedConstructors(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	raw := json.RawMessage(`{"key":"value"}`)

	tests := []struct {
		value  *Value
		read   func(*Value) (any, error)
		name   string
		wantID TID
		want   any
	}{
		{
			name:   "bool",
			value:  BoolVal(true),
			wantID: TIDBoolean,
			want:   true,
			read:   func(v *Value) (any, error) { return v.AsBool() },
		},
		{
			name:   "int",
			value:  IntVal(42),
			wantID: TIDInteger,
			want:   int64(42),
			read:   func(v *Value) (any, error) { return v.AsInt() },
		},
		{
			name:   "uint",
			value:  UintVal(42),
			wantID: TIDUnsigned,
			want:   uint64(42),
			read:   func(v *Value) (any, error) { return v.AsUint() },
		},
		{
			name:   "float",
			value:  FloatVal(3.14),
			wantID: TIDFloat,
			want:   3.14,
			read:   func(v *Value) (any, error) { return v.AsFloat() },
		},
		{
			name:   "string",
			value:  StringVal("hello"),
			wantID: TIDString,
			want:   "hello",
			read:   func(v *Value) (any, error) { return v.AsStr() },
		},
		{
			name:   "bytes",
			value:  BytesVal([]byte("hello")),
			wantID: TIDBytes,
			want:   []byte("hello"),
			read:   func(v *Value) (any, error) { return v.AsBytes() },
		},
		{
			name:   "time",
			value:  TimeVal(now),
			wantID: TIDTime,
			want:   now,
			read:   func(v *Value) (any, error) { return v.AsTime() },
		},
		{
			name:   "json",
			value:  JSONVal(raw),
			wantID: TIDJSON,
			want:   raw,
			read:   func(v *Value) (any, error) { return v.AsJSON() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.value)
			assert.Equal(t, tt.wantID, tt.value.Type().ID())

			got, err := tt.read(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
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

	elems := readArray(t, v)
	require.Len(t, elems, 2)
	assert.Equal(t, "a", readValue(t, elems[0]))
	assert.Equal(t, "b", readValue(t, elems[1]))
}

func TestArrayValConstructorEmpty(t *testing.T) {
	v := ArrayVal(TIDInteger)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())

	elems := readArray(t, v)
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
		assert.Equal(t, "a", readValue(t, got[0]))
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := StringVal("hello").AsArray()
		assert.ErrorIs(t, err, ErrTypeMismatch)
	})
}

// --- MustNewValue / NewValue ---

func TestNewSafeValueNil(t *testing.T) {
	v, err := NewValue(nil)
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNewSafeValuePassThrough(t *testing.T) {
	orig := BoolVal(true)
	v, err := NewValue(orig)
	require.NoError(t, err)
	assert.Same(t, orig, v)
}

func TestNewSafeValueBool(t *testing.T) {
	v, err := NewValue(true)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDBoolean, v.Type().ID())
	assert.Equal(t, true, readValue(t, v))
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
			v, err := NewValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDInteger, v.Type().ID())
			assert.Equal(t, tt.want, readValue(t, v))
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
			v, err := NewValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDUnsigned, v.Type().ID())
			assert.Equal(t, tt.want, readValue(t, v))
		})
	}
}

func TestNewSafeValueUint8IsByte(t *testing.T) {
	v, err := NewValue(uint8(7))
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
			v, err := NewValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDFloat, v.Type().ID())
			assert.InDelta(t, tt.want, readValue(t, v), 0.001)
		})
	}
}

func TestNewSafeValueString(t *testing.T) {
	v, err := NewValue("hello")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDString, v.Type().ID())
	assert.Equal(t, "hello", readValue(t, v))
}

func TestNewSafeValueBytes(t *testing.T) {
	v, err := NewValue([]byte("hello"))
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDBytes, v.Type().ID())
	assert.Equal(t, []byte("hello"), readValue(t, v))
}

func TestNewSafeValueByteArray(t *testing.T) {
	type buf []byte

	tests := []struct {
		name  string
		input any
		want  []byte
	}{
		{"array", [4]byte{1, 2, 3, 4}, []byte{1, 2, 3, 4}},
		{"empty array", [0]byte{}, []byte{}},
		{"named slice", buf("hello"), []byte("hello")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewValue(tt.input)
			require.NoError(t, err)
			require.NotNil(t, v)
			assert.Equal(t, TIDBytes, v.Type().ID())
			assert.Equal(t, tt.want, readValue(t, v))
		})
	}
}

func TestNewSafeValueTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	v, err := NewValue(now)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDTime, v.Type().ID())
	assert.Equal(t, now, readValue(t, v))
}

func TestNewSafeValueJSONRawMessage(t *testing.T) {
	raw := json.RawMessage(`{"key":"value"}`)
	v, err := NewValue(raw)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDJSON, v.Type().ID())
	assert.Equal(t, raw, readValue(t, v))
}

func TestNewSafeValuePointer(t *testing.T) {
	s := "hello"
	v, err := NewValue(&s)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDString, v.Type().ID())
	assert.Equal(t, "hello", readValue(t, v))
}

func TestNewSafeValueNilPointer(t *testing.T) {
	var s *string
	v, err := NewValue(s)
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNewSafeValueSlice(t *testing.T) {
	v, err := NewValue([]string{"a", "b", "c"})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDString, v.Type().ElemTypeID())

	elems := readArray(t, v)
	require.Len(t, elems, 3)
	assert.Equal(t, "a", readValue(t, elems[0]))
}

func TestNewSafeValueSliceInt(t *testing.T) {
	v, err := NewValue([]int64{1, 2, 3})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())
}

func TestNewSafeValueEmptySlice(t *testing.T) {
	// An empty typed slice is an empty array, not nil — the element type is
	// derived from the slice's static element type.
	v, err := NewValue([]string{})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDString, v.Type().ElemTypeID())

	elems, err := v.AsArray()
	require.NoError(t, err)
	assert.Empty(t, elems)
}

func TestNewSafeValueEmptySliceInt(t *testing.T) {
	v, err := NewValue([]int64{})
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())
}

func TestNewSafeValueEmptyAnySlice(t *testing.T) {
	// An empty []any has no derivable element type.
	_, err := NewValue([]any{})
	assert.ErrorIs(t, err, ErrUnsupportedValue)
}

func TestNewSafeValueHeterogeneousSlice(t *testing.T) {
	// Mixed element types are rejected, wrapped with the offending index.
	_, err := NewValue([]any{1, "a"})
	assert.ErrorIs(t, err, ErrUnsupportedValue)
}

func TestNewSafeValueHomogeneousAnySlice(t *testing.T) {
	v, err := NewValue([]any{int64(1), int64(2)})
	require.NoError(t, err)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDInteger, v.Type().ElemTypeID())
}

func TestNewSafeValueNestedSlice(t *testing.T) {
	v, err := NewValue([][]string{{"a"}, {"b"}})
	require.NoError(t, err)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDArray, v.Type().ElemTypeID())
}

func TestNewSafeValueSliceOfBytes(t *testing.T) {
	// []byte is bytes; [][]byte is an array of bytes.
	v, err := NewValue([][]byte{[]byte("a"), []byte("b")})
	require.NoError(t, err)
	assert.Equal(t, TIDArray, v.Type().ID())
	assert.Equal(t, TIDBytes, v.Type().ElemTypeID())
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
			_, err := NewValue(tt.input)
			assert.ErrorIs(t, err, ErrUnsupportedValue)
		})
	}
}

func TestNewValuePanicsOnUnsupported(t *testing.T) {
	assert.Panics(t, func() { MustNewValue(struct{}{}) })
}

func TestNewValuePrimitives(t *testing.T) {
	assert.Equal(t, TIDBoolean, MustNewValue(true).Type().ID())
	assert.Equal(t, TIDInteger, MustNewValue(42).Type().ID())
	assert.Equal(t, TIDString, MustNewValue("hello").Type().ID())
}

func TestNewValueNilReturnsNil(t *testing.T) {
	assert.Nil(t, MustNewValue(nil))
}

// --- NewTypedValue ---

func TestNewTypedValue(t *testing.T) {
	t.Run("built-in type", func(t *testing.T) {
		v := NewTypedValue(TypeBool, true)
		require.NotNil(t, v)
		assert.Equal(t, TIDBoolean, v.Type().ID())
		assert.Equal(t, true, readValue(t, v))
	})

	// A custom TID has no typed accessor, so Unwrap is the only way to read
	// it. This is the case that keeps Value.Unwrap exported.
	t.Run("custom type", func(t *testing.T) {
		customTID := TID("MONEY")
		customType := NewType(customTID)
		v := NewTypedValue(customType, int64(1999))
		require.NotNil(t, v)
		assert.Equal(t, customTID, v.Type().ID())
		assert.Equal(t, int64(1999), v.Unwrap())
	})
}
