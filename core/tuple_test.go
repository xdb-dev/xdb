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
}
