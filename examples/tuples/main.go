package main

import (
	"context"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
)

func main() {
	ctx := context.Background()
	repo := "com.example/posts"
	id := core.NewID(gofakeit.UUID())
	authorID := core.NewID(gofakeit.UUID())

	post := []*core.Tuple{
		core.NewTuple(repo, id, "title", "Hello, World!"),
		core.NewTuple(repo, id, "content", "This is my first post"),
		core.NewTuple(repo, id, "author_id", authorID),
		core.NewTuple(repo, id, "created_at", time.Now()),
		core.NewTuple(repo, id, "tags", []string{"xdb", "golang"}),
	}

	store := xdbmemory.New()
	err := store.PutTuples(ctx, post)
	if err != nil {
		panic(err)
	}

	uris := []*core.URI{
		core.NewURI(repo, id, "title"),
		core.NewURI(repo, id, "content"),
		core.NewURI(repo, id, "author_id"),
		core.NewURI(repo, id, "created_at"),
		core.NewURI(repo, id, "tags"),
	}

	tuples, _, err := store.GetTuples(ctx, uris)
	if err != nil {
		panic(err)
	}

	for _, tuple := range tuples {
		fmt.Println(tuple.URI().String(), tuple.Value().String())
	}

	err = store.DeleteTuples(ctx, uris)
	if err != nil {
		panic(err)
	}
}
