package main

import (
	"context"
	"fmt"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/core"
)

// TupleAPIExample demonstrates how to use the Tuple API in XDB.
func TupleAPIExample() {
	// create tuples
	tuples := []*core.Tuple{
		core.NewTuple("User", "123", "name", "John Doe"),
		core.NewTuple("User", "123", "age", 25),
		core.NewTuple("User", "123", "email", "john.doe@example.com"),
	}

	// create a store
	store := xdbmemory.New()

	// put tuples
	err := store.PutTuples(context.Background(), tuples)
	if err != nil {
		panic(err)
	}

	// get tuples
	keys := []*core.Key{
		core.NewKey("User", "123", "name"),
		core.NewKey("User", "123", "age"),
		core.NewKey("User", "123", "email"),
	}

	tuples, _, err = store.GetTuples(context.Background(), keys)
	if err != nil {
		panic(err)
	}

	for _, tuple := range tuples {
		fmt.Println("Kind:", tuple.Kind())
		fmt.Println("ID:", tuple.ID())
		fmt.Println("Attr:", tuple.Attr())
		fmt.Println("Value:", tuple.Value())
	}

	// delete tuples
	err = store.DeleteTuples(context.Background(), []*core.Key{
		core.NewKey("User", "123", "age"),
	})
	if err != nil {
		panic(err)
	}
}
