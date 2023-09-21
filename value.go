package xdb

import (
	"encoding"
	"encoding/json"
	"time"

	"github.com/spf13/cast"
)

// Value is a wrapper that can hold any type of value.
type Value struct {
	any
	typ Type
}

// NewValue creates a new value.
func NewValue(typ Type, v any) *Value {
	return &Value{any: v, typ: typ}
}

// Type returns the type of the value.
func (v Value) Type() Type {
	return v.typ
}

// String returns the value as a string.
func (v Value) String() string {
	return cast.ToString(v.any)
}

// Int returns the value as an int64.
func (v Value) Int() int64 {
	return cast.ToInt64(v.any)
}

// Uint returns the value as a uint64.
func (v Value) Uint() uint64 {
	return cast.ToUint64(v.any)
}

// Float returns the value as a float64.
func (v Value) Float() float64 {
	return cast.ToFloat64(v.any)
}

// Bool returns the value as a bool.
func (v Value) Bool() bool {
	return cast.ToBool(v.any)
}

// Time returns the value as a time.Time.
// The time is decoded as milliseconds since epoch.
// Always returns UTC time.
func (v Value) Time() time.Time {
	t := time.Unix(0, v.Int()*int64(time.Millisecond))

	return t.UTC()
}

// Bytes returns the value as a []byte.
func (v Value) Bytes() []byte {
	switch b := v.any.(type) {
	case []byte:
		return b
	case string:
		return []byte(b)
	default:
		return []byte(cast.ToString(b))
	}
}

// Binary decodes the value into a binary value.
func (v Value) Binary(b encoding.BinaryUnmarshaler) error {
	return b.UnmarshalBinary(v.Bytes())
}

// JSON decodes the value into a JSON value.
func (v Value) JSON(j json.Unmarshaler) error {
	return j.UnmarshalJSON(v.Bytes())
}

// Empty returns true if the value is empty.
func (v Value) Empty() bool {
	return v.typ == TypeEmpty || v.any == nil
}

// Key returns the value as a key.
func (v Value) Key() *Key {
	if v.typ != TypeKey {
		return nil
	}

	return v.any.(*Key)
}

// Empty creates an empty value.
func Empty() *Value {
	return &Value{typ: TypeEmpty}
}

// String encodes string.
func String(s string) *Value {
	return NewValue(TypeString, []byte(s))
}

// Time encodes time.Time.
// The time is encoded as milliseconds since epoch.
func Time(t time.Time) *Value {
	return NewValue(
		TypeTime,
		int64(t.UTC().UnixNano()/int64(time.Millisecond)),
	)
}

// Int encodes int64, int32, int16, int8, int.
func Int[T int64 | int32 | int16 | int8 | int](i T) *Value {
	return NewValue(TypeInt, i)
}

// Uint creates a new value from uint64, uint32, uint16, uint8, uint.
func Uint[T uint64 | uint32 | uint16 | uint8 | uint](i T) *Value {
	return NewValue(TypeUint, i)
}

// Float creates a new value from float64, float32.
func Float[T float64 | float32](f T) *Value {
	return NewValue(TypeFloat, f)
}

// Bool creates a new value from bool.
func Bool(b bool) *Value {
	return NewValue(TypeBool, b)
}

// Bytes creates a new value from []byte.
func Bytes(b []byte) *Value {
	return NewValue(TypeBytes, b)
}

// Binary creates a new value from any type
// that implements encoding.BinaryMarshaler.
func Binary(b encoding.BinaryMarshaler) *Value {
	return NewValue(TypeBinary, b)
}

// JSON creates a new value from any type
// that implements json.Marshaler.
func JSON(j json.Marshaler) *Value {
	return NewValue(TypeJSON, j)
}

// List creates a new []*Value from any list.
func List[T any](list []T) *Value {
	vs := make([]*Value, len(list))

	for i, elem := range list {
		vs[i] = newValue(elem)
	}

	return NewValue(TypeList, vs)
}

// key creates a new value from a key.
func key(k *Key) *Value {
	return NewValue(TypeKey, k)
}

func newValue(v any) *Value {
	switch v := v.(type) {
	case string:
		return String(v)
	case int64:
		return Int(v)
	case int32:
		return Int(v)
	case int16:
		return Int(v)
	case int8:
		return Int(v)
	case int:
		return Int(v)
	case uint64:
		return Uint(v)
	case uint32:
		return Uint(v)
	case uint16:
		return Uint(v)
	case uint8:
		return Uint(v)
	case uint:
		return Uint(v)
	case float64:
		return Float(v)
	case float32:
		return Float(v)
	case bool:
		return Bool(v)
	case []byte:
		return Bytes(v)
	case time.Time:
		return Time(v)
	case encoding.BinaryMarshaler:
		return Binary(v)
	case json.Marshaler:
		return JSON(v)
	default:
		return NewValue(TypeUnknown, v)
	}
}
