package main

import (
	"fmt"

	"github.com/xdb-dev/xdb/types"
)

func main() {
	// create a tuple
	tuple := types.NewTuple(
		types.NewKey("User", "123", "name"),
		"John Doe",
	)

	fmt.Println("Key:", tuple.Key())
	fmt.Println("Kind:", tuple.Kind())
	fmt.Println("ID:", tuple.ID())
	fmt.Println("Name:", tuple.Name())
	fmt.Println("Value:", tuple.Value())
}
