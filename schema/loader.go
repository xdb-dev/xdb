package schema

import (
	"encoding/json"

	"github.com/gojekfarm/xtools/errors"
	"gopkg.in/yaml.v3"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrInvalidSchema is returned when schema format is invalid.
	ErrInvalidSchema = errors.New("[xdb/schema] invalid schema format")
	// ErrInvalidType is returned when a type definition is invalid.
	ErrInvalidType = errors.New("[xdb/schema] invalid type definition")
)

// LoadFromJSON loads a schema from JSON data.
func LoadFromJSON(data []byte) (*core.SchemaDef, error) {
	var raw rawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.Wrap(ErrInvalidSchema, "error", err.Error())
	}
	return convert(&raw)
}

// LoadFromYAML loads a schema from YAML data.
func LoadFromYAML(data []byte) (*core.SchemaDef, error) {
	var raw rawSchema
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, errors.Wrap(ErrInvalidSchema, "error", err.Error())
	}
	return convert(&raw)
}

// rawSchema represents the intermediate schema format for unmarshaling.
type rawSchema struct {
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string     `json:"version,omitempty" yaml:"version,omitempty"`
	Mode        string     `json:"mode,omitempty" yaml:"mode,omitempty"`
	Fields      []rawField `json:"fields" yaml:"fields"`
	Required    []string   `json:"required,omitempty" yaml:"required,omitempty"`
}

// rawField represents the intermediate field format for unmarshaling.
type rawField struct {
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string      `json:"type" yaml:"type"`
	ArrayOf     string      `json:"array_of,omitempty" yaml:"array_of,omitempty"`
	MapKey      string      `json:"map_key,omitempty" yaml:"map_key,omitempty"`
	MapValue    string      `json:"map_value,omitempty" yaml:"map_value,omitempty"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
}

// convert converts a rawSchema to a core.Schema.
func convert(raw *rawSchema) (*core.SchemaDef, error) {
	if raw.Name == "" {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "schema name is required")
	}

	schema := &core.SchemaDef{
		Name:        raw.Name,
		Description: raw.Description,
		Version:     raw.Version,
		Required:    raw.Required,
		Fields:      make([]*core.FieldDef, 0, len(raw.Fields)),
	}

	for _, rf := range raw.Fields {
		field, err := convertField(&rf)
		if err != nil {
			return nil, errors.Wrap(err, "field_name", rf.Name)
		}
		schema.Fields = append(schema.Fields, field)
	}

	return schema, nil
}

// convertField converts a rawField to a core.FieldSchema.
func convertField(rf *rawField) (*core.FieldDef, error) {
	if rf.Name == "" {
		return nil, errors.Wrap(ErrInvalidSchema, "reason", "field name is required")
	}

	typ, err := parseType(rf)
	if err != nil {
		return nil, err
	}

	field := &core.FieldDef{
		Name:        rf.Name,
		Description: rf.Description,
		Type:        typ,
	}

	if rf.Default != nil {
		field.Default = core.NewValue(rf.Default)
	}

	return field, nil
}

// parseType parses a type definition from a rawField.
func parseType(rf *rawField) (core.Type, error) {
	if rf.Type == "" {
		return core.TypeUnknown, errors.Wrap(ErrInvalidType, "reason", "type is required")
	}

	// Handle array types
	if rf.Type == "ARRAY" {
		if rf.ArrayOf == "" {
			return core.TypeUnknown, errors.Wrap(ErrInvalidType,
				"reason", "array_of is required for ARRAY type")
		}

		elemTypeID, err := core.ParseType(rf.ArrayOf)
		if err != nil {
			return core.TypeUnknown, errors.Wrap(err, "reason", "invalid array element type")
		}

		return core.NewArrayType(elemTypeID), nil
	}

	// Handle map types
	if rf.Type == "MAP" {
		if rf.MapKey == "" || rf.MapValue == "" {
			return core.TypeUnknown, errors.Wrap(ErrInvalidType,
				"reason", "map_key and map_value are required for MAP type")
		}

		keyTypeID, err := core.ParseType(rf.MapKey)
		if err != nil {
			return core.TypeUnknown, errors.Wrap(err, "reason", "invalid map key type")
		}

		valueTypeID, err := core.ParseType(rf.MapValue)
		if err != nil {
			return core.TypeUnknown, errors.Wrap(err, "reason", "invalid map value type")
		}

		return core.NewMapType(keyTypeID, valueTypeID), nil
	}

	// Handle scalar types
	typeID, err := core.ParseType(rf.Type)
	if err != nil {
		return core.TypeUnknown, errors.Wrap(err, "reason", "invalid type")
	}

	switch typeID {
	case core.TypeIDBoolean:
		return core.TypeBool, nil
	case core.TypeIDInteger:
		return core.TypeInt, nil
	case core.TypeIDUnsigned:
		return core.TypeUnsigned, nil
	case core.TypeIDFloat:
		return core.TypeFloat, nil
	case core.TypeIDString:
		return core.TypeString, nil
	case core.TypeIDBytes:
		return core.TypeBytes, nil
	case core.TypeIDTime:
		return core.TypeTime, nil
	default:
		return core.TypeUnknown, errors.Wrap(ErrInvalidType, "type", rf.Type)
	}
}
