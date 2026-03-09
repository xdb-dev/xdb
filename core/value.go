package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrUnsupportedValue is returned when a value is not supported.
	ErrUnsupportedValue = errors.New("[xdb/core] unsupported value")
	// ErrTypeMismatch is returned when a value is not of the expected type.
	ErrTypeMismatch = errors.New("[xdb/core] type mismatch")
)

// Value represents an attribute value using a tagged union.
// A zero Value is considered a nil value.
type Value struct {
	typ  Type
	data any
}

// --- Typed constructors ---

// BoolVal creates a new boolean [Value].
func BoolVal(v bool) *Value {
	return &Value{typ: TypeBool, data: v}
}

// IntVal creates a new integer [Value] from an int64.
func IntVal(v int64) *Value {
	return &Value{typ: TypeInt, data: v}
}

// UintVal creates a new unsigned integer [Value] from a uint64.
func UintVal(v uint64) *Value {
	return &Value{typ: TypeUnsigned, data: v}
}

// FloatVal creates a new floating-point [Value] from a float64.
func FloatVal(v float64) *Value {
	return &Value{typ: TypeFloat, data: v}
}

// StringVal creates a new string [Value].
func StringVal(v string) *Value {
	return &Value{typ: TypeString, data: v}
}

// BytesVal creates a new byte slice [Value].
func BytesVal(v []byte) *Value {
	return &Value{typ: TypeBytes, data: v}
}

// TimeVal creates a new timestamp [Value] from a [time.Time].
func TimeVal(v time.Time) *Value {
	return &Value{typ: TypeTime, data: v}
}

// JSONVal creates a new JSON [Value] from a [json.RawMessage].
func JSONVal(v json.RawMessage) *Value {
	return &Value{typ: TypeJSON, data: v}
}

// ArrayVal creates a new array [Value] with the given element type and elements.
func ArrayVal(elemTypeID TID, elems ...*Value) *Value {
	return &Value{
		typ:  NewArrayType(elemTypeID),
		data: elems,
	}
}

// --- Utility methods ---

// Type returns the [Type] of the value.
func (v *Value) Type() Type {
	return v.typ
}

// Unwrap returns the raw data of the value.
func (v *Value) Unwrap() any {
	return v.data
}

// IsNil returns true if the value is nil (a nil pointer or zero Value).
func (v *Value) IsNil() bool {
	if v == nil {
		return true
	}
	return v.data == nil
}

// String returns a string representation of the value.
func (v *Value) String() string {
	if v.IsNil() {
		return "nil"
	}

	switch v.typ.ID() {
	case TIDBoolean:
		return strconv.FormatBool(v.data.(bool))
	case TIDInteger:
		return strconv.FormatInt(v.data.(int64), 10)
	case TIDUnsigned:
		return strconv.FormatUint(v.data.(uint64), 10)
	case TIDFloat:
		return strconv.FormatFloat(v.data.(float64), 'f', -1, 64)
	case TIDString:
		return v.data.(string)
	case TIDBytes:
		return string(v.data.([]byte))
	case TIDTime:
		return v.data.(time.Time).Format(time.RFC3339)
	case TIDJSON:
		return string(v.data.(json.RawMessage))
	case TIDArray:
		elems := v.data.([]*Value)
		parts := make([]string, len(elems))
		for i, elem := range elems {
			parts[i] = elem.String()
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	default:
		return ""
	}
}

// GoString returns Go syntax of the value.
func (v *Value) GoString() string {
	return fmt.Sprintf("Value(%s, %s)", v.typ.String(), v.String())
}

// --- Safe extractors (As-prefixed) ---

// AsBool returns the value as a bool.
// Returns [ErrTypeMismatch] if the value is not a boolean.
func (v *Value) AsBool() (bool, error) {
	if v == nil {
		return false, nil
	}
	if v.typ.id != TIDBoolean {
		return false, ErrTypeMismatch
	}
	return v.data.(bool), nil
}

// AsInt returns the value as an int64.
// Returns [ErrTypeMismatch] if the value is not an integer.
func (v *Value) AsInt() (int64, error) {
	if v == nil {
		return 0, nil
	}
	if v.typ.id != TIDInteger {
		return 0, ErrTypeMismatch
	}
	return v.data.(int64), nil
}

// AsUint returns the value as a uint64.
// Returns [ErrTypeMismatch] if the value is not an unsigned integer.
func (v *Value) AsUint() (uint64, error) {
	if v == nil {
		return 0, nil
	}
	if v.typ.id != TIDUnsigned {
		return 0, ErrTypeMismatch
	}
	return v.data.(uint64), nil
}

// AsFloat returns the value as a float64.
// Returns [ErrTypeMismatch] if the value is not a float.
func (v *Value) AsFloat() (float64, error) {
	if v == nil {
		return 0, nil
	}
	if v.typ.id != TIDFloat {
		return 0, ErrTypeMismatch
	}
	return v.data.(float64), nil
}

// AsStr returns the value as a string.
// Returns [ErrTypeMismatch] if the value is not a string.
func (v *Value) AsStr() (string, error) {
	if v == nil {
		return "", nil
	}
	if v.typ.id != TIDString {
		return "", ErrTypeMismatch
	}
	return v.data.(string), nil
}

// AsBytes returns the value as a []byte.
// Returns [ErrTypeMismatch] if the value is not bytes.
func (v *Value) AsBytes() ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	if v.typ.id != TIDBytes {
		return nil, ErrTypeMismatch
	}
	return v.data.([]byte), nil
}

// AsTime returns the value as a [time.Time].
// Returns [ErrTypeMismatch] if the value is not a timestamp.
func (v *Value) AsTime() (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	if v.typ.id != TIDTime {
		return time.Time{}, ErrTypeMismatch
	}
	return v.data.(time.Time), nil
}

// AsJSON returns the value as a [json.RawMessage].
// Returns [ErrTypeMismatch] if the value is not JSON.
func (v *Value) AsJSON() (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	if v.typ.id != TIDJSON {
		return nil, ErrTypeMismatch
	}
	return v.data.(json.RawMessage), nil
}

// AsArray returns the value as a slice of [*Value].
// Returns [ErrTypeMismatch] if the value is not an array.
func (v *Value) AsArray() ([]*Value, error) {
	if v == nil {
		return nil, nil
	}
	if v.typ.id != TIDArray {
		return nil, ErrTypeMismatch
	}
	return v.data.([]*Value), nil
}

// --- Must extractors ---

// MustBool returns the value as a bool or panics on type mismatch.
func (v *Value) MustBool() bool {
	b, err := v.AsBool()
	if err != nil {
		panic(err)
	}
	return b
}

// MustInt returns the value as an int64 or panics on type mismatch.
func (v *Value) MustInt() int64 {
	i, err := v.AsInt()
	if err != nil {
		panic(err)
	}
	return i
}

// MustUint returns the value as a uint64 or panics on type mismatch.
func (v *Value) MustUint() uint64 {
	u, err := v.AsUint()
	if err != nil {
		panic(err)
	}
	return u
}

// MustFloat returns the value as a float64 or panics on type mismatch.
func (v *Value) MustFloat() float64 {
	f, err := v.AsFloat()
	if err != nil {
		panic(err)
	}
	return f
}

// MustStr returns the value as a string or panics on type mismatch.
func (v *Value) MustStr() string {
	s, err := v.AsStr()
	if err != nil {
		panic(err)
	}
	return s
}

// MustBytes returns the value as a []byte or panics on type mismatch.
func (v *Value) MustBytes() []byte {
	b, err := v.AsBytes()
	if err != nil {
		panic(err)
	}
	return b
}

// MustTime returns the value as a [time.Time] or panics on type mismatch.
func (v *Value) MustTime() time.Time {
	t, err := v.AsTime()
	if err != nil {
		panic(err)
	}
	return t
}

// MustJSON returns the value as a [json.RawMessage] or panics on type mismatch.
func (v *Value) MustJSON() json.RawMessage {
	j, err := v.AsJSON()
	if err != nil {
		panic(err)
	}
	return j
}

// MustArray returns the value as a slice of [*Value] or panics on type mismatch.
func (v *Value) MustArray() []*Value {
	a, err := v.AsArray()
	if err != nil {
		panic(err)
	}
	return a
}

// --- Dynamic constructors ---

var (
	timeType   = reflect.TypeFor[time.Time]()
	rawMsgType = reflect.TypeFor[json.RawMessage]()
	byteSlice  = reflect.TypeFor[[]byte]()
)

// NewValue creates a new [Value] from the given input.
// Panics if the input type is not supported.
//
// Supported types: bool, int*, uint*, float*, string, []byte,
// [time.Time], [json.RawMessage], and slices of supported types.
func NewValue(input any) *Value {
	v, err := NewSafeValue(input)
	if err != nil {
		panic(err)
	}
	return v
}

// NewSafeValue creates a new [Value] from the given input.
// Returns [ErrUnsupportedValue] if the input type is not supported.
func NewSafeValue(input any) (*Value, error) {
	if input == nil {
		return nil, nil
	}

	if v, ok := input.(*Value); ok {
		return v, nil
	}

	iv := reflect.ValueOf(input)

	for iv.Kind() == reflect.Pointer {
		if iv.IsNil() {
			return nil, nil
		}
		iv = iv.Elem()
	}

	return newReflectValue(iv)
}

func newReflectValue(iv reflect.Value) (*Value, error) {
	// Check concrete types before kind-based dispatch.
	switch iv.Type() {
	case timeType:
		return TimeVal(iv.Interface().(time.Time)), nil
	case rawMsgType:
		return JSONVal(iv.Interface().(json.RawMessage)), nil
	}

	switch iv.Kind() {
	case reflect.Bool:
		return BoolVal(iv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return IntVal(iv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return UintVal(iv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return FloatVal(iv.Float()), nil
	case reflect.String:
		return StringVal(iv.String()), nil
	case reflect.Slice, reflect.Array:
		return newSliceValue(iv)
	default:
		return nil, errors.Wrap(ErrUnsupportedValue, "type", iv.Type().String())
	}
}

func newSliceValue(iv reflect.Value) (*Value, error) {
	// Check for []byte (but not json.RawMessage, handled above).
	if iv.Type() == byteSlice {
		return BytesVal(iv.Bytes()), nil
	}

	// Element kind is uint8 but type isn't []byte — treat as byte slice.
	if iv.Type().Elem().Kind() == reflect.Uint8 {
		return BytesVal(iv.Bytes()), nil
	}

	if iv.Len() == 0 {
		return nil, nil
	}

	elems := make([]*Value, iv.Len())
	for i := range iv.Len() {
		v, err := NewSafeValue(iv.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		elems[i] = v
	}

	elemType := elems[0].Type().ID()

	return ArrayVal(elemType, elems...), nil
}
