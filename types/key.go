package types

import (
	"fmt"
	"strings"
)

// Key is an unique reference to an attribute or a record.
type Key struct {
	parts []string
}

// NewKey creates a new Key.
func NewKey(parts ...string) *Key {
	if len(parts) == 0 {
		return nil
	}

	return &Key{parts: parts}
}

// With returns a new Key with the appended parts.
func (k *Key) With(parts ...string) *Key {
	return NewKey(append(k.parts, parts...)...)
}

// Value creates a new [Tuple] with the Key.
func (k *Key) Value(value any) *Tuple {
	return newTuple(k, value)
}

// Unwrap returns the key parts.
func (k *Key) Unwrap() []string {
	return k.parts
}

// String returns the key encoded as a string.
func (k *Key) String() string {
	return strings.Join(k.parts, "/")
}

// GoString returns Go syntax of the Key.
func (k *Key) GoString() string {
	return fmt.Sprintf("Key(%s)", k.String())
}

// Kind returns the kind of the Key.
// deprecated: use [Key.Unwrap] instead.
func (k *Key) Kind() string {
	return k.parts[0]
}

// ID returns the ID of the Key.
// deprecated: use [Key.Unwrap] instead.
func (k *Key) ID() string {
	return k.parts[1]
}

// Attr returns the attribute name in the Key.
// deprecated: use [Key.Unwrap] instead.
func (k *Key) Attr() string {
	if len(k.parts) < 3 {
		return ""
	}

	return k.parts[2]
}
