package core_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestRecord(t *testing.T) {
	record := core.NewRecord("com.example", "posts", "123")

	record.Set("title", "Hello, World!")
	record.Set("content", "This is my first post")
	record.Set("created_at", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	record.Set("author_id", "123")
	record.Set("tags", []string{"xdb", "golang"})

	t.Run("Getters", func(t *testing.T) {
		assert.Equal(t, "com.example", record.NS().String())
		assert.Equal(t, "posts", record.Schema())
		assert.Equal(t, "123", record.ID().String())
		assert.Equal(t, "xdb://com.example.posts/123", record.URI().String())
		assert.Equal(t, "Record(xdb://com.example.posts/123)", record.GoString())
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
			assert.Equal(t, "com.example", tuple.NS().String())
			assert.Equal(t, "posts", tuple.Schema())
			assert.Equal(t, "123", tuple.ID().String())
		}
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Empty Record", func(t *testing.T) {
			emptyRecord := core.NewRecord("com.example", "posts", "123")
			assert.True(t, emptyRecord.IsEmpty())
			assert.Equal(t, 0, len(emptyRecord.Tuples()))
			assert.Nil(t, emptyRecord.Get("nonexistent"))
		})

		t.Run("Attribute Overwrite", func(t *testing.T) {
			record := core.NewRecord("com.example", "posts", "123")
			record.Set("name", "John")
			record.Set("name", "Jane") // Overwrite

			assert.Equal(t, "Jane", record.Get("name").ToString())
			assert.Equal(t, 1, len(record.Tuples())) // Still only one tuple
		})

		t.Run("Nil Values", func(t *testing.T) {
			record := core.NewRecord("com.example", "users", "123")
			record.Set("name", nil)

			tuple := record.Get("name")
			assert.NotNil(t, tuple)
			assert.True(t, tuple.Value().IsNil())
		})

		t.Run("Nested Attributes", func(t *testing.T) {
			record := core.NewRecord("com.example", "users", "123")
			record.Set("profile.name", "John")
			record.Set("profile.age", 25)

			assert.Equal(t, "John", record.Get("profile.name").ToString())
			assert.Equal(t, int64(25), record.Get("profile.age").ToInt())
			assert.Equal(t, 2, len(record.Tuples()))
		})

		t.Run("Concurrent Access", func(t *testing.T) {
			record := core.NewRecord("com.example", "users", "123")

			// Test concurrent writes
			done := make(chan bool, 10)
			for i := 0; i < 10; i++ {
				go func(i int) {
					record.Set(fmt.Sprintf("attr%d", i), i)
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				<-done
			}

			assert.Equal(t, 10, len(record.Tuples()))

			// Test concurrent reads
			for i := 0; i < 10; i++ {
				go func(i int) {
					val := record.Get(fmt.Sprintf("attr%d", i))
					assert.Equal(t, int64(i), val.ToInt())
					done <- true
				}(i)
			}

			// Wait for all reads to complete
			for i := 0; i < 10; i++ {
				<-done
			}
		})

		t.Run("Get Non-existent Attribute", func(t *testing.T) {
			record := core.NewRecord("com.example", "users", "123")
			record.Set("name", "John")

			assert.Nil(t, record.Get("nonexistent"))
			assert.Nil(t, record.Get("profile.name"))
		})

		t.Run("Method Chaining", func(t *testing.T) {
			record := core.NewRecord("com.example", "users", "123").
				Set("name", "John").
				Set("age", 25).
				Set("email", "john@example.com")

			assert.Equal(t, 3, len(record.Tuples()))
			assert.Equal(t, "John", record.Get("name").ToString())
			assert.Equal(t, int64(25), record.Get("age").ToInt())
			assert.Equal(t, "john@example.com", record.Get("email").ToString())
		})
	})
}
