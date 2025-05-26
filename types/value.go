package types

import (
	"fmt"
	"reflect"
)

// Value represents an attribute value.
type Value struct {
	tid      TypeID
	repeated bool
	val      any
}

// NewValue creates a new value of the given type.
func NewValue(value any) *Value {
	if value == nil {
		return nil
	}

	if vv, ok := value.(*Value); ok {
		return vv
	}

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
			return TypeBytes, false
		}

		t, _ := TypeIDOf(first.Interface())
		return t, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInteger, false
	case reflect.Float32, reflect.Float64:
		return TypeFloat, false
	case reflect.String:
		return TypeString, false
	case reflect.Bool:
		return TypeBoolean, false
	case reflect.Struct:
		switch reflect.TypeOf(value).String() {
		case "time.Time":
			return TypeTime, false
		case "types.Point":
			return TypePoint, false
		default:
			return TypeUnknown, false
		}
	default:
		return TypeUnknown, false
	}
}
