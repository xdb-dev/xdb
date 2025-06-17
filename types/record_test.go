package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

func TestRecord(t *testing.T) {
	record := types.NewRecord("Post", "123")

	record.Set("title", "Hello, World!")
	record.Set("content", "This is my first post")
	record.Set("created_at", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	record.Set("author_id", "123")
	record.Set("tags", []string{"xdb", "golang"})

	t.Run("Getters", func(t *testing.T) {
		assert.Equal(t, "Post", record.Kind())
		assert.Equal(t, "123", record.ID())
		assert.Equal(t, "Record(Post, 123)", record.String())
		assert.Equal(t, "Key(Post/123)", record.Key().String())
		assert.False(t, record.IsEmpty())
	})

	t.Run("Get Attributes", func(t *testing.T) {
		title := record.Get("title").ToString()
		assert.Equal(t, "Hello, World!", title)

		content := record.Get("content").ToString()
		assert.Equal(t, "This is my first post", content)

		createdAt := record.Get("created_at").ToTime()
		assert.Equal(t, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), createdAt)

		authorID := record.Get("author_id").ToString()
		assert.Equal(t, "123", authorID)

		tags := record.Get("tags").ToStringArray()
		assert.Equal(t, []string{"xdb", "golang"}, tags)
	})

	t.Run("Tuples", func(t *testing.T) {
		tuples := record.Tuples()
		assert.Equal(t, 5, len(tuples))

		for _, tuple := range tuples {
			assert.NotNil(t, tuple)
			assert.Equal(t, "Post", tuple.Kind())
			assert.Equal(t, "123", tuple.ID())
		}
	})
}
