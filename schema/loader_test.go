package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

func TestLoader_LoadFromJSON_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected *schema.Def
	}{
		{
			name: "Scalar Types",
			input: `{
				"name": "User",
				"description": "User schema",
				"version": "1.0.0",
				"fields": [
					{"name": "name", "type": "STRING"},
					{"name": "age", "type": "INTEGER"},
					{"name": "active", "type": "BOOLEAN"}
				],
				"required": ["name"]
			}`,
			expected: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
					{Name: "active", Type: core.TypeBool},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "Array Types",
			input: `{
				"name": "Post",
				"fields": [
					{"name": "tags", "type": "ARRAY", "array_of": "STRING"},
					{"name": "scores", "type": "ARRAY", "array_of": "INTEGER"}
				]
			}`,
			expected: &schema.Def{
				Name: "Post",
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
					{Name: "scores", Type: core.NewArrayType(core.TIDInteger)},
				},
			},
		},
		{
			name: "Map Types",
			input: `{
				"name": "Config",
				"fields": [
					{"name": "settings", "type": "MAP", "map_key": "STRING", "map_value": "STRING"},
					{"name": "counts", "type": "MAP", "map_key": "STRING", "map_value": "INTEGER"}
				]
			}`,
			expected: &schema.Def{
				Name: "Config",
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
					{Name: "counts", Type: core.NewMapType(core.TIDString, core.TIDInteger)},
				},
			},
		},
		{
			name: "Nested Fields",
			input: `{
				"name": "User",
				"fields": [
					{"name": "name", "type": "STRING"},
					{"name": "profile.bio", "type": "STRING"},
					{"name": "settings.notifications.email", "type": "BOOLEAN"}
				]
			}`,
			expected: &schema.Def{
				Name: "User",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "profile.bio", Type: core.TypeString},
					{Name: "settings.notifications.email", Type: core.TypeBool},
				},
			},
		},
		{
			name: "All Scalar Types",
			input: `{
				"name": "Complete",
				"fields": [
					{"name": "bool_field", "type": "BOOLEAN"},
					{"name": "int_field", "type": "INTEGER"},
					{"name": "unsigned_field", "type": "UNSIGNED"},
					{"name": "float_field", "type": "FLOAT"},
					{"name": "string_field", "type": "STRING"},
					{"name": "bytes_field", "type": "BYTES"},
					{"name": "time_field", "type": "TIME"}
				]
			}`,
			expected: &schema.Def{
				Name: "Complete",
				Fields: []*schema.FieldDef{
					{Name: "bool_field", Type: core.TypeBool},
					{Name: "int_field", Type: core.TypeInt},
					{Name: "unsigned_field", Type: core.TypeUnsigned},
					{Name: "float_field", Type: core.TypeFloat},
					{Name: "string_field", Type: core.TypeString},
					{Name: "bytes_field", Type: core.TypeBytes},
					{Name: "time_field", Type: core.TypeTime},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.LoadFromJSON([]byte(tt.input))
			assert.NoError(t, err)
			tests.AssertDefEqual(t, tt.expected, s)
		})
	}
}

func TestLoader_LoadFromJSON_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name:        "Invalid JSON",
			input:       `{invalid json}`,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name:        "Missing Schema Name",
			input:       `{"fields": [{"name": "field1", "type": "STRING"}]}`,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name:        "Missing Field Name",
			input:       `{"name": "Test", "fields": [{"type": "STRING"}]}`,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name:        "Missing Field Type",
			input:       `{"name": "Test", "fields": [{"name": "field1"}]}`,
			expectedErr: schema.ErrInvalidType,
		},
		{
			name:        "Invalid Type Name",
			input:       `{"name": "Test", "fields": [{"name": "field1", "type": "INVALID_TYPE"}]}`,
			expectedErr: core.ErrUnknownType,
		},
		{
			name:        "Array Without array_of",
			input:       `{"name": "Test", "fields": [{"name": "tags", "type": "ARRAY"}]}`,
			expectedErr: schema.ErrInvalidType,
		},
		{
			name:        "Map Without map_key",
			input:       `{"name": "Test", "fields": [{"name": "settings", "type": "MAP", "map_value": "STRING"}]}`,
			expectedErr: schema.ErrInvalidType,
		},
		{
			name:        "Map Without map_value",
			input:       `{"name": "Test", "fields": [{"name": "settings", "type": "MAP", "map_key": "STRING"}]}`,
			expectedErr: schema.ErrInvalidType,
		},
		{
			name:        "Invalid Array Element Type",
			input:       `{"name": "Test", "fields": [{"name": "tags", "type": "ARRAY", "array_of": "INVALID"}]}`,
			expectedErr: core.ErrUnknownType,
		},
		{
			name:        "Invalid Map Key Type",
			input:       `{"name": "Test", "fields": [{"name": "settings", "type": "MAP", "map_key": "INVALID", "map_value": "STRING"}]}`,
			expectedErr: core.ErrUnknownType,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.LoadFromJSON([]byte(tt.input))
			assert.Error(t, err)
			assert.Nil(t, s)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestLoader_LoadFromYAML_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected *schema.Def
	}{
		{
			name: "Scalar Types",
			input: `
name: User
description: User schema
version: 1.0.0
fields:
  - name: name
    type: STRING
  - name: age
    type: INTEGER
required:
  - name
`,
			expected: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "Array Types",
			input: `
name: Post
fields:
  - name: tags
    type: ARRAY
    array_of: STRING
`,
			expected: &schema.Def{
				Name: "Post",
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
				},
			},
		},
		{
			name: "Map Types",
			input: `
name: Config
fields:
  - name: settings
    type: MAP
    map_key: STRING
    map_value: STRING
`,
			expected: &schema.Def{
				Name: "Config",
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
				},
			},
		},
		{
			name: "Nested Fields",
			input: `
name: User
fields:
  - name: name
    type: STRING
  - name: profile.bio
    type: STRING
  - name: settings.notifications.email
    type: BOOLEAN
`,
			expected: &schema.Def{
				Name: "User",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "profile.bio", Type: core.TypeString},
					{Name: "settings.notifications.email", Type: core.TypeBool},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.LoadFromYAML([]byte(tt.input))
			assert.NoError(t, err)
			tests.AssertDefEqual(t, tt.expected, s)
		})
	}
}

func TestLoader_LoadFromYAML_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{
			name: "Invalid YAML",
			input: `
name: Test
fields:
  - name: field1
    type: STRING
  invalid yaml here
`,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name: "Missing Schema Name",
			input: `
fields:
  - name: field1
    type: STRING
`,
			expectedErr: schema.ErrInvalidSchema,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.LoadFromYAML([]byte(tt.input))
			assert.Error(t, err)
			assert.Nil(t, s)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
