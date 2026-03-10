package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestDef_MarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		def      *schema.Def
		expected string
	}{
		{
			name: "with fields",
			def: &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/posts"),
				Mode: schema.ModeStrict,
				Fields: map[string]schema.FieldDef{
					"title": {Type: core.TIDString, Required: true},
				},
			},
			expected: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict",
				"fields": {
					"title": {"type": "STRING", "required": true}
				}
			}`,
		},
		{
			name: "flexible mode no fields",
			def: &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/users"),
				Mode: schema.ModeFlexible,
			},
			expected: `{
				"uri": "xdb://com.example/users",
				"mode": "flexible"
			}`,
		},
		{
			name: "multiple fields",
			def: &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/posts"),
				Mode: schema.ModeStrict,
				Fields: map[string]schema.FieldDef{
					"title":  {Type: core.TIDString, Required: true},
					"rating": {Type: core.TIDFloat},
					"active": {Type: core.TIDBoolean},
				},
			},
			expected: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict",
				"fields": {
					"title":  {"type": "STRING", "required": true},
					"rating": {"type": "FLOAT"},
					"active": {"type": "BOOLEAN"}
				}
			}`,
		},
		{
			name: "dynamic mode",
			def: &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/logs"),
				Mode: schema.ModeDynamic,
			},
			expected: `{
				"uri": "xdb://com.example/logs",
				"mode": "dynamic"
			}`,
		},
		{
			name: "zero-value mode defaults to strict",
			def: &schema.Def{
				URI: core.MustParseURI("xdb://com.example/posts"),
			},
			expected: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict"
			}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.def)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestDef_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		input        string
		expectedURI  string
		expectedMode schema.Mode
		expectedLen  int
	}{
		{
			name: "with fields",
			input: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict",
				"fields": {
					"title": {"type": "STRING", "required": true}
				}
			}`,
			expectedURI:  "xdb://com.example/posts",
			expectedMode: schema.ModeStrict,
			expectedLen:  1,
		},
		{
			name: "flexible mode no fields",
			input: `{
				"uri": "xdb://com.example/users",
				"mode": "flexible"
			}`,
			expectedURI:  "xdb://com.example/users",
			expectedMode: schema.ModeFlexible,
			expectedLen:  0,
		},
		{
			name: "dynamic mode",
			input: `{
				"uri": "xdb://com.example/logs",
				"mode": "dynamic"
			}`,
			expectedURI:  "xdb://com.example/logs",
			expectedMode: schema.ModeDynamic,
			expectedLen:  0,
		},
		{
			name: "missing mode defaults to strict",
			input: `{
				"uri": "xdb://com.example/posts"
			}`,
			expectedURI:  "xdb://com.example/posts",
			expectedMode: schema.ModeStrict,
			expectedLen:  0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var def schema.Def
			err := json.Unmarshal([]byte(tt.input), &def)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURI, def.URI.String())
			assert.Equal(t, tt.expectedMode, def.Mode)
			assert.Len(t, def.Fields, tt.expectedLen)
		})
	}
}

func TestDef_UnmarshalJSON_FieldDetails(t *testing.T) {
	t.Parallel()

	input := `{
		"uri": "xdb://com.example/posts",
		"mode": "strict",
		"fields": {
			"title":  {"type": "STRING", "required": true},
			"rating": {"type": "FLOAT"},
			"active": {"type": "BOOLEAN"}
		}
	}`

	var def schema.Def
	err := json.Unmarshal([]byte(input), &def)
	require.NoError(t, err)

	title, ok := def.Fields["title"]
	require.True(t, ok)
	assert.Equal(t, core.TIDString, title.Type)
	assert.True(t, title.Required)

	rating, ok := def.Fields["rating"]
	require.True(t, ok)
	assert.Equal(t, core.TIDFloat, rating.Type)
	assert.False(t, rating.Required)

	active, ok := def.Fields["active"]
	require.True(t, ok)
	assert.Equal(t, core.TIDBoolean, active.Type)
	assert.False(t, active.Required)
}

func TestDef_JSON_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/posts"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"title":   {Type: core.TIDString, Required: true},
			"content": {Type: core.TIDString},
			"rating":  {Type: core.TIDFloat},
			"active":  {Type: core.TIDBoolean},
			"count":   {Type: core.TIDInteger},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded schema.Def
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.URI.String(), decoded.URI.String())
	assert.Equal(t, original.Mode, decoded.Mode)
	require.Len(t, decoded.Fields, len(original.Fields))

	for name, expectedField := range original.Fields {
		actualField, ok := decoded.Fields[name]
		require.True(t, ok, "field %s not found", name)
		assert.Equal(t, expectedField.Type, actualField.Type)
		assert.Equal(t, expectedField.Required, actualField.Required)
	}
}

func TestDef_UnmarshalJSON_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"invalid json", `{not json}`},
		{"invalid mode", `{"uri": "xdb://com.example/posts", "mode": "bad"}`},
		{"invalid uri", `{"uri": "not-a-uri", "mode": "flexible"}`},
		{"invalid field type", `{"uri": "xdb://com.example/posts", "mode": "strict", "fields": {"f": {"type": "NOPE"}}}`},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var def schema.Def
			err := json.Unmarshal([]byte(tt.input), &def)
			assert.Error(t, err)
		})
	}
}
