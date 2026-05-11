package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestValidateTuples_Strict_Valid(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
			"age":  {Type: core.TIDInteger},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "name", "Alice"),
		core.NewTuple("com.example/users/1", "age", int64(30)),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.NoError(t, err)
}

func TestValidateTuples_Strict_UnknownField(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "unknown", "value"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.ErrorIs(t, err, schema.ErrUnknownField)
}

func TestValidateTuples_Strict_TypeMismatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"age": {Type: core.TIDInteger},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "age", "not an int"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.ErrorIs(t, err, schema.ErrTypeMismatch)
}

func TestValidateTuples_Dynamic_Valid(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "name", "Alice"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.NoError(t, err)
}

func TestValidateTuples_Dynamic_UnknownField(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "unknown", "value"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.ErrorIs(t, err, schema.ErrUnknownField)
}

func TestValidateTuples_Dynamic_TypeMismatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.FieldDef{
			"age": {Type: core.TIDInteger},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "age", "not an int"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.ErrorIs(t, err, schema.ErrTypeMismatch)
}

func TestValidateTuples_Flexible_AllowsUnknownFields(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeFlexible,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "unknown", "value"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.NoError(t, err)
}

func TestValidateTuples_Flexible_NilFields(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeFlexible,
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "anything", "value"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.NoError(t, err)
}

func TestValidateTuples_EmptyTuples(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
		},
	}

	err := schema.ValidateTuples(def, nil)
	assert.NoError(t, err)
}

func TestValidateRecords(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"name": {Type: core.TIDString},
			"age":  {Type: core.TIDInteger},
		},
	}

	t.Run("valid records", func(t *testing.T) {
		r := core.NewRecord("com.example", "users", "1").
			Set("name", "Alice").
			Set("age", int64(30))

		err := schema.ValidateRecords(def, []*core.Record{r})
		assert.NoError(t, err)
	})

	t.Run("unknown field in record", func(t *testing.T) {
		r := core.NewRecord("com.example", "users", "1").
			Set("name", "Alice").
			Set("extra", "bad")

		err := schema.ValidateRecords(def, []*core.Record{r})
		assert.ErrorIs(t, err, schema.ErrUnknownField)
	})

	t.Run("multiple records", func(t *testing.T) {
		r1 := core.NewRecord("com.example", "users", "1").
			Set("name", "Alice")
		r2 := core.NewRecord("com.example", "users", "2").
			Set("name", "Bob").
			Set("age", int64(25))

		err := schema.ValidateRecords(def, []*core.Record{r1, r2})
		assert.NoError(t, err)
	})

	t.Run("type mismatch in second record", func(t *testing.T) {
		r1 := core.NewRecord("com.example", "users", "1").
			Set("name", "Alice")
		r2 := core.NewRecord("com.example", "users", "2").
			Set("age", "not a number")

		err := schema.ValidateRecords(def, []*core.Record{r1, r2})
		assert.ErrorIs(t, err, schema.ErrTypeMismatch)
	})
}

func TestValidateTuples_Array_ElemTypeMismatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/posts"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple(
			"com.example/posts/1",
			"tags",
			core.ArrayVal(core.TIDInteger, core.IntVal(1), core.IntVal(2)),
		),
	}

	err := schema.ValidateTuples(def, tuples)
	require.ErrorIs(t, err, schema.ErrTypeMismatch)
	assert.Contains(t, err.Error(), "ARRAY<STRING>")
	assert.Contains(t, err.Error(), "ARRAY<INTEGER>")
}

func TestValidateTuples_Array_ElemTypeMatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/posts"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple(
			"com.example/posts/1",
			"tags",
			core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b")),
		),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.NoError(t, err)
}

func TestInferField(t *testing.T) {
	t.Parallel()

	t.Run("scalar value", func(t *testing.T) {
		f := schema.InferField(core.StringVal("hello"))
		assert.Equal(t, core.TIDString, f.Type)
		assert.Equal(t, core.TID(""), f.ElemType)
	})

	t.Run("array value captures elem type", func(t *testing.T) {
		f := schema.InferField(core.ArrayVal(core.TIDString, core.StringVal("a")))
		assert.Equal(t, core.TIDArray, f.Type)
		assert.Equal(t, core.TIDString, f.ElemType)
	})

	t.Run("array value of integers", func(t *testing.T) {
		f := schema.InferField(core.ArrayVal(core.TIDInteger, core.IntVal(1)))
		assert.Equal(t, core.TIDArray, f.Type)
		assert.Equal(t, core.TIDInteger, f.ElemType)
	})
}

func TestEvolveDynamic(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no new fields", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/events"),
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.FieldDef{
				"name": {Type: core.TIDString},
			},
		}
		tuples := []*core.Tuple{
			core.NewTuple("com.example/events/1", "name", "click"),
		}

		newFields, err := schema.EvolveDynamic(def, tuples)
		require.NoError(t, err)
		assert.Nil(t, newFields)
	})

	t.Run("infers new scalar field", func(t *testing.T) {
		def := &schema.Def{
			URI:    core.MustParseURI("xdb://com.example/events"),
			Mode:   schema.ModeDynamic,
			Fields: map[string]schema.FieldDef{},
		}
		tuples := []*core.Tuple{
			core.NewTuple("com.example/events/1", "count", int64(5)),
		}

		newFields, err := schema.EvolveDynamic(def, tuples)
		require.NoError(t, err)
		require.Len(t, newFields, 1)
		assert.Equal(t, core.TIDInteger, newFields["count"].Type)
	})

	t.Run("infers new array field with elem type", func(t *testing.T) {
		def := &schema.Def{
			URI:    core.MustParseURI("xdb://com.example/events"),
			Mode:   schema.ModeDynamic,
			Fields: map[string]schema.FieldDef{},
		}
		tuples := []*core.Tuple{
			core.NewTuple(
				"com.example/events/1",
				"tags",
				core.ArrayVal(core.TIDString, core.StringVal("a")),
			),
		}

		newFields, err := schema.EvolveDynamic(def, tuples)
		require.NoError(t, err)
		require.Len(t, newFields, 1)
		assert.Equal(t, core.TIDArray, newFields["tags"].Type)
		assert.Equal(t, core.TIDString, newFields["tags"].ElemType)
	})

	t.Run("rejects mismatched elem type on known array field", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/events"),
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.FieldDef{
				"tags": {Type: core.TIDArray, ElemType: core.TIDString},
			},
		}
		tuples := []*core.Tuple{
			core.NewTuple(
				"com.example/events/1",
				"tags",
				core.ArrayVal(core.TIDInteger, core.IntVal(1)),
			),
		}

		_, err := schema.EvolveDynamic(def, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})

	t.Run("dedupes duplicate new field names", func(t *testing.T) {
		def := &schema.Def{
			URI:    core.MustParseURI("xdb://com.example/events"),
			Mode:   schema.ModeDynamic,
			Fields: map[string]schema.FieldDef{},
		}
		tuples := []*core.Tuple{
			core.NewTuple("com.example/events/1", "x", int64(1)),
			core.NewTuple("com.example/events/2", "x", int64(2)),
		}

		newFields, err := schema.EvolveDynamic(def, tuples)
		require.NoError(t, err)
		assert.Len(t, newFields, 1)
	})
}

func TestDef_Validate(t *testing.T) {
	t.Parallel()

	t.Run("scalar fields are valid", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.FieldDef{
				"name": {Type: core.TIDString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("array with elem_type is valid", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.FieldDef{
				"tags": {Type: core.TIDArray, ElemType: core.TIDString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("array without elem_type is rejected in all modes", func(t *testing.T) {
		for _, mode := range []schema.Mode{
			schema.ModeStrict,
			schema.ModeDynamic,
			schema.ModeFlexible,
		} {
			t.Run(string(mode), func(t *testing.T) {
				def := &schema.Def{
					URI:  core.MustParseURI("xdb://x/y"),
					Mode: mode,
					Fields: map[string]schema.FieldDef{
						"tags": {Type: core.TIDArray},
					},
				}
				err := def.Validate()
				require.ErrorIs(t, err, schema.ErrInvalidField)
				assert.Contains(t, err.Error(), "tags")
			})
		}
	})
}

func TestValidateUpdate(t *testing.T) {
	t.Parallel()

	mkDef := func(fields map[string]schema.FieldDef) *schema.Def {
		return &schema.Def{
			URI:    core.MustParseURI("xdb://x/y"),
			Mode:   schema.ModeStrict,
			Fields: fields,
		}
	}

	t.Run("adding a new field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.FieldDef{"a": {Type: core.TIDString}})
		new := mkDef(map[string]schema.FieldDef{
			"a": {Type: core.TIDString},
			"b": {Type: core.TIDInteger},
		})
		assert.NoError(t, schema.ValidateUpdate(old, new))
	})

	t.Run("removing a field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.FieldDef{
			"a": {Type: core.TIDString},
			"b": {Type: core.TIDInteger},
		})
		new := mkDef(map[string]schema.FieldDef{"a": {Type: core.TIDString}})
		assert.NoError(t, schema.ValidateUpdate(old, new))
	})

	t.Run("changing field type is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.FieldDef{"a": {Type: core.TIDString}})
		new := mkDef(map[string]schema.FieldDef{"a": {Type: core.TIDInteger}})
		err := schema.ValidateUpdate(old, new)
		require.ErrorIs(t, err, schema.ErrImmutableField)
	})

	t.Run("changing array elem_type is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDString},
		})
		new := mkDef(map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDInteger},
		})
		err := schema.ValidateUpdate(old, new)
		require.ErrorIs(t, err, schema.ErrImmutableField)
		assert.Contains(t, err.Error(), "elem_type")
	})

	t.Run("same elem_type is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDString},
		})
		new := mkDef(map[string]schema.FieldDef{
			"tags": {Type: core.TIDArray, ElemType: core.TIDString},
		})
		assert.NoError(t, schema.ValidateUpdate(old, new))
	})
}

func TestValidateTuples_AllTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tid   core.TID
		value any
	}{
		{"boolean", core.TIDBoolean, true},
		{"integer", core.TIDInteger, int64(42)},
		{"unsigned", core.TIDUnsigned, uint64(42)},
		{"float", core.TIDFloat, 3.14},
		{"string", core.TIDString, "hello"},
		{"bytes", core.TIDBytes, []byte("data")},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			def := &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/test"),
				Mode: schema.ModeStrict,
				Fields: map[string]schema.FieldDef{
					"field": {Type: tt.tid},
				},
			}

			tuples := []*core.Tuple{
				core.NewTuple("com.example/test/1", "field", tt.value),
			}

			err := schema.ValidateTuples(def, tuples)
			assert.NoError(t, err)
		})
	}
}
