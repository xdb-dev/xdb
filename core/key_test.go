package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestKey(t *testing.T) {
	t.Run("NewKey", func(t *testing.T) {
		key := core.NewKey("User", "123", "name")

		assert.NotNil(t, key)
		assert.Equal(t, "User/123/name", key.String())
		assert.Equal(t, "Key(User/123/name)", key.GoString())
	})

	t.Run("With Method", func(t *testing.T) {
		baseKey := core.NewKey("User", "789")
		extendedKey := baseKey.With("email")

		assert.NotNil(t, extendedKey)
		assert.Equal(t, "User/789/email", extendedKey.String())
		assert.Equal(t, "User/789", baseKey.String())
	})

	t.Run("With Multiple Parts", func(t *testing.T) {
		baseKey := core.NewKey("Post", "101")
		extendedKey := baseKey.With("authored_by", "User", "123")

		assert.NotNil(t, extendedKey)
		assert.Equal(t, "Post/101/authored_by/User/123", extendedKey.String())
	})

	t.Run("Value Method", func(t *testing.T) {
		key := core.NewKey("Product", "202", "price")
		tuple := key.Value(29.99)

		assert.NotNil(t, tuple)
		assert.Equal(t, key.String(), tuple.Key().String())
		assert.Equal(t, 29.99, tuple.ToFloat())
		assert.Equal(t, "Product/202/price", tuple.Key().String())
	})

	t.Run("Unwrap Method", func(t *testing.T) {
		key := core.NewKey("Product", "202", "price")
		parts := key.Unwrap()

		assert.Equal(t, []string{"Product", "202", "price"}, parts)
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Empty Parts", func(t *testing.T) {
			key := core.NewKey("", "", "")

			assert.Equal(t, "", key.Kind())
			assert.Equal(t, "", key.ID())
			assert.Equal(t, "", key.Attr())
			assert.Equal(t, "//", key.String())
		})

		t.Run("Nil Parts", func(t *testing.T) {
			key := core.NewKey()

			assert.Nil(t, key)
		})

		t.Run("With Empty Parts", func(t *testing.T) {
			baseKey := core.NewKey("User", "123")
			extendedKey := baseKey.With("", "empty")

			assert.Equal(t, "User/123//empty", extendedKey.String())
		})
	})

	t.Run("Immutability", func(t *testing.T) {
		originalKey := core.NewKey("User", "123", "name")
		modifiedKey := originalKey.With("email")

		assert.Equal(t, "User/123/name", originalKey.String())
		assert.Equal(t, "User/123/name/email", modifiedKey.String())

		assert.NotEqual(t, originalKey, modifiedKey)
	})
}
