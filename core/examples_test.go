package core_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

func ExampleTuple() {
	tuple := core.New().NS("com.example").
		Schema("users").
		ID("123").
		MustTuple("name", "John Doe")

	fmt.Println(tuple.NS())
	fmt.Println(tuple.Schema())
	fmt.Println(tuple.ID())
	fmt.Println(tuple.Attr())
	fmt.Println(tuple.Value().ToString())
	fmt.Println(tuple.URI())

	// Output:
	// com.example
	// users
	// 123
	// name
	// John Doe
	// xdb://com.example.users/123#name

}

func ExampleRecord() {
	record := core.New().NS("com.example").
		Schema("users").
		ID("123").
		MustRecord()

	record.Set("name", "John Doe").
		Set("age", 25).
		Set("bio.interests", []string{"reading", "traveling", "coding"})

	// Reading attributes
	fmt.Println(record.Get("name").ToString())
	fmt.Println(record.Get("age").ToInt())
	fmt.Println(record.Get("bio.interests").ToStringArray())
	fmt.Println(record.URI())

	// Output:
	// John Doe
	// 25
	// [reading traveling coding]
	// xdb://com.example.users/123
}
