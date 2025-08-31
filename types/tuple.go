package types

import (
	"fmt"
)

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - Key: A unique reference to the tuple.
// - Value: The value of the attribute.
// - Options: Options are key-value pairs
type Tuple struct {
	key   *Key
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(k any, v any) *Tuple {
	var key *Key

	switch kt := k.(type) {
	case string:
		key = NewKey(kt)
	case []string:
		key = NewKey(kt...)
	case *Key:
		key = kt
	default:
		panic(fmt.Sprintf("invalid key type: %T", k))
	}

	return &Tuple{key: key, value: NewValue(v)}
}

// Key returns the key of the tuple.
func (t *Tuple) Key() *Key {
	return t.key
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}

// GoString returns Go syntax for the tuple.
func (t *Tuple) String() string {
	return fmt.Sprintf("Tuple(%s, %#v)", t.key.String(), t.value)
}
