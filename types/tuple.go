package types

import (
	"fmt"
)

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - Kind: The kind of the tuple.
// - ID: The ID of the tuple.
// - Attr: Name of the attribute.
// - Value: Value of the attribute.
// - Options: Options are key-value pairs
type Tuple struct {
	key   *Key
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(kind string, id string, attr string, value any) *Tuple {
	return newTuple(NewKey(kind, id, attr), value)
}

func newTuple(key *Key, value any) *Tuple {
	return &Tuple{key: key, value: NewValue(value)}
}

// Key returns a reference to the tuple.
func (t *Tuple) Key() *Key {
	return t.key
}

// Kind returns the kind of the tuple.
// deprecated: use [Tuple.Key] instead.
func (t *Tuple) Kind() string {
	return t.key.Kind()
}

// ID returns the ID of the tuple.
// deprecated: use [Tuple.Key] instead.
func (t *Tuple) ID() string {
	return t.key.ID()
}

// Attr returns the attribute name.
func (t *Tuple) Attr() string {
	parts := t.key.Unwrap()

	return parts[len(parts)-1]
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}

// GoString returns Go syntax of the tuple.
func (t *Tuple) GoString() string {
	return fmt.Sprintf("Tuple(%s, %#v)", t.key.String(), t.value)
}
