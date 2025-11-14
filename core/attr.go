package core

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidAttr is returned when an invalid Attr is encountered.
	ErrInvalidAttr = errors.New("[xdb/core] invalid Attr")
)

// Attr is a name of an attribute.
type Attr struct {
	name string
}

// NewAttr creates a new attribute name.
// Panics if the attribute name is invalid
// (contains characters outside [a-zA-Z0-9._/-]).
func NewAttr(raw string) *Attr {
	attr, err := ParseAttr(raw)
	if err != nil {
		panic(err)
	}
	return attr
}

// ParseAttr parses a string into an Attr.
// Returns [ErrInvalidAttr] if the Attr is invalid.
func ParseAttr(raw string) (*Attr, error) {
	if !isValidComponent(raw) {
		return nil, ErrInvalidAttr
	}
	return &Attr{name: raw}, nil
}

// String returns the Attr.
func (a *Attr) String() string { return a.name }

// Equals returns true if this Attr is equal to the other Attr.
func (a *Attr) Equals(other *Attr) bool { return a.name == other.name }
