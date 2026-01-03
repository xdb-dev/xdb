package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

func TestWriter_WriteToJSON_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		schema   *schema.Def
		expected string
	}{
		{
			name: "Scalar Types",
			schema: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
					{Name: "active", Type: core.TypeBool},
				},
			},
			expected: `{"name":"User","description":"User schema","version":"1.0.0","fields":[{"name":"name","type":"STRING"},{"name":"age","type":"INTEGER"},{"name":"active","type":"BOOLEAN"}]}`,
		},
		{
			name: "Array Types",
			schema: &schema.Def{
				Name: "Post",
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
					{Name: "scores", Type: core.NewArrayType(core.TIDInteger)},
				},
			},
			expected: `{"name":"Post","fields":[{"name":"tags","type":"ARRAY","array_of":"STRING"},{"name":"scores","type":"ARRAY","array_of":"INTEGER"}]}`,
		},
		{
			name: "Map Types",
			schema: &schema.Def{
				Name: "Config",
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
					{Name: "counts", Type: core.NewMapType(core.TIDString, core.TIDInteger)},
				},
			},
			expected: `{"name":"Config","fields":[{"name":"settings","type":"MAP","map_key":"STRING","map_value":"STRING"},{"name":"counts","type":"MAP","map_key":"STRING","map_value":"INTEGER"}]}`,
		},
		{
			name: "All Scalar Types",
			schema: &schema.Def{
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
			expected: `{"name":"Complete","fields":[{"name":"bool_field","type":"BOOLEAN"},{"name":"int_field","type":"INTEGER"},{"name":"unsigned_field","type":"UNSIGNED"},{"name":"float_field","type":"FLOAT"},{"name":"string_field","type":"STRING"},{"name":"bytes_field","type":"BYTES"},{"name":"time_field","type":"TIME"}]}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := schema.WriteToJSON(tt.schema)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestWriter_WriteToJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema *schema.Def
	}{
		{
			name: "Scalar Types",
			schema: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Mode:        schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
					{Name: "active", Type: core.TypeBool},
				},
			},
		},
		{
			name: "Array Types",
			schema: &schema.Def{
				Name: "Post",
				Mode: schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
					{Name: "scores", Type: core.NewArrayType(core.TIDInteger)},
				},
			},
		},
		{
			name: "Map Types",
			schema: &schema.Def{
				Name: "Config",
				Mode: schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
					{Name: "counts", Type: core.NewMapType(core.TIDString, core.TIDInteger)},
				},
			},
		},
		{
			name: "Nested Fields",
			schema: &schema.Def{
				Name: "User",
				Mode: schema.ModeStrict,
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
			data, err := schema.WriteToJSON(tt.schema)
			assert.NoError(t, err)

			loaded, err := schema.LoadFromJSON(data)
			assert.NoError(t, err)
			tests.AssertDefEqual(t, tt.schema, loaded)
		})
	}
}

func TestWriter_WriteToJSON_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		schema      *schema.Def
		expectedErr error
	}{
		{
			name:        "Nil Schema",
			schema:      nil,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name: "Empty Schema Name",
			schema: &schema.Def{
				Name: "",
				Fields: []*schema.FieldDef{
					{Name: "field1", Type: core.TypeString},
				},
			},
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name: "Nil Field",
			schema: &schema.Def{
				Name: "Test",
				Fields: []*schema.FieldDef{
					nil,
				},
			},
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name: "Empty Field Name",
			schema: &schema.Def{
				Name: "Test",
				Fields: []*schema.FieldDef{
					{Name: "", Type: core.TypeString},
				},
			},
			expectedErr: schema.ErrInvalidSchema,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := schema.WriteToJSON(tt.schema)
			assert.Error(t, err)
			assert.Nil(t, data)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestWriter_WriteToYAML_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		schema   *schema.Def
		contains []string
	}{
		{
			name: "Scalar Types",
			schema: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
				},
			},
			contains: []string{"name: User", "description: User schema", "version: 1.0.0", "type: STRING", "type: INTEGER"},
		},
		{
			name: "Array Types",
			schema: &schema.Def{
				Name: "Post",
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
				},
			},
			contains: []string{"name: Post", "type: ARRAY", "array_of: STRING"},
		},
		{
			name: "Map Types",
			schema: &schema.Def{
				Name: "Config",
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
				},
			},
			contains: []string{"name: Config", "type: MAP", "map_key: STRING", "map_value: STRING"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := schema.WriteToYAML(tt.schema)
			assert.NoError(t, err)
			dataStr := string(data)
			for _, substr := range tt.contains {
				assert.Contains(t, dataStr, substr)
			}
		})
	}
}

func TestWriter_WriteToYAML_RoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		schema *schema.Def
	}{
		{
			name: "Scalar Types",
			schema: &schema.Def{
				Name:        "User",
				Description: "User schema",
				Version:     "1.0.0",
				Mode:        schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
					{Name: "age", Type: core.TypeInt},
				},
			},
		},
		{
			name: "Array Types",
			schema: &schema.Def{
				Name: "Post",
				Mode: schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "tags", Type: core.NewArrayType(core.TIDString)},
					{Name: "scores", Type: core.NewArrayType(core.TIDInteger)},
				},
			},
		},
		{
			name: "Map Types",
			schema: &schema.Def{
				Name: "Config",
				Mode: schema.ModeStrict,
				Fields: []*schema.FieldDef{
					{Name: "settings", Type: core.NewMapType(core.TIDString, core.TIDString)},
					{Name: "counts", Type: core.NewMapType(core.TIDString, core.TIDInteger)},
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := schema.WriteToYAML(tt.schema)
			assert.NoError(t, err)

			loaded, err := schema.LoadFromYAML(data)
			assert.NoError(t, err)
			tests.AssertDefEqual(t, tt.schema, loaded)
		})
	}
}

func TestWriter_WriteToYAML_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		schema      *schema.Def
		expectedErr error
	}{
		{
			name:        "Nil Schema",
			schema:      nil,
			expectedErr: schema.ErrInvalidSchema,
		},
		{
			name: "Empty Schema Name",
			schema: &schema.Def{
				Name: "",
				Fields: []*schema.FieldDef{
					{Name: "field1", Type: core.TypeString},
				},
			},
			expectedErr: schema.ErrInvalidSchema,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := schema.WriteToYAML(tt.schema)
			assert.Error(t, err)
			assert.Nil(t, data)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
