package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestKey(t *testing.T) {
	id := core.NewID("User", "123")

	t.Run("NewKey_WithID", func(t *testing.T) {
		key := core.NewKey(id)
		assert.NotNil(t, key)
		assert.Equal(t, "User/123", key.String())
		assert.Equal(t, "Key(User/123)", key.GoString())
	})

	t.Run("NewKey_WithAttr", func(t *testing.T) {
		key := core.NewKey(id, "name")
		assert.NotNil(t, key)
		assert.Equal(t, "User/123/name", key.String())
		assert.Equal(t, "Key(User/123/name)", key.GoString())
	})

	t.Run("Value Method", func(t *testing.T) {
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

			assert.Equal(t, "", key.ID())
			assert.Equal(t, "", key.Attr())

		})

		t.Run("Nil Parts", func(t *testing.T) {
			key := core.NewKey()

			assert.Nil(t, key)
		})
	})
}
