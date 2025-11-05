package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestSchema_ValidateRecord(t *testing.T) {
	t.Parallel()

	schema := &core.Schema{
		Name:        "User",
		Description: "User record schema",
		Version:     "1.0.0",
		Fields: []*core.FieldSchema{
			{
				Name:        "name",
				Description: "User's full name",
				Type:        core.NewType(core.TypeIDString),
			},
			{
				Name:        "email",
				Description: "User's email address",
				Type:        core.NewType(core.TypeIDString),
			},
			{
				Name:        "age",
				Description: "User's age",
				Type:        core.NewType(core.TypeIDInteger),
			},
		},
		Required: []string{"name", "email"},
	}

	t.Run("Valid Record", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John Doe")
		record.Set("email", "john@example.com")
		record.Set("age", 25)

		err := schema.ValidateRecord(record)
		assert.NoError(t, err)
	})

	t.Run("Valid Record Without Optional Fields", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John Doe")
		record.Set("email", "john@example.com")

		err := schema.ValidateRecord(record)
		assert.NoError(t, err)
	})

	t.Run("Missing Required Field", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John Doe")
		// Missing required 'email' field

		err := schema.ValidateRecord(record)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrRequiredFieldMissing)
	})

	t.Run("Type Mismatch", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John Doe")
		record.Set("email", "john@example.com")
		record.Set("age", "twenty-five") // Should be integer

		err := schema.ValidateRecord(record)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Nil Record", func(t *testing.T) {
		err := schema.ValidateRecord(nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrEmptyRecord)
	})

	t.Run("Unknown Field Not In Schema", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John Doe")
		record.Set("email", "john@example.com")
		record.Set("unknown_field", "value")

		err := schema.ValidateRecord(record)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrFieldSchemaNotFound)
	})
}

func TestSchema_ValidateTuple(t *testing.T) {
	schema := &core.Schema{
		Name:    "Post",
		Version: "1.0.0",
		Fields: []*core.FieldSchema{
			{
				Name: "title",
				Type: core.NewType(core.TypeIDString),
			},
			{
				Name: "likes",
				Type: core.NewType(core.TypeIDInteger),
			},
		},
	}

	t.Run("Valid String Tuple", func(t *testing.T) {
		tuple := core.NewTuple("Post/123", "title", "My Post")
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Valid Integer Tuple", func(t *testing.T) {
		tuple := core.NewTuple("Post/123", "likes", 42)
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch", func(t *testing.T) {
		tuple := core.NewTuple("Post/123", "likes", "many") // Should be integer
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Field Not In Schema", func(t *testing.T) {
		tuple := core.NewTuple("Post/123", "unknown", "value")
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrFieldSchemaNotFound)
	})

	t.Run("Nil Tuple", func(t *testing.T) {
		err := schema.ValidateTuple(nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrEmptyTuple)
	})
}

func TestSchema_GetFieldSchema(t *testing.T) {
	schema := &core.Schema{
		Name: "Product",
		Fields: []*core.FieldSchema{
			{
				Name: "name",
				Type: core.NewType(core.TypeIDString),
			},
			{
				Name: "price",
				Type: core.NewType(core.TypeIDFloat),
			},
			{
				Name: "category.name",
				Type: core.NewType(core.TypeIDString),
			},
		},
	}

	t.Run("Get Existing Field", func(t *testing.T) {
		field := schema.GetFieldSchema("name")
		assert.NotNil(t, field)
		assert.Equal(t, "name", field.Name)
		assert.Equal(t, core.TypeIDString, field.Type.ID())
	})

	t.Run("Get Nested Field", func(t *testing.T) {
		field := schema.GetFieldSchema("category.name")
		assert.NotNil(t, field)
		assert.Equal(t, "category.name", field.Name)
		assert.Equal(t, core.TypeIDString, field.Type.ID())
	})

	t.Run("Get Non-existent Field", func(t *testing.T) {
		field := schema.GetFieldSchema("unknown")
		assert.Nil(t, field)
	})
}

func TestFieldSchema_ValidateValue(t *testing.T) {
	field := &core.FieldSchema{
		Name: "name",
		Type: core.NewType(core.TypeIDString),
	}

	t.Run("Valid", func(t *testing.T) {
		value := core.NewValue("John Doe")
		err := field.ValidateValue(value)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch", func(t *testing.T) {
		value := core.NewValue(123)
		err := field.ValidateValue(value)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})
}
