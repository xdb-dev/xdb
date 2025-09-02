package types_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/types"
)

func ExampleKey() {
	// This is an example of a key that references a record.
	key := types.NewKey("User", "123")
	fmt.Println(key)

	// This is an example of a key that references an attribute.
	key = types.NewKey("User", "123", "name")
	fmt.Println(key)

	// This is an example of a key that references an edge.
	key = types.NewKey("User", "123", "follows", "Post", "123")
	fmt.Println(key)

	// Output:
	// Key(User/123)
	// Key(User/123/name)
	// Key(User/123/follows/Post/123)
}

func ExampleTuple_key_value() {
	// Arbitrary key and value.
	kv := types.NewTuple("name", "John Doe")
	fmt.Println(kv.Key())
	fmt.Println(kv.Value())

	// Output:
	// name
	// John Doe
}

func ExampleTuple_object_attributes() {
	tuples := []*types.Tuple{
		types.NewTuple(types.NewKey("User", "123", "name"), "John Doe"),
		types.NewTuple(types.NewKey("User", "123", "age"), 25),
		types.NewTuple(types.NewKey("User", "123", "interests"), []string{"reading", "traveling", "coding"}),
	}

	_ = tuples
}

func ExampleRecord() {
	// This is an example of creating a record.
	record := types.NewRecord(types.NewKey("User", "123")).
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
