package main

import (
	"context"
	"fmt"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/types"
)

// TupleAPIExample demonstrates how to use the Tuple API in XDB.
func TupleAPIExample() {
	// create tuples
	tuples := []*types.Tuple{
		types.NewKey("User", "123", "name").Value("John Doe"),
		types.NewKey("User", "123", "age").Value(25),
		types.NewKey("User", "123", "email").Value("john.doe@example.com"),
	}

	// create a store
	store := xdbmemory.New()

	// put tuples
	err := store.PutTuples(context.Background(), tuples)
	if err != nil {
		panic(err)
	}

	// get tuples
	keys := []*types.Key{
		types.NewKey("User", "123", "name"),
		types.NewKey("User", "123", "age"),
		types.NewKey("User", "123", "email"),
	}

	tuples, _, err = store.GetTuples(context.Background(), keys)
	if err != nil {
		panic(err)
	}

	for _, tuple := range tuples {
		fmt.Println("Key:", tuple.Key())
		fmt.Println("Value:", tuple.Value())
	}

	// delete tuples
	err = store.DeleteTuples(context.Background(), []*types.Key{
		types.NewKey("User", "123", "age"),
	})
	if err != nil {
		panic(err)
	}
}
