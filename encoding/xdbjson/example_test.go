package xdbjson_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
)

func ExampleEncoder_FromRecord() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	encoder := xdbjson.NewDefaultEncoder()
	data, err := encoder.FromRecord(record)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"_id":"123","email":"john@example.com","name":"John Doe"}
}

func ExampleEncoder_FromRecord_withMetadata() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	encoder := xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})
	data, err := encoder.FromRecord(record)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"_id":"123","_ns":"com.example","_schema":"users","name":"John Doe"}
}

func ExampleEncoder_FromRecord_nestedStruct() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.city", "Boston").
		Set("address.location.lat", 42.3601).
		Set("address.location.lon", -71.0589)

	encoder := xdbjson.NewDefaultEncoder()
	data, err := encoder.FromRecordIndent(record, "", "  ")
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

func ExampleDecoder_ToRecord() {
	data := []byte(`{"_id":"123","name":"John Doe","age":30}`)

	decoder := xdbjson.NewDefaultDecoder("com.example", "users")
	record, err := decoder.ToRecord(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", record.URI())
	fmt.Println("Name:", record.Get("name").Value().ToString())
	fmt.Println("Age:", record.Get("age").Value().ToInt())

	// Output:
	// URI: xdb://com.example/users/123
	// Name: John Doe
	// Age: 30
}

func ExampleDecoder_ToRecord_withMetadata() {
	data := []byte(`{"_id":"123","_ns":"com.example","_schema":"users","name":"John Doe"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{})
	record, err := decoder.ToRecord(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("NS:", record.NS().String())
	fmt.Println("Schema:", record.Schema().String())
	fmt.Println("ID:", record.ID().String())
	fmt.Println("Name:", record.Get("name").Value().ToString())

	// Output:
	// NS: com.example
	// Schema: users
	// ID: 123
	// Name: John Doe
}

func ExampleDecoder_ToRecord_customFields() {
	data := []byte(`{"userId":"123","namespace":"com.example","type":"users","name":"John"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{
		IDField:     "userId",
		NSField:     "namespace",
		SchemaField: "type",
	})

	record, err := decoder.ToRecord(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("ID:", record.ID().String())
	fmt.Println("NS:", record.NS().String())
	fmt.Println("Schema:", record.Schema().String())

	// Output:
	// ID: 123
	// NS: com.example
	// Schema: users
}

func ExampleDecoder_ToRecord_nestedObject() {
	data := []byte(`{
		"_id": "123",
		"name": "John Doe",
		"address": {
			"street": "123 Main St",
			"city": "Boston"
		}
	}`)

	decoder := xdbjson.NewDefaultDecoder("com.example", "users")
	record, err := decoder.ToRecord(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("Name:", record.Get("name").Value().ToString())
	fmt.Println("Street:", record.Get("address.street").Value().ToString())
	fmt.Println("City:", record.Get("address.city").Value().ToString())

	// Output:
	// Name: John Doe
	// Street: 123 Main St
	// City: Boston
}

func Example_roundTrip() {
	original := core.NewRecord("com.example", "users", "user-789").
		Set("name", "Alice").
		Set("tags", []string{"admin", "developer"}).
		Set("score", 100)

	encoder := xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})
	data, err := encoder.FromRecord(original)
	if err != nil {
		panic(err)
	}

	decoder := xdbjson.NewDecoder(xdbjson.Options{})
	decoded, err := decoder.ToRecord(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", decoded.URI())
	fmt.Println("Name:", decoded.Get("name").Value().ToString())
	fmt.Println("Score:", decoded.Get("score").Value().ToInt())

	// Output:
	// URI: xdb://com.example/users/user-789
	// Name: Alice
	// Score: 100
}
