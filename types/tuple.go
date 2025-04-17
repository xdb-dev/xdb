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
	kind  string
	id    string
	attr  string
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(kind string, id string, attr string, value any) *Tuple {
	return &Tuple{kind: kind, id: id, attr: attr, value: NewValue(value)}
}

// Key returns a reference to the tuple.
func (t *Tuple) Key() *Key {
	return NewKey(t.kind, t.id, t.attr)
}

// Kind returns the kind of the tuple.
func (t *Tuple) Kind() string {
	return t.kind
}

// ID returns the ID of the tuple.
func (t *Tuple) ID() string {
	return t.id
}

// Attr returns the attribute name.
func (t *Tuple) Attr() string {
	return t.attr
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}

// String returns the string representation of the tuple.
func (t *Tuple) String() string {
	return fmt.Sprintf("Tuple(%s, %s, %s, %v)", t.kind, t.id, t.attr, t.value)
}
