package types

import (
	"fmt"
	"strings"
)

// Key is a unique reference to a tuple, edge or record.
type Key struct {
	parts []string
}

// NewKey creates a new Key.
func NewKey(parts ...string) *Key {
	return &Key{parts: parts}
}

// New creates a new [Key] by appending the given parts to the current key.
func (k *Key) New(parts ...string) *Key {
	return &Key{parts: append(k.parts, parts...)}
}

// Value creates a new [Tuple] with the current key.
func (k *Key) Value(value any) *Tuple {
	return NewTuple(k, value)
}

// String returns key encoded as a string.
func (k *Key) String() string {
	return strings.Join(k.parts, "/")
}

// GoString returns Go syntax for the key.
func (k *Key) GoString() string {
	return fmt.Sprintf("Key(%s)", k.String())
}

// Unwrap returns the parts of the Key.
func (k *Key) Unwrap() []string {
	return k.parts
}

// ParseKey parses a string into a [Key].
func ParseKey(s string) *Key {
	return NewKey(strings.Split(s, "/")...)
}
