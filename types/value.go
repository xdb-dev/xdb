package types

import (
	"fmt"
	"reflect"
)

// Value represents an attribute value.
type Value struct {
	tid      TypeID
	val      any
	repeated bool
}

// NewValue creates a new value of the given type.
func NewValue(value any) *Value {
	tid, repeated := TypeIDOf(value)
	return &Value{tid: tid, val: value, repeated: repeated}
}

// Unwrap returns the value as is.
func (v *Value) Unwrap() any {
	return v.val
}

// TypeID returns the type ID of the value.
func (v *Value) TypeID() TypeID {
	return v.tid
}

// Repeated returns whether the value is an array.
func (v *Value) Repeated() bool {
	return v.repeated
}

// String returns a string representation.
func (v *Value) String() string {
	return fmt.Sprintf("Value(%s, %v)", v.tid.String(), v.val)
}

// TypeIDOf returns the TypeID and whether the value is an array.
func TypeIDOf(value any) (TypeID, bool) {
	kind := reflect.TypeOf(value).Kind()
	switch kind {
	case reflect.Array, reflect.Slice:
		first := reflect.ValueOf(value).Index(0)
		if first.Kind() == reflect.Uint8 {
			return BytesType, false
		}

		t, _ := TypeIDOf(first.Interface())
		return t, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return IntegerType, false
	case reflect.Float32, reflect.Float64:
		return FloatType, false
	case reflect.String:
		return StringType, false
	case reflect.Bool:
		return BooleanType, false
	case reflect.Struct:
		switch reflect.TypeOf(value).String() {
		case "time.Time":
			return TimeType, false
		case "types.Point":
			return PointType, false
		default:
			return UnknownType, false
		}
	default:
		return UnknownType, false
	}
}
