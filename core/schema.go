package core

import (
	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrEmptyRecord is returned when a record is nil.
	ErrEmptyRecord = errors.New("[xdb/core] record is nil")
	// ErrEmptyTuple is returned when a tuple is nil.
	ErrEmptyTuple = errors.New("[xdb/core] tuple is nil")
	// ErrRequiredFieldMissing is returned when a required field is missing.
	ErrRequiredFieldMissing = errors.New("[xdb/core] required field missing")
	// ErrFieldSchemaNotFound is returned when a field schema is not found.
	ErrFieldSchemaNotFound = errors.New("[xdb/core] field schema not found")
)

// Schema defines the structure and validation rules for records.
// It provides metadata, field definitions, and record-level constraints.
type Schema struct {
	// Name is the schema name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Version tracks schema evolution (e.g., "1.0.0").
	Version string

	// Fields is a list of field schemas.
	// Use hierarchical paths for nested fields (e.g., "profile.email").
	Fields []*FieldSchema

	// Required is a list of field names that are required.
	Required []string
}

// FieldSchema defines the schema for a single field.
type FieldSchema struct {
	// Name is the field name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Type specifies the field's full type
	Type Type

	// Default specifies the default value when the field is missing.
	Default *Value
}

// ValidateRecord validates all tuples in a record against this schema.
// It checks that required fields are present and validates each tuple's value.
func (s *Schema) ValidateRecord(record *Record) error {
	if record == nil {
		return ErrEmptyRecord
	}

	for _, field := range s.Required {
		tuple := record.Get(field)
		if tuple == nil {
			return errors.Wrap(ErrRequiredFieldMissing,
				"record", record.ID().String(),
				"field", field,
			)
		}
	}

	for _, tuple := range record.Tuples() {
		if err := s.ValidateTuple(tuple); err != nil {
			return err
		}
	}

	return nil
}

// GetFieldSchema returns the field schema for the given path.
func (s *Schema) GetFieldSchema(path string) *FieldSchema {
	for _, field := range s.Fields {
		if field.Name == path {
			return field
		}
	}
	return nil
}

// ValidateTuple validates a single tuple against this schema.
// It looks up the field schema by the tuple's attribute path and validates the value.
func (s *Schema) ValidateTuple(tuple *Tuple) error {
	if tuple == nil {
		return ErrEmptyTuple
	}

	fieldSchema := s.GetFieldSchema(tuple.Attr().String())
	if fieldSchema == nil {
		return errors.Wrap(ErrFieldSchemaNotFound, "field", tuple.Attr().String())
	}

	return fieldSchema.ValidateValue(tuple.Value())
}

// ValidateValue validates a value against this field schema.
func (f *FieldSchema) ValidateValue(value *Value) error {
	if !value.Type().Equals(f.Type) {
		return errors.Wrap(ErrTypeMismatch,
			"expected", f.Type.String(),
			"got", value.Type().String())
	}

	return nil
}
