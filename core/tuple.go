package core

import (
	"fmt"
	"slices"
	"strings"
)

type ID []string

func NewID(raw ...string) ID {
	return ID(raw)
}

func (i ID) String() string {
	return strings.Join(i, "/")
}

func (i ID) Equals(other ID) bool {
	return slices.Equal(i, other)
}

type Attr []string

func NewAttr(raw ...string) Attr {
	return Attr(raw)
}

func (a Attr) String() string {
	return strings.Join(a, ".")
}

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - ID: The ID of the tuple.
// - Attr: Name of the attribute.
// - Value: Value of the attribute.
type Tuple struct {
	id    ID
	attr  Attr
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(id, attr, value any) *Tuple {
	return &Tuple{
		id:    newID(id),
		attr:  newAttr(attr),
		value: NewValue(value),
	}
}

// Key returns a reference to the tuple.
func (t *Tuple) Key() *Key {
	return NewKey(t.id, t.attr)
}

// ID returns the ID of the tuple.
func (t *Tuple) ID() ID {
	return t.id
}

// Attr returns the attribute name.
func (t *Tuple) Attr() Attr {
	return t.attr
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}

// GoString returns Go syntax of the tuple.
func (t *Tuple) GoString() string {
	return fmt.Sprintf("Tuple(%s, %s, %#v)", t.id.String(), t.attr.String(), t.value)
}

func newID(id any) ID {
	switch v := id.(type) {
	case ID:
		return v
	case string:
		return NewID(v)
	case []string:
		return ID(v)
	default:
		panic(fmt.Sprintf("invalid ID: %v", id))
	}
}

func newAttr(attr any) Attr {
	switch v := attr.(type) {
	case Attr:
		return v
	case string:
		return Attr{v}
	case []string:
		return Attr(v)
	default:
		panic(fmt.Sprintf("invalid Attr: %v", attr))
	}
}
