package core

import (
	"github.com/gojekfarm/xtools/errors"
	"slices"
)

var (
	// ErrInvalidSchema is returned when an invalid Schema is encountered.
	ErrInvalidSchema = errors.New("[xdb/core] invalid Schema")
)

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

// Mode defines how records are validated against the schema.
type Mode string

const (
	// ModeFlexible represents schemaless collections where
	// records can have arbitrary attributes.
	ModeFlexible Mode = "flexible"

	// ModeStrict represents structured collections where
	// records must have attributes defined in the schema.
	ModeStrict Mode = "strict"
)

// SchemaDef defines the structure and validation rules for records.
// It provides metadata, field definitions, and record-level constraints.
type SchemaDef struct {
	// Name is the schema name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Version tracks schema evolution (e.g., "1.0.0").
	Version string

	// Mode defines how records are validated against the schema.
	Mode Mode

	// Fields is a list of field schemas.
	// Use hierarchical paths for nested fields (e.g., "profile.email").
	Fields []*FieldDef

	// Required is a list of field names that are required.
	Required []string
}

// Clone returns a deep copy of the SchemaDef.
func (s *SchemaDef) Clone() *SchemaDef {
	clone := &SchemaDef{
		Name:        s.Name,
		Description: s.Description,
		Version:     s.Version,
		Fields:      make([]*FieldDef, 0, len(s.Fields)),
		Required:    slices.Clone(s.Required),
	}
	for _, field := range s.Fields {
		clone.Fields = append(clone.Fields, field.Clone())
	}
	return clone
}

// GetField returns the field definition for the given path.
func (s *SchemaDef) GetField(path string) *FieldDef {
	for _, field := range s.Fields {
		if field.Name == path {
			return field
		}
	}
	return nil
}

// FieldDef defines the definition for a single field.
type FieldDef struct {
	// Name is the field name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Type specifies the field's full type
	Type Type

	// Default specifies the default value when the field is missing.
	Default *Value
}

// Equals returns true if this FieldSchema is equal to the other FieldSchema.
func (f *FieldDef) Equals(other *FieldDef) bool {
	return f.Name == other.Name &&
		f.Description == other.Description &&
		f.Type.Equals(other.Type)
}

// Clone returns a deep copy of the FieldDef.
func (f *FieldDef) Clone() *FieldDef {
	return &FieldDef{
		Name:        f.Name,
		Description: f.Description,
		Type:        f.Type,
		Default:     f.Default,
	}
}
