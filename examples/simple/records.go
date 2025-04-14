package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/types"
)

func RecordAPIExample() {
	// create a record
	record := types.NewRecord("Post", "123").
		Set("title", "Hello, World!").
		Set("content", "This is my first post").
		Set("created_at", time.Now()).
		AddEdge("author", types.NewKey("User", "123")).
		AddEdge("tags", types.NewKey("Tag", "xdb")).
		AddEdge("tags", types.NewKey("Tag", "golang"))

	// create a store
	store := xdbmemory.New()
	ctx := context.Background()

	// put the record
	err := store.PutRecords(ctx, []*types.Record{record})
	if err != nil {
		panic(err)
	}

	// get the record
	records, _, err := store.GetRecords(ctx, []*types.Key{record.Key()})
	if err != nil {
		panic(err)
	}

	for _, record := range records {
		title := record.Get("title").ToString()
		content := record.Get("content").ToString()
		createdAt := record.Get("created_at").ToTime()

		author := record.GetEdge("author")
		tags := record.GetEdges("tags")

		fmt.Printf("Title: %s\n", title)
		fmt.Printf("Content: %s\n", content)
		fmt.Printf("Created At: %s\n", createdAt)
		fmt.Printf("Author: %s\n", author)
		fmt.Printf("Tags: %v\n", tags)
	}

	// delete the record
	err = store.DeleteRecords(ctx, []*types.Key{record.Key()})
	if err != nil {
		panic(err)
	}
}
