package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestNewURI(t *testing.T) {
	t.Parallel()

	t.Run("Namespace URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").URI()
		require.NoError(t, err)
		assert.Equal(t, "com.example", uri.NS().String())
		assert.Equal(t, "xdb://com.example", uri.String())
	})

	t.Run("Schema URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").
			Schema("posts").
			URI()
		require.NoError(t, err)
		assert.Equal(t, "com.example", uri.NS().String())
		assert.Equal(t, "posts", uri.Schema().String())
		assert.Equal(t, "xdb://com.example/posts", uri.String())
	})

	t.Run("Record URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").
			Schema("posts").
			ID("123").
			URI()
		require.NoError(t, err)
		assert.Equal(t, "123", uri.ID().String())
		assert.Equal(t, "xdb://com.example/posts/123", uri.String())
	})

	t.Run("Hierarchy ID URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").
			Schema("posts").
			ID("123/456").
			URI()
		require.NoError(t, err)
		assert.Equal(t, "123/456", uri.ID().String())
		assert.Equal(t, "xdb://com.example/posts/123/456", uri.String())
	})

	t.Run("Attribute URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").
			Schema("posts").
			ID("123").
			Attr("name").
			URI()
		require.NoError(t, err)
		assert.Equal(t, "name", uri.Attr().String())
		assert.Equal(t, "xdb://com.example/posts/123#name", uri.String())
	})

	t.Run("Nested Attribute URI", func(t *testing.T) {
		uri, err := core.New().NS("com.example").
			Schema("posts").
			ID("123").
			Attr("profile.name").
			URI()
		require.NoError(t, err)
		assert.Equal(t, "profile.name", uri.Attr().String())
		assert.Equal(t, "xdb://com.example/posts/123#profile.name", uri.String())
	})
}
