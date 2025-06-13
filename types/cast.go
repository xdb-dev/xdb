package types

import (
	"strconv"
	"time"

	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"
)

var (
	// ErrCastFailed is returned when a value cannot
	// be converted to the desired type.
	ErrCastFailed = errors.New("xdb/types: cast failed")
)

// ToBool returns the tuple's value as a bool.
func (t *Tuple) ToBool() bool {
	b, err := toBool(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return b
}

// ToBoolArray returns the tuple's value as a []bool.
func (t *Tuple) ToBoolArray() []bool {
	arr, err := castArray(t.value, toBool)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToInt returns the tuple's value as an int64.
func (t *Tuple) ToInt() int64 {
	i, err := toInt64(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return i
}

// ToIntArray returns the tuple's value as a []int64.
func (t *Tuple) ToIntArray() []int64 {
	arr, err := castArray(t.value, toInt64)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToUint returns the tuple's value as a uint64.
func (t *Tuple) ToUint() uint64 {
	i, err := toUint64(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return i
}

// ToUintArray returns the tuple's value as a []uint64.
func (t *Tuple) ToUintArray() []uint64 {
	arr, err := castArray(t.value, toUint64)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToFloat returns the tuple's value as a float64.
func (t *Tuple) ToFloat() float64 {
	f, err := toFloat64(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return f
}

// ToFloatArray returns the tuple's value as a []float64.
func (t *Tuple) ToFloatArray() []float64 {
	arr, err := castArray(t.value, toFloat64)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToString returns the tuple's value as a string.
func (t *Tuple) ToString() string {
	s, err := toString(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return s
}

// ToStringArray returns the tuple's value as a []string.
func (t *Tuple) ToStringArray() []string {
	arr, err := castArray(t.value, toString)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToBytes returns the tuple's value as a []byte.
func (t *Tuple) ToBytes() []byte {
	b, err := toBytes(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return b
}

// ToBytesArray returns the tuple's value as a [][]byte.
func (t *Tuple) ToBytesArray() [][]byte {
	arr, err := castArray(t.value, toBytes)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

// ToTime returns the tuple's value as a time.Time.
func (t *Tuple) ToTime() time.Time {
	tm, err := toTime(t.value)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return tm
}

// ToTimeArray returns the tuple's value as a []time.Time.
func (t *Tuple) ToTimeArray() []time.Time {
	arr, err := castArray(t.value, toTime)
	if err != nil {
		panic(errors.Wrap(err, "tuple", t.String()))
	}
	return arr
}

func toBool(v Value) (bool, error) {
	switch v := v.(type) {
	case Bool:
		return bool(v), nil
	case Int64:
		return v != 0, nil
	case Uint64:
		return v != 0, nil
	case Float64:
		return v != 0, nil
	case String:
		return cast.ToBool(string(v)), nil
	case Bytes:
		return cast.ToBool(string(v)), nil
	default:
		return false, ErrCastFailed
	}
}

func toInt64(v Value) (int64, error) {
	switch v := v.(type) {
	case Bool:
		if bool(v) {
			return 1, nil
		}
		return 0, nil
	case Int64:
		return int64(v), nil
	case Uint64:
		return int64(v), nil
	case Float64:
		return int64(v), nil
	case String:
		return cast.ToInt64(string(v)), nil
	case Bytes:
		return cast.ToInt64(string(v)), nil
	default:
		return 0, ErrCastFailed
	}
}

func toUint64(v Value) (uint64, error) {
	switch v := v.(type) {
	case Bool:
		if bool(v) {
			return 1, nil
		}
		return 0, nil
	case Int64:
		return uint64(v), nil
	case Uint64:
		return uint64(v), nil
	case Float64:
		return uint64(v), nil
	case String:
		return cast.ToUint64(string(v)), nil
	case Bytes:
		return cast.ToUint64(string(v)), nil
	default:
		return 0, ErrCastFailed
	}
}

func toFloat64(v Value) (float64, error) {
	switch v := v.(type) {
	case Bool:
		if bool(v) {
			return 1, nil
		}
		return 0, nil
	case Int64:
		return float64(v), nil
	case Uint64:
		return float64(v), nil
	case Float64:
		return float64(v), nil
	case String:
		return cast.ToFloat64(string(v)), nil
	case Bytes:
		return cast.ToFloat64(string(v)), nil
	default:
		return 0, ErrCastFailed
	}
}

func toString(v Value) (string, error) {
	switch v := v.(type) {
	case Bool:
		return strconv.FormatBool(bool(v)), nil
	case Int64:
		return strconv.FormatInt(int64(v), 10), nil
	case Uint64:
		return strconv.FormatUint(uint64(v), 10), nil
	case Float64:
		return strconv.FormatFloat(float64(v), 'f', -1, 64), nil
	case String:
		return string(v), nil
	case Bytes:
		return string(v), nil
	default:
		return "", ErrCastFailed
	}
}

func toBytes(v Value) ([]byte, error) {
	switch v := v.(type) {
	case Bool:
		return []byte(strconv.FormatBool(bool(v))), nil
	case Int64:
		return []byte(strconv.FormatInt(int64(v), 10)), nil
	case Uint64:
		return []byte(strconv.FormatUint(uint64(v), 10)), nil
	case Float64:
		return []byte(strconv.FormatFloat(float64(v), 'f', -1, 64)), nil
	case String:
		return []byte(v), nil
	case Bytes:
		return []byte(v), nil
	default:
		return nil, ErrCastFailed
	}
}

func toTime(v Value) (time.Time, error) {
	switch v := v.(type) {
	case Int64:
		return time.UnixMilli(int64(v)), nil
	case Uint64:
		return time.UnixMilli(int64(v)), nil
	case Float64:
		return time.UnixMilli(int64(v)), nil
	case String:
		return time.Parse(time.RFC3339, string(v))
	case Bytes:
		return time.Parse(time.RFC3339, string(v))
	case Time:
		return time.Time(v), nil
	default:
		return time.Time{}, ErrCastFailed
	}
}

func castArray[T any](v Value, f func(Value) (T, error)) ([]T, error) {
	if v.Type().ID() != TypeArray {
		return nil, ErrCastFailed
	}

	arr := make([]T, len(v.(*Array).values))
	for i, v := range v.(*Array).values {
		var err error
		arr[i], err = f(v)
		if err != nil {
			return nil, err
		}
	}
	return arr, nil
}
