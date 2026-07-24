package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestDescribeField_IndexedUnique(t *testing.T) {
	t.Run("emits indexed and unique when set", func(t *testing.T) {
		entry := describeField("email", schema.Field{
			Type:   core.TypeString,
			Unique: true,
		})
		assert.Equal(t, true, entry["unique"])
		_, hasIndexed := entry["indexed"]
		assert.False(t, hasIndexed, "indexed omitted when false")
	})

	t.Run("emits indexed", func(t *testing.T) {
		entry := describeField("status", schema.Field{
			Type:    core.TypeString,
			Indexed: true,
		})
		assert.Equal(t, true, entry["indexed"])
	})

	t.Run("omits both when unset", func(t *testing.T) {
		entry := describeField("name", schema.Field{Type: core.TypeString})
		_, hasIndexed := entry["indexed"]
		_, hasUnique := entry["unique"]
		assert.False(t, hasIndexed)
		assert.False(t, hasUnique)
	})
}
