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

// NewID creates a new ID.
// Panics if the ID is invalid.
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
func (id *ID) String() string { return id.id }

// Equals returns true if this ID is equal to the other ID.
func (id *ID) Equals(other *ID) bool { return id.id == other.id }
