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
	data any
	typ  Type
}

// NewTypedValue creates a [Value] with the given [Type] and data.
// This is intended for custom type implementations where the built-in
// typed constructors are not applicable.
func NewTypedValue(typ Type, data any) *Value {
	return &Value{typ: typ, data: data}
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

// BytesVal creates a byte slice [Value] without copying v.
// Changes to v or the slice returned by [Value.AsBytes] affect the value.
func BytesVal(v []byte) *Value {
	return &Value{typ: TypeBytes, data: v}
}

// TimeVal creates a new timestamp [Value] from a [time.Time].
func TimeVal(v time.Time) *Value {
	return &Value{typ: TypeTime, data: v}
}

// JSONVal creates a JSON [Value] without copying v.
// Changes to v or the bytes returned by [Value.AsJSON] affect the value.
func JSONVal(v json.RawMessage) *Value {
	return &Value{typ: TypeJSON, data: v}
}

// ArrayVal creates a new array [Value] with the given element type and elements.
//
// ArrayVal retains elems and its values without copying them.
// It does not verify that each element's type matches elemTypeID.
// [NewValue] and [NewSafeValue] derive and validate the
// element type when building arrays from Go slices.
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

// --- Dynamic constructors ---

var (
	timeType   = reflect.TypeFor[time.Time]()
	rawMsgType = reflect.TypeFor[json.RawMessage]()
	byteSlice  = reflect.TypeFor[[]byte]()
)

// NewValue creates a new [Value] from the given input.
// Panics if the input type is not supported.
//
// Supported types: bool, int*, uint*, float*, string, []byte, byte arrays,
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
//
// A slice or an array whose element type is a byte becomes [TypeBytes];
// the bytes of an array are copied, so the value does not alias the input.
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
	// Any slice or array of bytes is bytes (json.RawMessage is handled above).
	if iv.Type().Elem().Kind() == reflect.Uint8 {
		return BytesVal(bytesFrom(iv)), nil
	}

	// Empty slice: an empty array whose element type comes from the slice's
	// static element type. An empty []any has no derivable element type.
	if iv.Len() == 0 {
		elemTID, err := elemTIDFromType(iv.Type().Elem())
		if err != nil {
			return nil, err
		}
		return ArrayVal(elemTID), nil
	}

	elems := make([]*Value, iv.Len())
	var elemType Type
	haveType := false
	for i := range iv.Len() {
		v, err := NewSafeValue(iv.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		elems[i] = v

		if v == nil {
			continue // explicit nil element carries no type
		}
		if !haveType {
			elemType = v.Type()
			haveType = true
			continue
		}
		if v.Type() != elemType {
			return nil, errors.Wrap(ErrUnsupportedValue, "index", strconv.Itoa(i))
		}
	}

	// All elements were nil: fall back to the static element type.
	if !haveType {
		elemTID, err := elemTIDFromType(iv.Type().Elem())
		if err != nil {
			return nil, err
		}
		return ArrayVal(elemTID, elems...), nil
	}

	return ArrayVal(elemType.ID(), elems...), nil
}

// bytesFrom copies the bytes out of a slice or array of uint8.
// An array is not addressable, so [reflect.Value.Bytes] cannot be used on it.
func bytesFrom(iv reflect.Value) []byte {
	if iv.Kind() == reflect.Slice {
		return iv.Bytes()
	}

	out := make([]byte, iv.Len())
	reflect.Copy(reflect.ValueOf(out), iv)

	return out
}

// elemTIDFromType maps a static Go element type to its [TID].
// Returns [ErrUnsupportedValue] for types with no XDB equivalent
// (e.g. an interface element type, as in []any).
func elemTIDFromType(t reflect.Type) (TID, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t {
	case timeType:
		return TIDTime, nil
	case rawMsgType:
		return TIDJSON, nil
	case byteSlice:
		return TIDBytes, nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return TIDBoolean, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TIDInteger, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TIDUnsigned, nil
	case reflect.Float32, reflect.Float64:
		return TIDFloat, nil
	case reflect.String:
		return TIDString, nil
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return TIDBytes, nil
		}
		return TIDArray, nil
	default:
		return TIDUnknown, errors.Wrap(ErrUnsupportedValue, "type", t.String())
	}
}
