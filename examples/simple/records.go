package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/core"
)

// RecordAPIExample demonstrates how to use the Record API in XDB.
func RecordAPIExample() {
	// create a record
	record := core.NewRecord("Post", "123").
		Set("title", "Hello, World!").
		Set("content", "This is my first post").
		Set("created_at", time.Now()).
		Set("author_id", "123").
		Set("tags", []string{"xdb", "golang"})

	// create a store
	store := xdbmemory.New()
	ctx := context.Background()

	// put the record
	err := store.PutRecords(ctx, []*core.Record{record})
	if err != nil {
		panic(err)
	}

	// get the record
	records, _, err := store.GetRecords(ctx, []*core.Key{record.Key()})
	if err != nil {
		panic(err)
	}

	for _, record := range records {
		title := record.Get("title").ToString()
		content := record.Get("content").ToString()
		createdAt := record.Get("created_at").ToTime()

		authorID := record.Get("author_id").ToString()
		tags := record.Get("tags").ToString()

		fmt.Printf("Title: %s\n", title)
		fmt.Printf("Content: %s\n", content)
		fmt.Printf("Created At: %s\n", createdAt)
		fmt.Printf("Author ID: %s\n", authorID)
		fmt.Printf("Tags: %v\n", tags)
	}

	// delete the record
	err = store.DeleteRecords(ctx, []*core.Key{record.Key()})
	if err != nil {
		panic(err)
	}
}
