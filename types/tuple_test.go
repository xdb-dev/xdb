package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

func TestTuple(t *testing.T) {
	t.Run("NewTuple", func(t *testing.T) {
		tuple := types.NewTuple("Post", "123", "title", "Hello, World!")

		assert.NotNil(t, tuple)
		assert.Equal(t, "title", tuple.Attr())
		assert.Equal(t, "Post/123/title", tuple.Key().String())
		assert.Equal(t, "Hello, World!", tuple.Value().ToString())
		assert.Equal(t, "Tuple(Post/123/title, Value(STRING, Hello, World!))", tuple.GoString())
	})
}
