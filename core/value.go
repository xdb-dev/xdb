package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrUnsupportedValue is returned when a value is not supported.
	ErrUnsupportedValue = errors.New("xdb/types: unsupported value")
	// ErrTypeMismatch is returned when a value is not of the expected type.
	ErrTypeMismatch = errors.New("xdb/types: type mismatch")
)

// Value represents an attribute value using a tagged union.
// A zero Value is considered a nil value.
type Value struct {
	typ  Type
	data any
}

// NewValue creates a new value.
// Panics if the value is not supported.
func NewValue(input any) *Value {
	v, err := NewSafeValue(input)
	if err != nil {
		panic(err)
	}

	return v
}

// NewSafeValue creates a new value.
// Returns an error if the value is not supported.
func NewSafeValue(input any) (*Value, error) {
	if input == nil {
		return nil, nil // A zero Value represents nil
	}

	// If it's already a Value, just return it.
	if v, ok := input.(*Value); ok {
		return v, nil
	}

	iv := reflect.ValueOf(input)

	for iv.Kind() == reflect.Ptr {
		if iv.IsNil() {
			return nil, nil
		}
		iv = iv.Elem()
	}

	return newValue(iv)
}

func newValue(iv reflect.Value) (*Value, error) {
	switch iv.Kind() {
	case reflect.Bool:
		return &Value{typ: booleanType, data: iv.Bool()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Value{typ: integerType, data: iv.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Value{typ: unsignedType, data: iv.Uint()}, nil
	case reflect.Float32, reflect.Float64:
		return &Value{typ: floatType, data: iv.Float()}, nil
	case reflect.String:
		return &Value{typ: stringType, data: iv.String()}, nil
	case reflect.Struct:
		// Well-known types
		if iv.Type() == reflect.TypeOf(time.Time{}) {
			return &Value{typ: timeType, data: iv.Interface().(time.Time)}, nil
		}

		return nil, errors.Wrap(ErrUnsupportedValue, "type", iv.Type().String())
	case reflect.Slice, reflect.Array:
		// Special case for []byte
		if iv.Type().Elem().Kind() == reflect.Uint8 {
			return &Value{typ: bytesType, data: iv.Bytes()}, nil
		}

		if iv.Len() == 0 {
			return nil, nil
		}

		arr := make([]*Value, iv.Len())
		for i := 0; i < iv.Len(); i++ {
			v, err := NewSafeValue(iv.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			arr[i] = v
		}

		arrayType := NewArrayType(arr[0].Type().ID())

		return &Value{typ: arrayType, data: arr}, nil
	case reflect.Map:
		if iv.Len() == 0 {
			return nil, nil
		}

		mp := make(map[*Value]*Value)
		keys := iv.MapKeys()

		var firstKey, firstValue *Value

		for i, key := range keys {
			k, err := NewSafeValue(key.Interface())
			if err != nil {
				return nil, err
			}

			v, err := NewSafeValue(iv.MapIndex(key).Interface())
			if err != nil {
				return nil, err
			}

			if i == 0 {
				firstKey = k
				firstValue = v
			}

			mp[k] = v
		}

		mapType := NewMapType(firstKey.Type().ID(), firstValue.Type().ID())
		return &Value{typ: mapType, data: mp}, nil
	default:
		return nil, errors.Wrap(ErrUnsupportedValue, "type", iv.Type().String())
	}
}

// Type returns the type of the value.
func (v *Value) Type() Type {
	return v.typ
}

// Unwrap returns the raw data of the value.
func (v *Value) Unwrap() any {
	return v.data
}

// IsNil returns true if the value is nil (a zero Value).
func (v *Value) IsNil() bool {
	return v.data == nil
}

// String returns a string representation of the value.
func (v *Value) String() string {
	if v.IsNil() {
		return "nil"
	}

	switch v.typ.ID() {
	case TypeIDBoolean:
		return strconv.FormatBool(v.data.(bool))
	case TypeIDInteger:
		return strconv.FormatInt(v.data.(int64), 10)
	case TypeIDUnsigned:
		return strconv.FormatUint(v.data.(uint64), 10)
	case TypeIDFloat:
		return strconv.FormatFloat(v.data.(float64), 'f', -1, 64)
	case TypeIDString:
		return v.data.(string)
	case TypeIDBytes:
		return string(v.data.([]byte))
	case TypeIDTime:
		return v.data.(time.Time).Format(time.RFC3339)
	case TypeIDArray:
		values := v.data.([]Value)
		s := make([]string, len(values))
		for i, val := range values {
			s[i] = val.String()
		}
		return fmt.Sprintf("[%s]", strings.Join(s, ", "))
	case TypeIDMap:
		values := v.data.(map[Value]Value)
		s := make([]string, 0, len(values))
		for k, val := range values {
			s = append(s, fmt.Sprintf("%s: %s", k.String(), val.String()))
		}
		return fmt.Sprintf("{%s}", strings.Join(s, ", "))
	default:
		return ""
	}
}

// GoString returns Go syntax of the value.
func (v *Value) GoString() string {
	return fmt.Sprintf("Value(%s, %s)", v.typ.Name(), v.String())
}
