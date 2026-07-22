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
				Fields: map[string]schema.Field{
					"title": {Type: core.TypeString, Required: true},
				},
			},
			expected: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict",
				"fields": {
					"title": {"type": "string", "required": true}
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
				Fields: map[string]schema.Field{
					"title":  {Type: core.TypeString, Required: true},
					"rating": {Type: core.TypeFloat},
					"active": {Type: core.TypeBool},
				},
			},
			expected: `{
				"uri": "xdb://com.example/posts",
				"mode": "strict",
				"fields": {
					"title":  {"type": "string", "required": true},
					"rating": {"type": "float"},
					"active": {"type": "boolean"}
				}
			}`,
		},
		{
			name: "description, annotations and revision",
			def: &schema.Def{
				URI:         core.MustParseURI("xdb://com.example/users"),
				Mode:        schema.ModeStrict,
				Description: "a user",
				Revision:    7,
				Annotations: map[string]string{"source": "proto"},
				Fields: map[string]schema.Field{
					"name": {
						Type:        core.TypeString,
						Description: "the name",
						Annotations: map[string]string{"proto.number": "3"},
					},
				},
			},
			expected: `{
				"uri": "xdb://com.example/users",
				"mode": "strict",
				"description": "a user",
				"revision": 7,
				"annotations": {"source": "proto"},
				"fields": {
					"name": {
						"type": "string",
						"description": "the name",
						"annotations": {"proto.number": "3"}
					}
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
					"title": {"type": "string", "required": true}
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
			"title":  {"type": "string", "required": true, "description": "the title"},
			"rating": {"type": "float"},
			"active": {"type": "boolean"}
		}
	}`

	var def schema.Def
	err := json.Unmarshal([]byte(input), &def)
	require.NoError(t, err)

	title, ok := def.Fields["title"]
	require.True(t, ok)
	assert.Equal(t, core.TIDString, title.Type.ID())
	assert.True(t, title.Required)
	assert.Equal(t, "the title", title.Description)

	rating, ok := def.Fields["rating"]
	require.True(t, ok)
	assert.Equal(t, core.TIDFloat, rating.Type.ID())
	assert.False(t, rating.Required)

	active, ok := def.Fields["active"]
	require.True(t, ok)
	assert.Equal(t, core.TIDBoolean, active.Type.ID())
	assert.False(t, active.Required)
}

func TestDef_JSON_RoundTrip(t *testing.T) {
	t.Parallel()

	original := &schema.Def{
		URI:         core.MustParseURI("xdb://com.example/posts"),
		Mode:        schema.ModeStrict,
		Description: "a post",
		Revision:    3,
		Annotations: map[string]string{"source": "go"},
		Fields: map[string]schema.Field{
			"title":   {Type: core.TypeString, Required: true, Description: "d"},
			"content": {Type: core.TypeString},
			"rating":  {Type: core.TypeFloat},
			"active":  {Type: core.TypeBool},
			"count":   {Type: core.TypeInt, Annotations: map[string]string{"go.type": "int32"}},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded schema.Def
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.URI.String(), decoded.URI.String())
	assert.Equal(t, original.Mode, decoded.Mode)
	assert.Equal(t, original.Description, decoded.Description)
	assert.Equal(t, original.Revision, decoded.Revision)
	assert.Equal(t, original.Annotations, decoded.Annotations)
	require.Len(t, decoded.Fields, len(original.Fields))

	for name, expectedField := range original.Fields {
		actualField, ok := decoded.Fields[name]
		require.True(t, ok, "field %s not found", name)
		assert.Equal(t, expectedField.Type, actualField.Type)
		assert.Equal(t, expectedField.Required, actualField.Required)
		assert.Equal(t, expectedField.Description, actualField.Description)
		assert.Equal(t, expectedField.Annotations, actualField.Annotations)
	}
}

func TestDef_JSON_Array(t *testing.T) {
	t.Parallel()

	t.Run("marshal emits elem_type for typed arrays", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/posts"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
			},
		}

		data, err := json.Marshal(def)
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"uri": "xdb://com.example/posts",
			"mode": "strict",
			"fields": {
				"tags": {"type": "array", "elem_type": "string"}
			}
		}`, string(data))
	})

	t.Run("marshal omits elem_type when unset", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/posts"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType("")},
			},
		}

		data, err := json.Marshal(def)
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"uri": "xdb://com.example/posts",
			"mode": "strict",
			"fields": {
				"tags": {"type": "array"}
			}
		}`, string(data))
	})

	// Backward compatibility: schemas stored as {type, elem_type} still parse.
	t.Run("unmarshal parses legacy type/elem_type", func(t *testing.T) {
		input := `{
			"uri": "xdb://com.example/posts",
			"mode": "strict",
			"fields": {
				"tags": {"type": "array", "elem_type": "string"}
			}
		}`

		var def schema.Def
		err := json.Unmarshal([]byte(input), &def)
		require.NoError(t, err)

		tags, ok := def.Fields["tags"]
		require.True(t, ok)
		assert.Equal(t, core.TIDArray, tags.Type.ID())
		assert.Equal(t, core.TIDString, tags.Type.ElemTypeID())
	})

	// Unmarshal is permissive; well-formedness is enforced by Def.Validate
	// at the store boundary.
	t.Run("unmarshal accepts array without elem_type", func(t *testing.T) {
		input := `{
			"uri": "xdb://com.example/posts",
			"mode": "strict",
			"fields": {
				"tags": {"type": "array"}
			}
		}`

		var def schema.Def
		err := json.Unmarshal([]byte(input), &def)
		require.NoError(t, err)

		tags, ok := def.Fields["tags"]
		require.True(t, ok)
		assert.Equal(t, core.TIDArray, tags.Type.ID())
		assert.Equal(t, core.TID(""), tags.Type.ElemTypeID())
	})

	t.Run("unmarshal rejects invalid elem_type", func(t *testing.T) {
		input := `{
			"uri": "xdb://com.example/posts",
			"mode": "strict",
			"fields": {
				"tags": {"type": "array", "elem_type": "NOPE"}
			}
		}`

		var def schema.Def
		err := json.Unmarshal([]byte(input), &def)
		assert.Error(t, err)
	})

	t.Run("roundtrip preserves elem_type", func(t *testing.T) {
		original := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/posts"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"tags":   {Type: core.NewArrayType(core.TIDString)},
				"counts": {Type: core.NewArrayType(core.TIDInteger)},
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded schema.Def
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, core.TIDString, decoded.Fields["tags"].Type.ElemTypeID())
		assert.Equal(t, core.TIDInteger, decoded.Fields["counts"].Type.ElemTypeID())
	})
}

func TestDef_JSON_ObjectArrayItems(t *testing.T) {
	t.Parallel()

	t.Run("marshal emits nested items", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/orders"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"sku": {Type: core.TypeString, Required: true},
						"qty": {Type: core.TypeInt},
					},
				},
			},
		}

		data, err := json.Marshal(def)
		require.NoError(t, err)
		assert.JSONEq(t, `{
			"uri": "xdb://com.example/orders",
			"mode": "strict",
			"fields": {
				"lines": {
					"type": "array",
					"elem_type": "json",
					"items": {
						"sku": {"type": "string", "required": true},
						"qty": {"type": "integer"}
					}
				}
			}
		}`, string(data))
	})

	t.Run("roundtrip preserves items", func(t *testing.T) {
		original := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/orders"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"sku":    {Type: core.TypeString, Required: true},
						"placed": {Type: core.TypeTime},
						"tags": {
							Type:  core.NewArrayType(core.TIDJSON),
							Items: map[string]schema.Field{"name": {Type: core.TypeString}},
						},
					},
				},
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded schema.Def
		require.NoError(t, json.Unmarshal(data, &decoded))

		lines := decoded.Fields["lines"]
		require.NotNil(t, lines.Items)
		assert.Equal(t, core.TIDString, lines.Items["sku"].Type.ID())
		assert.True(t, lines.Items["sku"].Required)
		assert.Equal(t, core.TIDTime, lines.Items["placed"].Type.ID())

		tags := lines.Items["tags"]
		assert.Equal(t, core.TIDJSON, tags.Type.ElemTypeID())
		assert.Equal(t, core.TIDString, tags.Items["name"].Type.ID())
	})
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

func TestDef_UnmarshalJSON_InvalidModeListsValidModes(t *testing.T) {
	t.Parallel()

	var def schema.Def
	err := json.Unmarshal([]byte(`{"uri": "xdb://com.example/posts", "mode": "bad"}`), &def)

	require.ErrorIs(t, err, schema.ErrInvalidMode)
	assert.Contains(t, err.Error(), "flexible")
	assert.Contains(t, err.Error(), "strict")
	assert.Contains(t, err.Error(), "dynamic")
}

func TestDef_CloneWithFields(t *testing.T) {
	uri, err := core.ParseURI("xdb://test/Post")
	require.NoError(t, err)

	def := &schema.Def{
		URI:         uri,
		Description: "posts",
		Mode:        schema.ModeDynamic,
		Revision:    3,
		Annotations: map[string]string{"source": "test"},
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString, Required: true},
		},
	}

	evolved := def.CloneWithFields(map[string]schema.Field{
		"views": {Type: core.TypeInt},
	})

	// Metadata is preserved and the revision is bumped.
	assert.Equal(t, def.URI, evolved.URI)
	assert.Equal(t, def.Description, evolved.Description)
	assert.Equal(t, def.Mode, evolved.Mode)
	assert.Equal(t, def.Annotations, evolved.Annotations)
	assert.Equal(t, int64(4), evolved.Revision)

	// Existing and new fields are merged.
	assert.Equal(t, map[string]schema.Field{
		"title": {Type: core.TypeString, Required: true},
		"views": {Type: core.TypeInt},
	}, evolved.Fields)

	// The original def and its field map are untouched.
	assert.Equal(t, int64(3), def.Revision)
	assert.Equal(t, map[string]schema.Field{
		"title": {Type: core.TypeString, Required: true},
	}, def.Fields)
}
