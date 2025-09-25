package core_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

func ExampleTuple() {
	id := core.NewID("User", "123")
	tuple := core.NewTuple(id, "name", "John Doe")

	fmt.Println(tuple.ID())
	fmt.Println(tuple.Attr())
	fmt.Println(tuple.Value())

	// Output:
	// User/123
	// name
	// John Doe
}

func ExampleRecord() {
	// This is an example of creating a record.
	record := core.NewRecord("User", "123").
		Set("name", "John Doe").
		Set("age", 25).
		Set("interests", []string{"reading", "traveling", "coding"})

	// Reading attributes
	fmt.Println(record.Get("name").ToString())
	fmt.Println(record.Get("age").ToInt())
	fmt.Println(record.Get("interests").ToStringArray())

	// Output:
	// John Doe
	// 25
	// [reading traveling coding]
}
