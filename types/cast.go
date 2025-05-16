package types

import (
	"time"

	"github.com/spf13/cast"
)

// ToInt returns the value as an int64.
func (v *Value) ToInt() int64 {
	return cast.ToInt64(v.val)
}

// ToIntSlice returns the value as a []int64.
func (v *Value) ToIntSlice() []int64 {
	return v.val.([]int64)
}

// ToFloat returns the value as a float64.
func (v *Value) ToFloat() float64 {
	return cast.ToFloat64(v.val)
}

// ToFloatSlice returns the value as a []float64.
func (v *Value) ToFloatSlice() []float64 {
	return v.val.([]float64)
}

// ToString returns the value as a string.
func (v *Value) ToString() string {
	return cast.ToString(v.val)
}

// ToStringSlice returns the value as a []string.
func (v *Value) ToStringSlice() []string {
	return cast.ToStringSlice(v.val)
}

// ToBool returns the value as a bool.
func (v *Value) ToBool() bool {
	return cast.ToBool(v.val)
}

// ToBoolSlice returns the value as a []bool.
func (v *Value) ToBoolSlice() []bool {
	return v.val.([]bool)
}

// ToBytes returns the value as a []byte.
func (v *Value) ToBytes() []byte {
	return v.val.([]byte)
}

// ToBytesSlice returns the value as a [][]byte.
func (v *Value) ToBytesSlice() [][]byte {
	return v.val.([][]byte)
}

// ToTime returns the value as a time.Time.
func (v *Value) ToTime() time.Time {
	return cast.ToTime(v.val)
}

// ToTimeSlice returns the value as a []time.Time.
func (v *Value) ToTimeSlice() []time.Time {
	return v.val.([]time.Time)
}

// ToPoint returns the value as a Point.
func (v *Value) ToPoint() Point {
	return v.val.(Point)
}

// ToPointSlice returns the value as a []Point.
func (v *Value) ToPointSlice() []Point {
	return v.val.([]Point)
}

// ToInt returns the tuple's value as an int64.
func (t *Tuple) ToInt() int64 {
	return t.value.ToInt()
}

// ToIntSlice returns the tuple's value as a []int64.
func (t *Tuple) ToIntSlice() []int64 {
	return t.value.ToIntSlice()
}

// ToFloat returns the tuple's value as a float64.
func (t *Tuple) ToFloat() float64 {
	return t.value.ToFloat()
}

// ToFloatSlice returns the tuple's value as a []float64.
func (t *Tuple) ToFloatSlice() []float64 {
	return t.value.ToFloatSlice()
}

// ToBool returns the tuple's value as a bool.
func (t *Tuple) ToBool() bool {
	return t.value.ToBool()
}

// ToBoolSlice returns the tuple's value as a []bool.
func (t *Tuple) ToBoolSlice() []bool {
	return t.value.ToBoolSlice()
}

// ToString returns the tuple's value as a string.
func (t *Tuple) ToString() string {
	return t.value.ToString()
}

// ToStringSlice returns the tuple's value as a []string.
func (t *Tuple) ToStringSlice() []string {
	return t.value.ToStringSlice()
}

// ToBytes returns the tuple's value as a []byte.
func (t *Tuple) ToBytes() []byte {
	return t.value.ToBytes()
}

// ToBytesSlice returns the tuple's value as a [][]byte.
func (t *Tuple) ToBytesSlice() [][]byte {
	return t.value.ToBytesSlice()
}

// ToTime returns the tuple's value as a time.Time.
func (t *Tuple) ToTime() time.Time {
	return t.value.ToTime()
}

// ToTimeSlice returns the tuple's value as a []time.Time.
func (t *Tuple) ToTimeSlice() []time.Time {
	return t.value.ToTimeSlice()
}

// ToPoint returns the tuple's value as a Point.
func (t *Tuple) ToPoint() Point {
	return t.value.ToPoint()
}

// ToPointSlice returns the tuple's value as a []Point.
func (t *Tuple) ToPointSlice() []Point {
	return t.value.ToPointSlice()
}

func castArray[T any](arr []any, fn func(any) T) []T {
	res := make([]T, len(arr))
	for i, v := range arr {
		res[i] = fn(v)
	}
	return res
}
