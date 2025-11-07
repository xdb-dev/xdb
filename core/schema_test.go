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
				Type:        core.TypeString,
			},
			{
				Name:        "email",
				Description: "User's email address",
				Type:        core.TypeString,
			},
			{
				Name:        "age",
				Description: "User's age",
				Type:        core.TypeInt,
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
				Type: core.TypeString,
			},
			{
				Name: "likes",
				Type: core.TypeInt,
			},
		},
	}

	t.Run("Valid String Tuple", func(t *testing.T) {
		tuple := core.NewTuple("com.example.posts", "123", "title", "My Post")
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Valid Integer Tuple", func(t *testing.T) {
		tuple := core.NewTuple("com.example.posts", "123", "likes", 42)
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch", func(t *testing.T) {
		tuple := core.NewTuple("com.example.posts", "123", "likes", "many") // Should be integer
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Field Not In Schema", func(t *testing.T) {
		tuple := core.NewTuple("com.example.posts", "123", "unknown", "value")
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
				Type: core.TypeString,
			},
			{
				Name: "price",
				Type: core.TypeFloat,
			},
			{
				Name: "category.name",
				Type: core.TypeString,
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
		Type: core.TypeString,
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

func TestSchema_ArrayTypes(t *testing.T) {
	t.Parallel()

	schema := &core.Schema{
		Name: "Post",
		Fields: []*core.FieldSchema{
			{
				Name: "tags",
				Type: core.NewArrayType(core.TypeIDString),
			},
			{
				Name: "scores",
				Type: core.NewArrayType(core.TypeIDInteger),
			},
		},
	}

	t.Run("Valid String Array", func(t *testing.T) {
		tuple := core.NewTuple("Post", "123", "tags", []string{"go", "xdb", "database"})
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Valid Integer Array", func(t *testing.T) {
		tuple := core.NewTuple("Post", "123", "scores", []int64{1, 2, 3, 4, 5})
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch - Wrong Array Element Type", func(t *testing.T) {
		// tags should be string array, not int array
		tuple := core.NewTuple("Post", "123", "tags", []int64{1, 2, 3})
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Type Mismatch - Scalar Instead of Array", func(t *testing.T) {
		tuple := core.NewTuple("Post", "123", "tags", "single-tag")
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})
}

func TestSchema_MapTypes(t *testing.T) {
	t.Parallel()

	schema := &core.Schema{
		Name: "Config",
		Fields: []*core.FieldSchema{
			{
				Name: "settings",
				Type: core.NewMapType(core.TypeIDString, core.TypeIDString),
			},
			{
				Name: "counts",
				Type: core.NewMapType(core.TypeIDString, core.TypeIDInteger),
			},
		},
	}

	t.Run("Valid String-String Map", func(t *testing.T) {
		settings := map[string]string{
			"theme": "dark",
			"lang":  "en",
		}
		tuple := core.NewTuple("Config", "123", "settings", settings)
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Valid String-Integer Map", func(t *testing.T) {
		counts := map[string]int64{
			"views":  100,
			"likes":  25,
			"shares": 10,
		}
		tuple := core.NewTuple("Config", "123", "counts", counts)
		err := schema.ValidateTuple(tuple)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch - Wrong Value Type", func(t *testing.T) {
		// counts expects int64 values, not string
		wrongCounts := map[string]string{
			"views": "many",
		}
		tuple := core.NewTuple("Config", "123", "counts", wrongCounts)
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Type Mismatch - Scalar Instead of Map", func(t *testing.T) {
		tuple := core.NewTuple("Config", "123", "settings", "just-a-string")
		err := schema.ValidateTuple(tuple)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})
}

func TestSchema_NestedFields(t *testing.T) {
	t.Parallel()

	schema := &core.Schema{
		Name: "User",
		Fields: []*core.FieldSchema{
			{
				Name: "name",
				Type: core.TypeString,
			},
			{
				Name: "profile.bio",
				Type: core.TypeString,
			},
			{
				Name: "profile.age",
				Type: core.TypeInt,
			},
			{
				Name: "settings.notifications.email",
				Type: core.TypeBool,
			},
		},
		Required: []string{"name"},
	}

	t.Run("Valid Nested Fields", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John")
		record.Set(core.NewAttr("profile", "bio"), "Software Engineer")
		record.Set(core.NewAttr("profile", "age"), 30)
		record.Set(core.NewAttr("settings", "notifications", "email"), true)

		err := schema.ValidateRecord(record)
		assert.NoError(t, err)
	})

	t.Run("Type Mismatch in Nested Field", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set("name", "John")
		record.Set(core.NewAttr("profile", "age"), "thirty") // Should be int

		err := schema.ValidateRecord(record)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrTypeMismatch)
	})

	t.Run("Missing Required Top-Level Field", func(t *testing.T) {
		record := core.NewRecord("User", "123")
		record.Set(core.NewAttr("profile", "bio"), "Engineer")

		err := schema.ValidateRecord(record)
		assert.Error(t, err)
		assert.ErrorIs(t, err, core.ErrRequiredFieldMissing)
	})
}
