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
	record.Set("created_at", time.Now())
	record.Set("author_id", "123")
	record.Set("tags", []string{"xdb", "golang"})

	assert.Equal(t, "Post", record.Kind())
	assert.Equal(t, "123", record.ID())
	assert.False(t, record.IsEmpty())

	title := record.Get("title").ToString()
	assert.Equal(t, "Hello, World!", title)

	content := record.Get("content").ToString()
	assert.Equal(t, "This is my first post", content)

	createdAt := record.Get("created_at").ToTime()
	assert.NotNil(t, createdAt)

	authorID := record.Get("author_id").ToString()
	assert.Equal(t, "123", authorID)

	tags := record.Get("tags").ToString()
	assert.Equal(t, "xdb,golang", tags)
}
