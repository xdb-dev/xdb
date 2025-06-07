package types

import (
	"fmt"
	"reflect"
	"time"
)

// Value represents an attribute value.
type Value interface {
	Type() Type
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

	if iv.Kind() == reflect.Ptr {
		iv = iv.Elem()
	}

	return newValue(iv)
}

func newValue(iv reflect.Value) (Value, error) {
	switch iv.Kind() {
	case reflect.Bool:
		b := Bool(iv.Interface().(bool))
		return &b, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := Int64(iv.Interface().(int64))
		return &i, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := Uint64(iv.Interface().(uint64))
		return &u, nil
	case reflect.Float32, reflect.Float64:
		f := Float64(iv.Interface().(float64))
		return &f, nil
	case reflect.String:
		s := String(iv.Interface().(string))
		return &s, nil
	case reflect.Slice:
		if iv.Len() == 0 {
			return nil, nil
		}

		arr := make([]Value, iv.Len())
		for i := 0; i < iv.Len(); i++ {
			v, err := NewSafeValue(iv.Index(i).Interface())
			if err != nil {
				return nil, err
			}

			arr[i] = v
		}

		return NewArray(NewArrayType(arr[0].Type()), arr...), nil
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

		mp := NewMap(key1.Type(), value1.Type())

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
		return nil, fmt.Errorf("unsupported value type: %T", iv.Interface())
	}
}

type (
	Bool    bool
	Int64   int64
	Uint64  uint64
	Float64 float64
	String  string
	Bytes   []byte
	Time    time.Time
)

func (b *Bool) Type() Type {
	return typeBoolean
}

func (t *Int64) Type() Type {
	return typeInteger
}

func (t *Uint64) Type() Type {
	return typeUnsigned
}

func (t *Float64) Type() Type {
	return typeFloat
}

func (t *String) Type() Type {
	return typeString
}

func (t *Bytes) Type() Type {
	return typeBytes
}

func (t *Time) Type() Type {
	return typeTime
}

type Array struct {
	typ    *ArrayType
	values []Value
}

func NewArray(t Type, values ...Value) *Array {
	return &Array{
		typ:    NewArrayType(t),
		values: values,
	}
}

func (t *Array) Type() Type {
	return t.typ
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
	typ    *MapType
	values map[Value]Value
}

func NewMap(kt, vt Type) *Map {
	return &Map{
		typ:    NewMapType(kt, vt),
		values: map[Value]Value{},
	}
}

func (t *Map) Type() Type {
	return t.typ
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
