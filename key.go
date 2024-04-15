package xdb

import "fmt"

// Key is a unique identifier for a record.
type Key struct {
	kind string
	id   string
}

// NewKey creates a new key.
func NewKey(kind, id string) *Key {
	return &Key{
		kind: kind,
		id:   id,
	}
}

// Kind returns the kind of the key.
func (k *Key) Kind() string {
	return k.kind
}

// ID returns the id of the key.
func (k *Key) ID() string {
	return k.id
}

// String returns a readable form of the key.
func (k *Key) String() string {
	return fmt.Sprintf("%s:%s", k.kind, k.id)
}
