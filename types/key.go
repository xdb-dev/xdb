package types

import "strings"

// Key is an immutable unique reference to one of:
// - Record
// - Tuple
type Key struct {
	parts []string
}

// NewKey creates a new key from the given values.
func NewKey(parts ...string) *Key {
	return &Key{parts: parts}
}

// String returns the string representation of the key.
func (k *Key) String() string {
	return strings.Join(k.parts, "/")
}

// Kind returns the type of the record.
func (k *Key) Kind() string {
	return k.parts[0]
}

// ID returns the unique ID of the record.
func (k *Key) ID() string {
	return k.parts[1]
}

// Name returns the attribute name represented by the key.
func (k *Key) Name() string {
	return k.parts[2]
}
