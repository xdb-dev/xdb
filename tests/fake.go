package tests

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

// FakePost creates a fake post record with the given ID.
func FakePost(id string) *core.Record {
	return core.NewRecord("com.example", "posts", id).
		Set("title", fmt.Sprintf("Post %s", id)).
		Set("content", fmt.Sprintf("Content for %s", id)).
		Set("rating", 4.5).
		Set("active", true)
}
