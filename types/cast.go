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
	switch v := t.value.(type) {
	case Bool:
		return bool(v)
	case Int64:
		return int64(v) > 0
	case Uint64:
		return uint64(v) > 0
	case Float64:
		return float64(v) > 0
	case String:
		return cast.ToString(v) != ""
	case Time:
		return !time.Time(v).IsZero()
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToInt returns the tuple's value as an int64.
func (t *Tuple) ToInt() int64 {
	switch v := t.value.(type) {
	case Bool:
		if bool(v) {
			return 1
		}
		return 0
	case Int64:
		return int64(v)
	case Uint64:
		return int64(v)
	case Float64:
		return int64(v)
	case String:
		return cast.ToInt64(string(v))
	case Time:
		return int64(time.Time(v).UnixMilli())
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToUint returns the tuple's value as a uint64.
func (t *Tuple) ToUint() uint64 {
	switch v := t.value.(type) {
	case Bool:
		if bool(v) {
			return 1
		}
		return 0
	case Int64:
		return uint64(v)
	case Uint64:
		return uint64(v)
	case Float64:
		return uint64(v)
	case String:
		return cast.ToUint64(string(v))
	case Time:
		return uint64(time.Time(v).UnixMilli())
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToFloat returns the tuple's value as a float64.
func (t *Tuple) ToFloat() float64 {
	switch v := t.value.(type) {
	case Bool:
		if bool(v) {
			return 1
		}
		return 0
	case Int64:
		return float64(v)
	case Uint64:
		return float64(v)
	case Float64:
		return float64(v)
	case String:
		return cast.ToFloat64(string(v))
	case Time:
		return float64(time.Time(v).UnixMilli())
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToString returns the tuple's value as a string.
func (t *Tuple) ToString() string {
	switch v := t.value.(type) {
	case Bool:
		return strconv.FormatBool(bool(v))
	case Int64:
		return strconv.FormatInt(int64(v), 10)
	case Uint64:
		return strconv.FormatUint(uint64(v), 10)
	case Float64:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case String:
		return string(v)
	case Bytes:
		return string(v)
	case Time:
		return time.Time(v).Format(time.RFC3339)
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToBytes returns the tuple's value as a []byte.
func (t *Tuple) ToBytes() []byte {
	switch v := t.value.(type) {
	case Bool:
		return []byte(strconv.FormatBool(bool(v)))
	case Int64:
		return []byte(strconv.FormatInt(int64(v), 10))
	case Uint64:
		return []byte(strconv.FormatUint(uint64(v), 10))
	case Float64:
		return []byte(strconv.FormatFloat(float64(v), 'f', -1, 64))
	case String:
		return []byte(v)
	case Bytes:
		return []byte(v)
	case Time:
		return []byte(time.Time(v).Format(time.RFC3339))
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}

// ToTime returns the tuple's value as a time.Time.
func (t *Tuple) ToTime() time.Time {
	switch v := t.value.(type) {
	case Int64:
		return time.UnixMilli(int64(v)).UTC()
	case Uint64:
		return time.UnixMilli(int64(v)).UTC()
	case Float64:
		return time.UnixMilli(int64(v)).UTC()
	case String:
		t, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			panic(errors.Wrap(err, "tuple", t.String()))
		}
		return t.UTC()
	case Bytes:
		t, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			panic(errors.Wrap(err, "tuple", t.String()))
		}
		return t.UTC()
	case Time:
		return time.Time(v)
	default:
		panic(errors.Wrap(ErrCastFailed,
			"tuple", t.String(),
		))
	}
}
