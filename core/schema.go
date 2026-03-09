package core

import (
	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidSchema is returned when an invalid Schema is encountered.
var ErrInvalidSchema = errors.New("[xdb/core] invalid Schema")

// Schema is a name of a schema.
type Schema struct {
	name string
}

// NewSchema creates a new Schema.
// Panics if the Schema name is invalid
// (contains characters outside [a-zA-Z0-9._/-]).
func NewSchema(raw string) *Schema {
	schema, err := ParseSchema(raw)
	if err != nil {
		panic(err)
	}
	return schema
}

// ParseSchema parses a string into a Schema.
// Returns [ErrInvalidSchema] if the Schema is invalid.
func ParseSchema(raw string) (*Schema, error) {
	if !isValidComponent(raw) {
		return nil, ErrInvalidSchema
	}
	return &Schema{name: raw}, nil
}

// String returns the Schema.
func (s *Schema) String() string { return s.name }

// Equals returns true if this Schema is equal to the other Schema.
func (s *Schema) Equals(other *Schema) bool { return s.name == other.name }
