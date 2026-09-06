package xdbjson_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
)

func ExampleUnmarshal() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	data, err := xdbjson.Unmarshal(record)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"_id":"123","email":"john@example.com","name":"John Doe"}
}

func ExampleUnmarshal_withMetadata() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	data, err := xdbjson.Unmarshal(record, xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"_id":"123","_ns":"com.example","_schema":"users","name":"John Doe"}
}

func ExampleUnmarshal_nestedStruct() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.city", "Boston").
		Set("address.location.lat", 42.3601).
		Set("address.location.lon", -71.0589)

	data, err := xdbjson.Unmarshal(record, xdbjson.WithIndent("", "  "))
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {
	//   "_id": "123",
	//   "address": {
	//     "city": "Boston",
	//     "location": {
	//       "lat": 42.3601,
	//       "lon": -71.0589
	//     },
	//     "street": "123 Main St"
	//   },
	//   "name": "John Doe"
	// }
}

func ExampleMarshal() {
	data := []byte(`{"_id":"123","name":"John Doe","age":30}`)

	record, err := xdbjson.Marshal(data, xdbjson.WithNS("com.example"), xdbjson.WithSchema("users"))
	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", record.URI())
	fmt.Println("Name:", vStr(record.Get("name").Value()))

	// Whole-looking JSON numbers decode as int64 without a schema definition.
	fmt.Println("Age:", vInt(record.Get("age").Value()))

	// Output:
	// URI: xdb://com.example/users/123
	// Name: John Doe
	// Age: 30
}

func ExampleMarshal_withMetadata() {
	data := []byte(`{"_id":"123","_ns":"com.example","_schema":"users","name":"John Doe"}`)

	record, err := xdbjson.Marshal(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("NS:", record.URI().NS())
	fmt.Println("Schema:", record.URI().Schema())
	fmt.Println("ID:", record.URI().ID())
	fmt.Println("Name:", vStr(record.Get("name").Value()))

	// Output:
	// NS: com.example
	// Schema: users
	// ID: 123
	// Name: John Doe
}

func ExampleMarshal_customFields() {
	data := []byte(`{"userId":"123","namespace":"com.example","type":"users","name":"John"}`)

	decOpts := []xdbjson.Option{
		xdbjson.WithIDField("userId"),
		xdbjson.WithNSField("namespace"),
		xdbjson.WithSchemaField("type"),
	}

	record, err := xdbjson.Marshal(data, decOpts...)
	if err != nil {
		panic(err)
	}

	fmt.Println("ID:", record.URI().ID())
	fmt.Println("NS:", record.URI().NS())
	fmt.Println("Schema:", record.URI().Schema())

	// Output:
	// ID: 123
	// NS: com.example
	// Schema: users
}

func ExampleMarshal_nestedObject() {
	data := []byte(`{
		"_id": "123",
		"name": "John Doe",
		"address": {
			"street": "123 Main St",
			"city": "Boston"
		}
	}`)

	record, err := xdbjson.Marshal(data, xdbjson.WithNS("com.example"), xdbjson.WithSchema("users"))
	if err != nil {
		panic(err)
	}

	fmt.Println("Name:", vStr(record.Get("name").Value()))
	fmt.Println("Street:", vStr(record.Get("address.street").Value()))
	fmt.Println("City:", vStr(record.Get("address.city").Value()))

	// Output:
	// Name: John Doe
	// Street: 123 Main St
	// City: Boston
}

func ExampleMarshalInto() {
	// The caller owns the identity, so the document's metadata is ignored.
	record := core.NewRecord("com.example", "users", "123")

	data := []byte(`{"_id":"ignored","name":"John Doe","age":30}`)

	if err := xdbjson.MarshalInto(data, record); err != nil {
		panic(err)
	}

	fmt.Println("URI:", record.URI())
	fmt.Println("Name:", vStr(record.Get("name").Value()))

	// Output:
	// URI: xdb://com.example/users/123
	// Name: John Doe
}

func Example_roundTrip() {
	original := core.NewRecord("com.example", "users", "user-789").
		Set("name", "Alice").
		Set("tags", []string{"admin", "developer"}).
		Set("score", 100)

	data, err := xdbjson.Unmarshal(original, xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())
	if err != nil {
		panic(err)
	}

	decoded, err := xdbjson.Marshal(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", decoded.URI())
	fmt.Println("Name:", vStr(decoded.Get("name").Value()))

	// Whole-looking JSON numbers decode as int64 without a schema definition.
	fmt.Println("Score:", vInt(decoded.Get("score").Value()))

	// Output:
	// URI: xdb://com.example/users/user-789
	// Name: Alice
	// Score: 100
}
