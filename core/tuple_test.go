package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestTuple(t *testing.T) {
	t.Parallel()

	t.Run("NewTuple", func(t *testing.T) {
		tuple := core.NewTuple("com.example/posts/123", "title", "Hello, World!")

		assert.NotNil(t, tuple)
		assert.Equal(t, "com.example", tuple.NS().String())
		assert.Equal(t, "posts", tuple.Schema().String())
		assert.Equal(t, "123", tuple.ID().String())
		assert.Equal(t, "title", tuple.Attr().String())
		assert.Equal(t, "Hello, World!", tuple.Value().ToString())
		assert.Equal(t, "xdb://com.example.posts/123#title", tuple.URI().String())
		assert.Equal(t, "Tuple(com.example.posts, 123, title, Value(STRING, Hello, World!))", tuple.GoString())
	})

	t.Run("Hierarchy ID", func(t *testing.T) {
		tuple := core.NewTuple(
			"com.example/posts/org-123/user-456",
			"name",
			"John",
		)
		assert.Equal(t, "com.example", tuple.NS().String())
		assert.Equal(t, "posts", tuple.Schema().String())
		assert.Equal(t, "org-123/user-456", tuple.ID().String())
		assert.Equal(t, "name", tuple.Attr().String())
	})

	t.Run("Nested Attr", func(t *testing.T) {
		tuple := core.NewTuple(
			"com.example/posts/123",
			"profile.name",
			"John",
		)
		assert.Equal(t, "com.example", tuple.NS().String())
		assert.Equal(t, "posts", tuple.Schema().String())
		assert.Equal(t, "123", tuple.ID().String())
		assert.Equal(t, "profile.name", tuple.Attr().String())
		assert.Equal(t, "John", tuple.Value().ToString())
	})

	t.Run("Nil Value", func(t *testing.T) {
		tuple := core.NewTuple(
			"com.example/posts/123",
			"name",
			nil,
		)
		assert.NotNil(t, tuple)
		assert.True(t, tuple.Value().IsNil())
	})
}
