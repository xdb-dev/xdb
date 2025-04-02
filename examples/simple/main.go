package main

import (
	"context"
	"fmt"

	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/types"
)

func main() {
	// create tuples
	tuples := []*types.Tuple{
		types.NewTuple(
			types.NewKey("User", "123", "name"),
			"John Doe",
		),
		types.NewTuple(
			types.NewKey("User", "123", "age"),
			25,
		),
		types.NewTuple(
			types.NewKey("User", "123", "email"),
			"john.doe@example.com",
		),
	}

	// create a store
	store := xdbmemory.New()

	// put tuples
	err := store.PutTuples(context.Background(), tuples)
	if err != nil {
		panic(err)
	}

	// get tuples
	tuples, _, err = store.GetTuples(context.Background(), []*types.Key{
		types.NewKey("User", "123", "name"),
		types.NewKey("User", "123", "age"),
		types.NewKey("User", "123", "email"),
	})
	if err != nil {
		panic(err)
	}

	for _, tuple := range tuples {
		fmt.Println("Key:", tuple.Key())
		fmt.Println("Kind:", tuple.Kind())
		fmt.Println("ID:", tuple.ID())
		fmt.Println("Name:", tuple.Name())
		fmt.Println("Value:", tuple.Value().ToString())
	}

	// delete tuples
	err = store.DeleteTuples(context.Background(), []*types.Key{
		types.NewKey("User", "123", "age"),
	})
	if err != nil {
		panic(err)
	}
}
