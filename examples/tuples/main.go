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
	id := core.NewID(gofakeit.UUID())
	authorID := core.NewID(gofakeit.UUID())

	post := []*core.Tuple{
		core.NewTuple(id, "title", "Hello, World!"),
		core.NewTuple(id, "content", "This is my first post"),
		core.NewTuple(id, "author_id", authorID),
		core.NewTuple(id, "created_at", time.Now()),
		core.NewTuple(id, "tags", []string{"xdb", "golang"}),
	}

	store := xdbmemory.New()
	err := store.PutTuples(ctx, post)
	if err != nil {
		panic(err)
	}

	keys := []*core.Key{
		core.NewKey(id, "title"),
		core.NewKey(id, "content"),
		core.NewKey(id, "author_id"),
		core.NewKey(id, "created_at"),
		core.NewKey(id, "tags"),
	}

	tuples, _, err := store.GetTuples(ctx, keys)
	if err != nil {
		panic(err)
	}

	for _, tuple := range tuples {
		fmt.Println(tuple.ID(), tuple.Attr(), tuple.Value().String())
	}

	err = store.DeleteTuples(ctx, keys)
	if err != nil {
		panic(err)
	}
}
