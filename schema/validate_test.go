package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
