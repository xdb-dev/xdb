package core_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

func ExampleTuple() {
	id := core.NewID("123")
	tuple := core.NewTuple("com.example.users", id, "name", "John Doe")

	fmt.Println(tuple.ID())
	fmt.Println(tuple.Attr())
	fmt.Println(tuple.Value())
	fmt.Println(tuple.URI())

	// Output:
	// 123
	// name
	// John Doe
	// xdb://com.example.users/123#name
}

func ExampleRecord() {
	// This is an example of creating a record.
	record := core.NewRecord("com.example.users", "123").
		Set("name", "John Doe").
		Set("age", 25).
		Set("interests", []string{"reading", "traveling", "coding"})

	// Reading attributes
	fmt.Println(record.Get("name").ToString())
	fmt.Println(record.Get("age").ToInt())
	fmt.Println(record.Get("interests").ToStringArray())
	fmt.Println(record.URI())

	// Output:
	// John Doe
	// 25
	// [reading traveling coding]
	// xdb://com.example.users/123
}
