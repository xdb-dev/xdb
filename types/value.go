package types

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
)

// Value represents an attribute value.
type Value interface {
	Type() Type
	String() string
}

// NewValue creates a new value.
// Panics if the value is not supported.
func NewValue(input any) Value {
	v, err := NewSafeValue(input)
	if err != nil {
		panic(err)
	}

	return v
}

// NewSafeValue creates a new value.
// Returns an error if the value is not supported.
func NewSafeValue(input any) (Value, error) {
	if input == nil {
		return nil, nil
	}

	if v, ok := input.(Value); ok {
		return v, nil
	}

	iv := reflect.ValueOf(input)

	for iv.Kind() == reflect.Ptr {
		iv = iv.Elem()
	}

	return newValue(iv)
}

func newValue(iv reflect.Value) (Value, error) {
	switch iv.Kind() {
	case reflect.Bool:
		return Bool(iv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Int64(iv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Uint64(iv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return Float64(iv.Float()), nil
	case reflect.String:
		return String(iv.String()), nil
	case reflect.Struct:
		// Well-known types
		if iv.Type() == reflect.TypeOf(time.Time{}) {
			return Time(iv.Interface().(time.Time)), nil
		}

		return nil, errors.Wrap(ErrUnsupportedValue, "type", iv.Type().String())
	case reflect.Slice, reflect.Array:
		if iv.Len() == 0 {
			return nil, nil
		}

		first := iv.Index(0)

		if first.Kind() == reflect.Uint8 {
			return Bytes(iv.Interface().([]byte)), nil
		}

		arr := make([]Value, iv.Len())
		for i := 0; i < iv.Len(); i++ {
			v, err := NewSafeValue(iv.Index(i).Interface())
			if err != nil {
				return nil, err
			}

			arr[i] = v
		}

		return NewArray(arr[0].Type().ID(), arr...), nil
	case reflect.Map:
		if iv.Len() == 0 {
			return nil, nil
		}

		keys := iv.MapKeys()

		key1, err := NewSafeValue(keys[0].Interface())
		if err != nil {
			return nil, err
		}

		value1, err := NewSafeValue(iv.MapIndex(keys[0]).Interface())
		if err != nil {
			return nil, err
		}

		mp := NewMap(key1.Type().ID(), value1.Type().ID())

		for _, key := range keys {
			k, err := NewSafeValue(key.Interface())
			if err != nil {
				return nil, err
			}

			v, err := NewSafeValue(iv.MapIndex(key).Interface())
			if err != nil {
				return nil, err
			}

			mp.Set(k, v)
		}

		return mp, nil
	default:
		return nil, errors.Wrap(ErrUnsupportedValue, "type", iv.Type().String())
	}
}

type Bool bool

func (b Bool) Type() Type { return BooleanType }

func (b Bool) String() string {
	return strconv.FormatBool(bool(b))
}

type Int64 int64

func (t Int64) Type() Type { return IntegerType }

func (t Int64) String() string {
	return strconv.FormatInt(int64(t), 10)
}

type Uint64 uint64

func (t Uint64) Type() Type { return UnsignedType }

func (t Uint64) String() string {
	return strconv.FormatUint(uint64(t), 10)
}

type Float64 float64

func (t Float64) Type() Type { return FloatType }

func (t Float64) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64)
}

type String string

func (t String) Type() Type { return StringType }

func (t String) String() string { return string(t) }

type Bytes []byte

func (t Bytes) Type() Type { return BytesType }

func (t Bytes) String() string { return string(t) }

type Time time.Time

func (t Time) Type() Type { return TimeType }

func (t Time) String() string { return time.Time(t).Format(time.RFC3339) }

type Array struct {
	typ    Type
	values []Value
}

func NewArray(t TypeID, values ...Value) *Array {
	return &Array{
		typ:    NewArrayType(t),
		values: values,
	}
}

func (t *Array) Type() Type { return t.typ }

func (t *Array) String() string {
	values := make([]string, len(t.values))
	for i, v := range t.values {
		values[i] = v.String()
	}

	return fmt.Sprintf("[%s]", strings.Join(values, ", "))
}

func (t *Array) ValueType() TypeID {
	return t.typ.ValueType()
}

func (t *Array) Values() []Value {
	return t.values
}

func (t *Array) Append(v ...Value) *Array {
	t.values = append(t.values, v...)
	return t
}

func (t *Array) Get(i int) Value {
	return t.values[i]
}

type Map struct {
	typ    Type
	values map[Value]Value
}

func NewMap(kt, vt TypeID) *Map {
	return &Map{
		typ:    NewMapType(kt, vt),
		values: map[Value]Value{},
	}
}

func (t *Map) Type() Type {
	return t.typ
}

func (t *Map) String() string {
	values := make([]string, 0, len(t.values))
	for k, v := range t.values {
		values = append(values,
			fmt.Sprintf("%s: %s", k.String(), v.String()),
		)
	}

	return fmt.Sprintf("{%s}", strings.Join(values, ", "))
}

func (t *Map) Values() map[Value]Value {
	return t.values
}

func (t *Map) Set(k, v Value) *Map {
	t.values[k] = v
	return t
}

func (t *Map) Get(k Value) Value {
	return t.values[k]
}
