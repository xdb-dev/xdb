package types

import (
	"time"

	"github.com/spf13/cast"
)

// Value represents an attribute value.
type Value struct {
	any
}

// NewValue creates a new value of the given type.
func NewValue[T Type](value T) *Value {
	return &Value{value}
}

// ToString returns the value as a string.
func (v *Value) ToString() string {
	return cast.ToString(v.any)
}

// ToStringList returns the value as a list of strings.
func (v *Value) ToStringList() []string {
	return cast.ToStringSlice(v.any)
}

// ToInt returns the value as an int64.
func (v *Value) ToInt() int64 {
	return cast.ToInt64(v.any)
}

// ToFloat returns the value as a float64.
func (v *Value) ToFloat() float64 {
	return cast.ToFloat64(v.any)
}

// ToBool returns the value as a boolean.
func (v *Value) ToBool() bool {
	return cast.ToBool(v.any)
}

// ToTime returns the value as a time.Time.
func (v *Value) ToTime() time.Time {
	return cast.ToTime(v.any)
}

// ToPoint returns the value as a Point.
func (v *Value) ToPoint() Point {
	return v.any.(Point)
}

// Unwrap returns the value as is.
func (v *Value) Unwrap() any {
	return v.any
}
