package schema

import (
	"encoding/json"

	"github.com/gojekfarm/xtools/errors"
	"gopkg.in/yaml.v3"

	"github.com/xdb-dev/xdb/core"
)

// WriteToJSON writes a schema to JSON format.
func WriteToJSON(schema *core.SchemaDef) ([]byte, error) {
	raw, err := convertToRaw(schema)
	if err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

// WriteToYAML writes a schema to YAML format.
func WriteToYAML(schema *core.SchemaDef) ([]byte, error) {
	raw, err := convertToRaw(schema)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(raw)
}

// convertToRaw converts a core.Schema to a rawSchema.
func convertToRaw(schema *core.SchemaDef) (*rawSchema, error) {
	if schema == nil {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "schema is nil")
	}

	if schema.Name == "" {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "schema name is required")
	}

	raw := &rawSchema{
		Name:        schema.Name,
		Description: schema.Description,
		Version:     schema.Version,
		Required:    schema.Required,
		Fields:      make([]rawField, 0, len(schema.Fields)),
	}

	for _, field := range schema.Fields {
		rf, err := convertFieldToRaw(field)
		if err != nil {
			return nil, errors.Wrap(err, "field_name", field.Name)
		}
		raw.Fields = append(raw.Fields, *rf)
	}

	return raw, nil
}

// convertFieldToRaw converts a core.FieldSchema to a rawField.
func convertFieldToRaw(field *core.FieldDef) (*rawField, error) {
	if field == nil {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "field is nil")
	}

	if field.Name == "" {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "field name is required")
	}

	rf := &rawField{
		Name:        field.Name,
		Description: field.Description,
		Type:        field.Type.String(),
	}

	if field.Type.ID() == core.TypeIDArray {
		rf.ArrayOf = field.Type.ValueType().String()
	} else if field.Type.ID() == core.TypeIDMap {
		rf.MapKey = field.Type.KeyType().String()
		rf.MapValue = field.Type.ValueType().String()
	}

	if field.Default != nil {
		rf.Default = field.Default.Unwrap()
	}

	return rf, nil
}
