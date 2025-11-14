package core

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidID is returned when an invalid ID is encountered.
	ErrInvalidID = errors.New("[xdb/core] invalid ID")
)

// ID is an unique identifier for a record.
type ID struct {
	id string
}

// NewID creates a new ID from a string.
// Panics if the ID is invalid (contains characters outside [a-zA-Z0-9._/-]).
func NewID(raw string) *ID {
	id, err := ParseID(raw)
	if err != nil {
		panic(err)
	}
	return id
}

// ParseID parses a string into an ID.
// Returns [ErrInvalidID] if the ID is invalid.
func ParseID(raw string) (*ID, error) {
	if !isValidComponent(raw) {
		return nil, ErrInvalidID
	}
	return &ID{id: raw}, nil
}

// String returns the ID.
func (id *ID) String() string {
	if id == nil {
		return ""
	}
	return id.id
}

// Equals returns true if this ID is equal to the other ID.
func (id *ID) Equals(other *ID) bool {
	if id == nil && other == nil {
		return true
	}
	if id == nil || other == nil {
		return false
	}
	return id.id == other.id
}
