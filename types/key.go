package types

import (
	"fmt"
	"strings"
)

// Key is an unique reference to one of:
// - tuple
// - edge
// - record
type Key struct {
	parts []string
}

// NewKey creates a new Key.
//
// NewKey("User", "123") is a reference to a record.
// NewKey("User", "123", "name") is a reference to a tuple.
// NewKey("User", "123", "follows", "Post", "123") is a reference to an edge.
func NewKey(parts ...string) *Key {
	return &Key{parts: parts}
}

// String returns the string representation of the Key.
func (k *Key) String() string {
	return fmt.Sprintf("Key(%s)", strings.Join(k.parts, "/"))
}

// Kind returns the kind of the Key.
func (k *Key) Kind() string {
	return k.parts[0]
}

// ID returns the ID of the Key.
func (k *Key) ID() string {
	return k.parts[1]
}

// Attr returns the attribute name in the Key.
func (k *Key) Attr() string {
	return k.parts[2]
}
