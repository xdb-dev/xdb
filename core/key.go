package core

import (
	"fmt"
)

// Key is an unique reference to an attribute or a record.
type Key struct {
	id   ID
	attr Attr
}

// NewKey creates a new Key.
//
// Only the following patterns are supported:
// - NewKey("{id}")
// - NewKey("{id}", "{attr}")
// - NewKey(NewID("{id}"), "{attr}")
// - NewKey(NewID("{id}"), NewAttr("{attr}")
//
// Panics if the key is invalid.
func NewKey(parts ...any) *Key {
	if len(parts) == 0 {
		return nil
	}

	switch len(parts) {
	case 1:
		return &Key{id: newID(parts[0])}
	case 2:
		return &Key{id: newID(parts[0]), attr: newAttr(parts[1])}
	default:
		panic(fmt.Sprintf("invalid key: %v", parts))
	}
}

// Value creates a new [Tuple] with the Key.
func (k *Key) Value(value any) *Tuple {
	return NewTuple(k.id, k.attr, value)
}

// String returns the key encoded as a string.
func (k *Key) String() string {
	if len(k.attr) > 0 {
		return fmt.Sprintf("%s/%s", k.id.String(), k.attr.String())
	}

	return k.id.String()
}

// GoString returns Go syntax of the Key.
func (k *Key) GoString() string {
	return fmt.Sprintf("Key(%s)", k.String())
}

// ID returns the ID of the Key.
func (k *Key) ID() ID {
	return k.id
}

// Attr returns the attribute name in the Key.
func (k *Key) Attr() Attr {
	return k.attr
}
