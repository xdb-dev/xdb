package xdb

import (
	"time"
)

type Empty struct{}

type Value interface {
	int | int32 | uint32 | int64 | uint64 |
		float32 | float64 | string | bool | []byte |
		time.Time | Empty |
		[]int | []int32 | []uint32 | []int64 | []uint64 |
		[]float32 | []float64 | []string | []bool
}

// Tuple is a key-attribute-value tuple.
type Tuple struct {
	key   *Key
	name  string
	value any
}

// NewTuple creates a new tuple.
func NewTuple[T Value](key *Key, name string, value T) *Tuple {
	return &Tuple{
		key:   key,
		name:  name,
		value: value,
	}
}

func (t *Tuple) Key() *Key {
	return t.key
}

func (t *Tuple) Name() string {
	return t.name
}

func (t *Tuple) Value() any {
	return t.value
}

func (t *Tuple) Hash() string {
	return t.key.String() + "/" + t.name
}
