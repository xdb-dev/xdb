package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestTuple(t *testing.T) {
	t.Run("NewTuple", func(t *testing.T) {
		tuple := core.NewTuple(
			[]string{"Post", "123"},
			"title",
			"Hello, World!",
		)

		assert.NotNil(t, tuple)
		assert.Equal(t, "title", tuple.Attr().String())
		assert.Equal(t, "Post/123", tuple.ID().String())
		assert.Equal(t, "Hello, World!", tuple.Value().ToString())
		assert.Equal(t, "Tuple(Post/123, title, Value(STRING, Hello, World!))", tuple.GoString())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Different ID Types", func(t *testing.T) {
			// Test with ID type
			id := core.NewID("User", "123")
			tuple1 := core.NewTuple(id, "name", "John")
			assert.Equal(t, "User/123", tuple1.ID().String())
			assert.Equal(t, "name", tuple1.Attr().String())

			// Test with string
			tuple2 := core.NewTuple("Product", "price", 29.99)
			assert.Equal(t, "Product", tuple2.ID().String())
			assert.Equal(t, "price", tuple2.Attr().String())
		})

		t.Run("Different Attr Types", func(t *testing.T) {
			// Test with Attr type
			attr := core.NewAttr("profile", "name")
			tuple1 := core.NewTuple("User", attr, "John")
			assert.Equal(t, "profile.name", tuple1.Attr().String())

			// Test with []string
			tuple2 := core.NewTuple("User", []string{"settings", "theme"}, "dark")
			assert.Equal(t, "settings.theme", tuple2.Attr().String())
		})

		t.Run("Nil Value", func(t *testing.T) {
			tuple := core.NewTuple("User", "name", nil)
			assert.NotNil(t, tuple)
			// Check if the value is nil by checking the tuple's value
			if tuple.Value() != nil {
				assert.True(t, tuple.Value().IsNil())
			}
		})

		t.Run("Invalid ID Type", func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewTuple(123, "name", "John") // Invalid ID type
			})
		})

		t.Run("Invalid Attr Type", func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewTuple("User", 123, "John") // Invalid Attr type
			})
		})

		t.Run("Empty ID", func(t *testing.T) {
			tuple := core.NewTuple("", "name", "John")
			assert.Equal(t, "", tuple.ID().String())
		})

		t.Run("Empty Attr", func(t *testing.T) {
			tuple := core.NewTuple("User", "", "John")
			assert.Equal(t, "", tuple.Attr().String())
		})

		t.Run("Key Method", func(t *testing.T) {
			tuple := core.NewTuple("User", "name", "John")
			key := tuple.Key()
			assert.NotNil(t, key)
			assert.Equal(t, "User/name", key.String())
		})

		t.Run("Value Method", func(t *testing.T) {
			tuple := core.NewTuple("User", "age", 25)
			value := tuple.Value()
			assert.NotNil(t, value)
			assert.Equal(t, int64(25), value.ToInt())
		})
	})
}
