package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestTuple(t *testing.T) {
	t.Parallel()

	t.Run("NewTuple", func(t *testing.T) {
		tuple := core.NewTuple(
			"com.example.posts",
			"123",
			"title",
			"Hello, World!",
		)

		assert.NotNil(t, tuple)
		assert.Equal(t, "com.example.posts", tuple.Repo())
		assert.Equal(t, "title", tuple.Attr().String())
		assert.Equal(t, "123", tuple.ID().String())
		assert.Equal(t, "Hello, World!", tuple.Value().ToString())
		assert.Equal(t, "xdb://com.example.posts/123#title", tuple.URI().String())
		assert.Equal(t, "Tuple(com.example.posts, 123, title, Value(STRING, Hello, World!))", tuple.GoString())
	})

	t.Run("Hierarchy ID", func(t *testing.T) {
		id := core.NewID("org-123", "user-456")
		tuple1 := core.NewTuple("com.example.users", id, "name", "John")
		assert.Equal(t, "com.example.users", tuple1.Repo())
		assert.Equal(t, "org-123/user-456", tuple1.ID().String())
		assert.Equal(t, "name", tuple1.Attr().String())
	})

	t.Run("Nested Attr", func(t *testing.T) {
		attr := core.NewAttr("profile", "name")
		tuple := core.NewTuple("com.example.users", "123", attr, "John")
		assert.Equal(t, "profile.name", tuple.Attr().String())
	})

	t.Run("Nil Value", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "123", "name", nil)
		assert.NotNil(t, tuple)
		assert.True(t, tuple.IsNil())
	})

	t.Run("Invalid ID Type", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewTuple("com.example.users", 123, "name", "John") // Invalid ID type
		})
	})

	t.Run("Invalid Attr Type", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewTuple("com.example.users", "123", 123, "John") // Invalid Attr type
		})
	})

	t.Run("Empty ID", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "", "name", "John")
		assert.Equal(t, "", tuple.ID().String())
	})

	t.Run("Empty Attr", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "123", "", "John")
		assert.Equal(t, "", tuple.Attr().String())
	})
}
