package schema_test

import (
	"encoding/json"
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
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString},
			"age":  {Type: core.TypeInt},
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
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString},
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
		Fields: map[string]schema.Field{
			"age": {Type: core.TypeInt},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("com.example/users/1", "age", "not an int"),
	}

	err := schema.ValidateTuples(def, tuples)
	assert.ErrorIs(t, err, schema.ErrTypeMismatch)
}

// Under the fixed mode semantics, declared fields type-check in every mode.
func TestValidateTuples_Flexible_TypeChecksDeclaredFields(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeFlexible,
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString},
		},
	}

	t.Run("rejects wrong type for declared field", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "name", int64(42)),
		}
		err := schema.ValidateTuples(def, tuples)
		assert.ErrorIs(t, err, schema.ErrTypeMismatch)
	})

	t.Run("accepts correct type for declared field", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "name", "Alice"),
		}
		err := schema.ValidateTuples(def, tuples)
		assert.NoError(t, err)
	})

	t.Run("ignores unknown fields", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "unknown", "value"),
		}
		err := schema.ValidateTuples(def, tuples)
		assert.NoError(t, err)
	})
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
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString},
		},
	}

	err := schema.ValidateTuples(def, nil)
	assert.NoError(t, err)
}

func TestCheckRequired(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString, Required: true},
			"age":  {Type: core.TypeInt},
		},
	}

	t.Run("passes when required field present", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "name", "Alice"),
		}
		assert.NoError(t, schema.CheckRequired(def, tuples))
	})

	t.Run("explicit null satisfies required", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "name", nil),
		}
		assert.NoError(t, schema.CheckRequired(def, tuples))
	})

	t.Run("fails when required field missing", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/users/1", "age", int64(30)),
		}
		err := schema.CheckRequired(def, tuples)
		require.ErrorIs(t, err, schema.ErrMissingRequired)
		assert.Contains(t, err.Error(), "name")
	})
}

func TestValidateTuples_Array_ElemTypeMismatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/posts"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
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
		Fields: map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
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
		assert.Equal(t, core.TIDString, f.Type.ID())
		assert.Equal(t, core.TID(""), f.Type.ElemTypeID())
	})

	t.Run("array value captures elem type", func(t *testing.T) {
		f := schema.InferField(core.ArrayVal(core.TIDString, core.StringVal("a")))
		assert.Equal(t, core.TIDArray, f.Type.ID())
		assert.Equal(t, core.TIDString, f.Type.ElemTypeID())
	})

	t.Run("array value of integers", func(t *testing.T) {
		f := schema.InferField(core.ArrayVal(core.TIDInteger, core.IntVal(1)))
		assert.Equal(t, core.TIDArray, f.Type.ID())
		assert.Equal(t, core.TIDInteger, f.Type.ElemTypeID())
	})
}

func TestEvolveDynamic(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when no new fields", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/events"),
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString},
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
			Fields: map[string]schema.Field{},
		}
		tuples := []*core.Tuple{
			core.NewTuple("com.example/events/1", "count", int64(5)),
		}

		newFields, err := schema.EvolveDynamic(def, tuples)
		require.NoError(t, err)
		require.Len(t, newFields, 1)
		assert.Equal(t, core.TIDInteger, newFields["count"].Type.ID())
	})

	t.Run("infers new array field with elem type", func(t *testing.T) {
		def := &schema.Def{
			URI:    core.MustParseURI("xdb://com.example/events"),
			Mode:   schema.ModeDynamic,
			Fields: map[string]schema.Field{},
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
		assert.Equal(t, core.TIDArray, newFields["tags"].Type.ID())
		assert.Equal(t, core.TIDString, newFields["tags"].Type.ElemTypeID())
	})

	t.Run("rejects mismatched elem type on known array field", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://com.example/events"),
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
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
			Fields: map[string]schema.Field{},
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
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("dotted attr paths are valid", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"profile.name": {Type: core.TypeString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("array with elem_type is valid", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
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
					Fields: map[string]schema.Field{
						"tags": {Type: core.NewArrayType("")},
					},
				}
				err := def.Validate()
				require.ErrorIs(t, err, schema.ErrInvalidField)
				assert.Contains(t, err.Error(), "tags")
			})
		}
	})

	t.Run("empty mode is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI: core.MustParseURI("xdb://x/y"),
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidMode)
	})

	t.Run("unrecognized mode is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.Mode("bogus"),
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidMode)
		assert.Contains(t, err.Error(), "flexible")
		assert.Contains(t, err.Error(), "strict")
		assert.Contains(t, err.Error(), "dynamic")
	})

	t.Run("invalid field name is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"bad name!": {Type: core.TypeString},
			},
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "bad name!")
	})

	t.Run("prefix conflict is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"profile":      {Type: core.TypeString},
				"profile.name": {Type: core.TypeString},
			},
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "profile")
	})

	t.Run("shared prefix without dot boundary is allowed", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"profile":     {Type: core.TypeString},
				"profilename": {Type: core.TypeString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("object array with valid items is accepted", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"sku": {Type: core.TypeString},
						"qty": {Type: core.TypeInt},
					},
				},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("object array with invalid item name is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"bad name!": {Type: core.TypeString},
					},
				},
			},
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "bad name!")
	})

	t.Run("object array with nested array missing elem_type is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"tags": {Type: core.NewArrayType("")},
					},
				},
			},
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "tags")
	})

	t.Run("items on a non-object-array field is rejected", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"name": {
					Type:  core.TypeString,
					Items: map[string]schema.Field{"x": {Type: core.TypeString}},
				},
			},
		}
		err := def.Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "name")
	})
}

func objectArrayDef(items map[string]schema.Field) *schema.Def {
	return &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/orders"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"lines": {
				Type:  core.NewArrayType(core.TIDJSON),
				Items: items,
			},
		},
	}
}

func objectArrayTuple(elems ...json.RawMessage) []*core.Tuple {
	vals := make([]*core.Value, len(elems))
	for i, e := range elems {
		vals[i] = core.JSONVal(e)
	}
	return []*core.Tuple{
		core.NewTuple(
			"com.example/orders/1",
			"lines",
			core.ArrayVal(core.TIDJSON, vals...),
		),
	}
}

func TestValidateTuples_ObjectArray(t *testing.T) {
	t.Parallel()

	items := map[string]schema.Field{
		"sku": {Type: core.TypeString, Required: true},
		"qty": {Type: core.TypeInt},
	}

	t.Run("valid elements pass", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(
			json.RawMessage(`{"sku":"A-1","qty":3}`),
			json.RawMessage(`{"sku":"B-2","qty":5}`),
		)
		assert.NoError(t, schema.ValidateTuples(def, tuples))
	})

	t.Run("member type mismatch is rejected", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(json.RawMessage(`{"sku":"A-1","qty":"three"}`))
		err := schema.ValidateTuples(def, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
		assert.Contains(t, err.Error(), "qty")
	})

	t.Run("missing required member is rejected", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(json.RawMessage(`{"qty":3}`))
		err := schema.ValidateTuples(def, tuples)
		require.ErrorIs(t, err, schema.ErrMissingRequired)
		assert.Contains(t, err.Error(), "sku")
	})

	t.Run("unknown member is rejected", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(json.RawMessage(`{"sku":"A-1","extra":true}`))
		err := schema.ValidateTuples(def, tuples)
		require.ErrorIs(t, err, schema.ErrUnknownField)
		assert.Contains(t, err.Error(), "extra")
	})

	t.Run("non-object element is rejected", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(json.RawMessage(`"not-an-object"`))
		err := schema.ValidateTuples(def, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})

	t.Run("explicit null member satisfies required", func(t *testing.T) {
		def := objectArrayDef(items)
		tuples := objectArrayTuple(json.RawMessage(`{"sku":null,"qty":3}`))
		assert.NoError(t, schema.ValidateTuples(def, tuples))
	})

	t.Run("typed scalar members validate", func(t *testing.T) {
		def := objectArrayDef(map[string]schema.Field{
			"placed": {Type: core.TypeTime, Required: true},
			"blob":   {Type: core.TypeBytes},
		})
		tuples := objectArrayTuple(
			json.RawMessage(`{"placed":"2026-07-17T10:00:00Z","blob":"SGVsbG8="}`),
		)
		assert.NoError(t, schema.ValidateTuples(def, tuples))
	})

	t.Run("bad time member is rejected", func(t *testing.T) {
		def := objectArrayDef(map[string]schema.Field{
			"placed": {Type: core.TypeTime},
		})
		tuples := objectArrayTuple(json.RawMessage(`{"placed":"not-a-time"}`))
		err := schema.ValidateTuples(def, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
		assert.Contains(t, err.Error(), "placed")
	})

	t.Run("one level of nesting validates", func(t *testing.T) {
		def := objectArrayDef(map[string]schema.Field{
			"sku": {Type: core.TypeString},
			"tags": {
				Type:  core.NewArrayType(core.TIDJSON),
				Items: map[string]schema.Field{"name": {Type: core.TypeString, Required: true}},
			},
		})

		t.Run("valid nested", func(t *testing.T) {
			tuples := objectArrayTuple(
				json.RawMessage(`{"sku":"A","tags":[{"name":"x"},{"name":"y"}]}`),
			)
			assert.NoError(t, schema.ValidateTuples(def, tuples))
		})

		t.Run("nested missing required", func(t *testing.T) {
			tuples := objectArrayTuple(
				json.RawMessage(`{"sku":"A","tags":[{"other":"x"}]}`),
			)
			err := schema.ValidateTuples(def, tuples)
			require.ErrorIs(t, err, schema.ErrUnknownField)
		})
	})
}

func TestDef_Validate_IndexedUnique(t *testing.T) {
	t.Parallel()

	mkDef := func(f schema.Field) *schema.Def {
		return &schema.Def{
			URI:    core.MustParseURI("xdb://x/y"),
			Mode:   schema.ModeStrict,
			Fields: map[string]schema.Field{"f": f},
		}
	}

	t.Run("indexed scalar is valid", func(t *testing.T) {
		assert.NoError(t, mkDef(schema.Field{Type: core.TypeString, Indexed: true}).Validate())
	})

	t.Run("unique scalar is valid", func(t *testing.T) {
		assert.NoError(t, mkDef(schema.Field{Type: core.TypeInt, Unique: true}).Validate())
	})

	t.Run("indexed array is rejected", func(t *testing.T) {
		err := mkDef(schema.Field{Type: core.NewArrayType(core.TIDString), Indexed: true}).Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
		assert.Contains(t, err.Error(), "f")
	})

	t.Run("unique array is rejected", func(t *testing.T) {
		err := mkDef(schema.Field{Type: core.NewArrayType(core.TIDString), Unique: true}).Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
	})

	t.Run("indexed json is rejected", func(t *testing.T) {
		err := mkDef(schema.Field{Type: core.NewType(core.TIDJSON), Indexed: true}).Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
	})

	t.Run("unique json is rejected", func(t *testing.T) {
		err := mkDef(schema.Field{Type: core.NewType(core.TIDJSON), Unique: true}).Validate()
		require.ErrorIs(t, err, schema.ErrInvalidField)
	})
}

func TestValidateUpdate(t *testing.T) {
	t.Parallel()

	mkDef := func(fields map[string]schema.Field) *schema.Def {
		return &schema.Def{
			URI:    core.MustParseURI("xdb://x/y"),
			Mode:   schema.ModeStrict,
			Fields: fields,
		}
	}

	t.Run("adding a new field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeInt},
		})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("removing a field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeInt},
		})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("changing field type is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeInt}})
		err := schema.ValidateUpdate(old, updated)
		require.ErrorIs(t, err, schema.ErrImmutableField)
	})

	t.Run("changing array elem_type is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
		})
		updated := mkDef(map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDInteger)},
		})
		err := schema.ValidateUpdate(old, updated)
		require.ErrorIs(t, err, schema.ErrImmutableField)
		assert.Contains(t, err.Error(), "elem_type")
	})

	t.Run("same elem_type is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
		})
		updated := mkDef(map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
		})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("changing mode is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated.Mode = schema.ModeFlexible

		err := schema.ValidateUpdate(old, updated)
		require.ErrorIs(t, err, schema.ErrImmutableMode)
		assert.Contains(t, err.Error(), "mode")
	})

	t.Run("same mode is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("toggling indexed on an existing field is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString, Indexed: true}})
		err := schema.ValidateUpdate(old, updated)
		require.ErrorIs(t, err, schema.ErrImmutableField)
		assert.Contains(t, err.Error(), "indexed")
	})

	t.Run("toggling unique on an existing field is rejected", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString, Unique: true}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		err := schema.ValidateUpdate(old, updated)
		require.ErrorIs(t, err, schema.ErrImmutableField)
		assert.Contains(t, err.Error(), "unique")
	})

	t.Run("adding a new indexed field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		updated := mkDef(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeString, Indexed: true, Unique: true},
		})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("removing an indexed field is allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeString, Indexed: true},
		})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString}})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})

	t.Run("unchanged flags are allowed", func(t *testing.T) {
		old := mkDef(map[string]schema.Field{"a": {Type: core.TypeString, Indexed: true, Unique: true}})
		updated := mkDef(map[string]schema.Field{"a": {Type: core.TypeString, Indexed: true, Unique: true}})
		assert.NoError(t, schema.ValidateUpdate(old, updated))
	})
}

func TestValidateTuples_AllTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		typ   core.Type
		value any
	}{
		{"boolean", core.TypeBool, true},
		{"integer", core.TypeInt, int64(42)},
		{"unsigned", core.TypeUnsigned, uint64(42)},
		{"float", core.TypeFloat, 3.14},
		{"string", core.TypeString, "hello"},
		{"bytes", core.TypeBytes, []byte("data")},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			def := &schema.Def{
				URI:  core.MustParseURI("xdb://com.example/test"),
				Mode: schema.ModeStrict,
				Fields: map[string]schema.Field{
					"field": {Type: tt.typ},
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

func TestValidModes(t *testing.T) {
	t.Parallel()

	want := []schema.Mode{schema.ModeFlexible, schema.ModeStrict, schema.ModeDynamic}
	assert.Equal(t, want, schema.ValidModes())
}

func TestNextRevision(t *testing.T) {
	t.Parallel()

	t.Run("unconditional bumps from current", func(t *testing.T) {
		next, err := schema.NextRevision(5, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(6), next)
	})

	t.Run("matching base bumps", func(t *testing.T) {
		next, err := schema.NextRevision(5, 5)
		require.NoError(t, err)
		assert.Equal(t, int64(6), next)
	})

	t.Run("stale base conflicts", func(t *testing.T) {
		_, err := schema.NextRevision(5, 4)
		require.ErrorIs(t, err, core.ErrConflict)
	})

	t.Run("conflict does not wrap ErrNotFound", func(t *testing.T) {
		_, err := schema.NextRevision(5, 4)
		assert.NotErrorIs(t, err, core.ErrNotFound)
	})
}

func TestDef_Validate_ReservedFieldNames(t *testing.T) {
	t.Parallel()

	modes := []schema.Mode{
		schema.ModeStrict,
		schema.ModeDynamic,
		schema.ModeFlexible,
	}

	t.Run("underscore-prefixed field names are rejected", func(t *testing.T) {
		for _, name := range []string{"_id", "_version", "_updated", "_custom"} {
			for _, mode := range modes {
				t.Run(name+"/"+string(mode), func(t *testing.T) {
					def := &schema.Def{
						URI:  core.MustParseURI("xdb://x/y"),
						Mode: mode,
						Fields: map[string]schema.Field{
							name: {Type: core.TypeString},
						},
					}
					err := def.Validate()
					require.ErrorIs(t, err, schema.ErrInvalidField)
					assert.Contains(t, err.Error(), name)
				})
			}
		}
	})

	t.Run("underscore inside a dotted path is allowed", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"profile._id": {Type: core.TypeString},
			},
		}
		assert.NoError(t, def.Validate())
	})

	t.Run("underscore-prefixed names are allowed inside object array items", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://x/y"),
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"docs": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"_id": {Type: core.TypeString},
					},
				},
			},
		}
		assert.NoError(t, def.Validate())
	})
}

func TestIsSystemField(t *testing.T) {
	t.Parallel()

	assert.True(t, schema.IsSystemField(schema.FieldID))
	assert.True(t, schema.IsSystemField(schema.FieldVersion))
	assert.True(t, schema.IsSystemField(schema.FieldUpdated))
	assert.True(t, schema.IsSystemField("_anything"))

	assert.False(t, schema.IsSystemField("name"))
	assert.False(t, schema.IsSystemField("profile._id"))
	assert.False(t, schema.IsSystemField(""))
}

func TestValidateTuples_ExplicitNull(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}
	record := core.NewRecord("com.example", "users", "u1").Set("title", nil)

	assert.NoError(t, schema.ValidateTuples(def, record.Tuples()),
		"an explicit null carries no type to check")
}

func TestEvolveDynamic_ExplicitNull(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/users"),
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}
	record := core.NewRecord("com.example", "users", "u1").
		Set("title", nil).
		Set("extra", nil)

	newFields, err := schema.EvolveDynamic(def, record.Tuples())
	require.NoError(t, err)
	assert.Empty(t, newFields, "a null carries no type to infer")
}
