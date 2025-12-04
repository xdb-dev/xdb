package core

import (
	"strconv"
	"time"

	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"
)

// ErrCastFailed is returned when a value cannot
// be converted to the desired type.
var ErrCastFailed = errors.New("[xdb/core] cast failed")

// ToBool returns the value as a bool.
func (v *Value) ToBool() bool {
	b, err := toBool(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return b
}

// ToInt returns the value as an int64.
func (v *Value) ToInt() int64 {
	i, err := toInt64(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return i
}

// ToUint returns the value as a uint64.
func (v *Value) ToUint() uint64 {
	u, err := toUint64(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return u
}

// ToFloat returns the value as a float64.
func (v *Value) ToFloat() float64 {
	f, err := toFloat64(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return f
}

// ToString returns the value as a string.
func (v *Value) ToString() string {
	s, err := toString(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return s
}

// ToBytes returns the value as a []byte.
func (v *Value) ToBytes() []byte {
	b, err := toBytes(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return b
}

// ToTime returns the value as a time.Time.
func (v *Value) ToTime() time.Time {
	t, err := toTime(v)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return t
}

// ToBoolArray returns the value as a []bool.
func (v *Value) ToBoolArray() []bool {
	arr, err := castArray(v, toBool)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToIntArray returns the value as a []int64.
func (v *Value) ToIntArray() []int64 {
	arr, err := castArray(v, toInt64)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToUintArray returns the value as a []uint64.
func (v *Value) ToUintArray() []uint64 {
	arr, err := castArray(v, toUint64)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToFloatArray returns the value as a []float64.
func (v *Value) ToFloatArray() []float64 {
	arr, err := castArray(v, toFloat64)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToStringArray returns the value as a []string.
func (v *Value) ToStringArray() []string {
	arr, err := castArray(v, toString)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToBytesArray returns the value as a [][]byte.
func (v *Value) ToBytesArray() [][]byte {
	arr, err := castArray(v, toBytes)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToTimeArray returns the value as a []time.Time.
func (v *Value) ToTimeArray() []time.Time {
	arr, err := castArray(v, toTime)
	if err != nil {
		panic(errors.Wrap(err, "value", v.String()))
	}
	return arr
}

// ToBool returns the tuple's value as a bool.
func (t *Tuple) ToBool() bool {
	return t.value.ToBool()
}

// ToBoolArray returns the tuple's value as a []bool.
func (t *Tuple) ToBoolArray() []bool {
	return t.value.ToBoolArray()
}

// ToInt returns the tuple's value as an int64.
func (t *Tuple) ToInt() int64 {
	return t.value.ToInt()
}

// ToIntArray returns the tuple's value as a []int64.
func (t *Tuple) ToIntArray() []int64 {
	return t.value.ToIntArray()
}

// ToUint returns the tuple's value as a uint64.
func (t *Tuple) ToUint() uint64 {
	return t.value.ToUint()
}

// ToUintArray returns the tuple's value as a []uint64.
func (t *Tuple) ToUintArray() []uint64 {
	return t.value.ToUintArray()
}

// ToFloat returns the tuple's value as a float64.
func (t *Tuple) ToFloat() float64 {
	return t.value.ToFloat()
}

// ToFloatArray returns the tuple's value as a []float64.
func (t *Tuple) ToFloatArray() []float64 {
	return t.value.ToFloatArray()
}

// ToString returns the tuple's value as a string.
func (t *Tuple) ToString() string {
	return t.value.ToString()
}

// ToStringArray returns the tuple's value as a []string.
func (t *Tuple) ToStringArray() []string {
	return t.value.ToStringArray()
}

// ToBytes returns the tuple's value as a []byte.
func (t *Tuple) ToBytes() []byte {
	return t.value.ToBytes()
}

// ToBytesArray returns the tuple's value as a [][]byte.
func (t *Tuple) ToBytesArray() [][]byte {
	return t.value.ToBytesArray()
}

// ToTime returns the tuple's value as a time.Time.
func (t *Tuple) ToTime() time.Time {
	return t.value.ToTime()
}

// ToTimeArray returns the tuple's value as a []time.Time.
func (t *Tuple) ToTimeArray() []time.Time {
	return t.value.ToTimeArray()
}

func toBool(v *Value) (bool, error) {
	if v == nil {
		return false, nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		return v.data.(bool), nil
	case TIDInteger:
		return v.data.(int64) != 0, nil
	case TIDUnsigned:
		return v.data.(uint64) != 0, nil
	case TIDFloat:
		return v.data.(float64) != 0, nil
	case TIDString:
		return cast.ToBool(v.data.(string)), nil
	case TIDBytes:
		return cast.ToBool(string(v.data.([]byte))), nil
	default:
		return false, ErrCastFailed
	}
}

func toInt64(v *Value) (int64, error) {
	if v == nil {
		return 0, nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		if v.data.(bool) {
			return 1, nil
		}
		return 0, nil
	case TIDInteger:
		return v.data.(int64), nil
	case TIDUnsigned:
		return int64(v.data.(uint64)), nil
	case TIDFloat:
		return int64(v.data.(float64)), nil
	case TIDString:
		return cast.ToInt64(v.data.(string)), nil
	case TIDBytes:
		return cast.ToInt64(string(v.data.([]byte))), nil
	default:
		return 0, ErrCastFailed
	}
}

func toUint64(v *Value) (uint64, error) {
	if v == nil {
		return 0, nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		if v.data.(bool) {
			return 1, nil
		}
		return 0, nil
	case TIDInteger:
		return uint64(v.data.(int64)), nil
	case TIDUnsigned:
		return v.data.(uint64), nil
	case TIDFloat:
		return uint64(v.data.(float64)), nil
	case TIDString:
		return cast.ToUint64(v.data.(string)), nil
	case TIDBytes:
		return cast.ToUint64(string(v.data.([]byte))), nil
	default:
		return 0, ErrCastFailed
	}
}

func toFloat64(v *Value) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		if v.data.(bool) {
			return 1, nil
		}
		return 0, nil
	case TIDInteger:
		return float64(v.data.(int64)), nil
	case TIDUnsigned:
		return float64(v.data.(uint64)), nil
	case TIDFloat:
		return v.data.(float64), nil
	case TIDString:
		return cast.ToFloat64(v.data.(string)), nil
	case TIDBytes:
		return cast.ToFloat64(string(v.data.([]byte))), nil
	default:
		return 0, ErrCastFailed
	}
}

func toString(v *Value) (string, error) {
	if v == nil {
		return "", nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		return strconv.FormatBool(v.data.(bool)), nil
	case TIDInteger:
		return strconv.FormatInt(v.data.(int64), 10), nil
	case TIDUnsigned:
		return strconv.FormatUint(v.data.(uint64), 10), nil
	case TIDFloat:
		return strconv.FormatFloat(v.data.(float64), 'f', -1, 64), nil
	case TIDString:
		return v.data.(string), nil
	case TIDBytes:
		return string(v.data.([]byte)), nil
	default:
		return "", ErrCastFailed
	}
}

func toBytes(v *Value) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	switch v.Type().ID() {
	case TIDBoolean:
		return []byte(strconv.FormatBool(v.data.(bool))), nil
	case TIDInteger:
		return []byte(strconv.FormatInt(v.data.(int64), 10)), nil
	case TIDUnsigned:
		return []byte(strconv.FormatUint(v.data.(uint64), 10)), nil
	case TIDFloat:
		return []byte(strconv.FormatFloat(v.data.(float64), 'f', -1, 64)), nil
	case TIDString:
		return []byte(v.data.(string)), nil
	case TIDBytes:
		return v.data.([]byte), nil
	default:
		return nil, ErrCastFailed
	}
}

func toTime(v *Value) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	switch v.Type().ID() {
	case TIDInteger:
		return time.UnixMilli(v.data.(int64)), nil
	case TIDUnsigned:
		return time.UnixMilli(int64(v.data.(uint64))), nil
	case TIDFloat:
		return time.UnixMilli(int64(v.data.(float64))), nil
	case TIDString:
		return time.Parse(time.RFC3339, v.data.(string))
	case TIDBytes:
		return time.Parse(time.RFC3339, string(v.data.([]byte)))
	case TIDTime:
		return v.data.(time.Time), nil
	default:
		return time.Time{}, ErrCastFailed
	}
}

func castArray[T any](v *Value, f func(*Value) (T, error)) ([]T, error) {
	if v == nil {
		return nil, nil
	}

	if v.Type().ID() != TIDArray {
		return nil, ErrCastFailed
	}

	arr := v.data.([]*Value)
	result := make([]T, len(arr))
	for i, elem := range arr {
		val, err := f(elem)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}
