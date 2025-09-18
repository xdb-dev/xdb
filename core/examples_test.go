package core_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

func ExampleKey() {
	// This is an example of a key that references a record.
	key := core.NewKey("User", "123")
	fmt.Println(key)

	// This is an example of a key that references a tuple.
	key = core.NewKey("User", "123", "name")
	fmt.Println(key)

	// This is an example of a key that references an edge.
	key = core.NewKey("User", "123", "follows", "Post", "123")
	fmt.Println(key)

	// This is an example of a multi-tenant key.
	key = core.NewKey("example.com", "User", "123")
	fmt.Println(key)

	// Output:
	// User/123
	// User/123/name
	// User/123/follows/Post/123
	// example.com/User/123
}

func ExampleTuple() {
	// This is an example of a tuple.
	// Read this as:
	// - "User" is the kind
	// - "123" is the ID
	// - "name" is the attribute
	// - "John Doe" is the value
	tuple := core.NewTuple("User", "123", "name", "John Doe")

	fmt.Println(tuple.Kind())
	fmt.Println(tuple.ID())
	fmt.Println(tuple.Attr())
	fmt.Println(tuple.Value())

	// Output:
	// User
	// 123
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
