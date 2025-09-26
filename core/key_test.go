package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestKey(t *testing.T) {
	t.Parallel()

	t.Run("NewKey_WithID", func(t *testing.T) {
		id := core.NewID("User", "123")
		key := core.NewKey(id)
		assert.NotNil(t, key)
		assert.Equal(t, "User/123", key.String())
		assert.Equal(t, "Key(User/123)", key.GoString())
	})

	t.Run("NewKey_WithAttr", func(t *testing.T) {
		id := core.NewID("User", "123")
		key := core.NewKey(id, "name")
		assert.NotNil(t, key)
		assert.Equal(t, "User/123/name", key.String())
		assert.Equal(t, "Key(User/123/name)", key.GoString())
	})

	t.Run("Value Method", func(t *testing.T) {
		id := core.NewID("Product", "202")
		key := core.NewKey(id, "price")
		tuple := key.Value(29.99)

		assert.NotNil(t, tuple)
		assert.Equal(t, key.String(), tuple.Key().String())
		assert.Equal(t, 29.99, tuple.ToFloat())
		assert.Equal(t, "Product/202/price", tuple.Key().String())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Empty Parts", func(t *testing.T) {
			key := core.NewKey("", "")

			assert.Equal(t, "", key.ID().String())
			assert.Equal(t, "", key.Attr().String())
		})

		t.Run("Nil Parts", func(t *testing.T) {
			key := core.NewKey()

			assert.Nil(t, key)
		})

		t.Run("Invalid Key Pattern - Too Many Parts", func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewKey("a", "b", "c")
			})
		})

		t.Run("Different ID Types", func(t *testing.T) {
			// Test with ID type
			id := core.NewID("User", "123")
			key1 := core.NewKey(id)
			assert.Equal(t, "User/123", key1.String())

			// Test with []string
			key2 := core.NewKey([]string{"Product", "456"})
			assert.Equal(t, "Product/456", key2.String())
		})

		t.Run("Different Attr Types", func(t *testing.T) {
			id := core.NewID("User", "123")

			// Test with Attr type
			attr := core.NewAttr("profile", "name")
			key1 := core.NewKey(id, attr)
			assert.Equal(t, "User/123/profile.name", key1.String())

			// Test with []string
			key2 := core.NewKey(id, []string{"settings", "theme"})
			assert.Equal(t, "User/123/settings.theme", key2.String())
		})

		t.Run("Invalid ID Type", func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewKey(123) // Invalid ID type
			})
		})

		t.Run("Invalid Attr Type", func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewKey("User", 123) // Invalid Attr type
			})
		})
	})
}
