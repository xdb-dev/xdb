package types

import (
	"time"

	"github.com/spf13/cast"
)

// ToInt returns the tuple's value as an int64.
func (t *Tuple) ToInt() int64 {
	return cast.ToInt64(t.value)
}

// ToFloat returns the tuple's value as a float64.
func (t *Tuple) ToFloat() float64 {
	return cast.ToFloat64(t.value)
}

// ToBool returns the tuple's value as a bool.
func (t *Tuple) ToBool() bool {
	return cast.ToBool(t.value)
}

// ToTime returns the tuple's value as a time.Time.
func (t *Tuple) ToTime() time.Time {
	return cast.ToTime(t.value)
}
