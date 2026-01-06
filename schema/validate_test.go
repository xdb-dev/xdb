package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestInferFields(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name: "users",
		Mode: schema.ModeDynamic,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "name", "Alice"),
		core.NewTuple("test/users/1", "age", int64(30)),
		core.NewTuple("test/users/2", "age", int64(25)),
	}

	fields, err := schema.InferFields(def, tuples)

	assert.NoError(t, err)
	assert.Len(t, fields, 1)
	assert.Equal(t, "age", fields[0].Name)
	assert.Equal(t, core.TypeInt, fields[0].Type)
}

func TestInferFields_NoDuplicates(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name:   "users",
		Mode:   schema.ModeDynamic,
		Fields: []*schema.FieldDef{},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "name", "Alice"),
		core.NewTuple("test/users/2", "name", "Bob"),
	}

	fields, err := schema.InferFields(def, tuples)

	assert.NoError(t, err)
	assert.Len(t, fields, 1)
	assert.Equal(t, "name", fields[0].Name)
}

func TestValidateTuples_Strict_UnknownField(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name: "users",
		Mode: schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "unknown", "value"),
	}

	err := schema.ValidateTuples(def, tuples)

	assert.ErrorIs(t, err, schema.ErrUnknownField)
}

func TestValidateTuples_Flexible_UnknownField(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name: "users",
		Mode: schema.ModeFlexible,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "unknown", "value"),
	}

	err := schema.ValidateTuples(def, tuples)

	assert.NoError(t, err)
}

func TestValidateTuples_TypeMismatch(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name: "users",
		Mode: schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "age", Type: core.TypeInt},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "age", "not an int"),
	}

	err := schema.ValidateTuples(def, tuples)

	assert.ErrorIs(t, err, schema.ErrTypeMismatch)
}

func TestValidateTuples_Valid(t *testing.T) {
	t.Parallel()

	def := &schema.Def{
		Name: "users",
		Mode: schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
			{Name: "age", Type: core.TypeInt},
		},
	}

	tuples := []*core.Tuple{
		core.NewTuple("test/users/1", "name", "Alice"),
		core.NewTuple("test/users/1", "age", int64(30)),
	}

	err := schema.ValidateTuples(def, tuples)

	assert.NoError(t, err)
}
