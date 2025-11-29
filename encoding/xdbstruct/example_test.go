package xdbstruct_test

import (
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

func ExampleEncoder_Encode() {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Email string `xdb:"email"`
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "xdb",
		NS:     "com.example",
		Schema: "users",
	})
	record, err := encoder.Encode(&User{
		ID:    "123",
		Name:  "John Doe",
		Email: "john@example.com",
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", record.URI())
	fmt.Println("ID:", record.ID().String())
	fmt.Println("Name:", record.Get("name").ToString())
	fmt.Println("Email:", record.Get("email").ToString())

	// Output:
	// URI: xdb://com.example/users/123
	// ID: 123
	// Name: John Doe
	// Email: john@example.com
}

type UserWithMethods struct {
	ID    string `xdb:"id,primary_key"`
	Name  string `xdb:"name"`
	Email string `xdb:"email"`
}

func (u *UserWithMethods) GetNS() string     { return "com.example" }
func (u *UserWithMethods) GetSchema() string { return "users" }

func ExampleEncoder_Encode_withInterfaces() {
	user := UserWithMethods{
		ID:    "123",
		Name:  "John Doe",
		Email: "john@example.com",
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{Tag: "xdb"})
	record, err := encoder.Encode(&user)

	if err != nil {
		panic(err)
	}

	fmt.Println("URI:", record.URI())

	// Output:
	// URI: xdb://com.example/users/123
}

func ExampleEncoder_Encode_nestedStruct() {
	type Address struct {
		Street string `xdb:"street"`
		City   string `xdb:"city"`
	}

	type User struct {
		ID      string  `xdb:"id,primary_key"`
		Name    string  `xdb:"name"`
		Address Address `xdb:"address"`
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "xdb",
		NS:     "com.example",
		Schema: "users",
	})
	record, err := encoder.Encode(&User{
		ID:   "123",
		Name: "John Doe",
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("ID:", record.ID().String())
	fmt.Println("Name:", record.Get("name").ToString())
	fmt.Println("Street:", record.Get("address.street").ToString())
	fmt.Println("City:", record.Get("address.city").ToString())

	// Output:
	// ID: 123
	// Name: John Doe
	// Street: 123 Main St
	// City: Boston
}

func ExampleEncoder_Encode_customMarshaler() {
	type User struct {
		ID       string          `xdb:"id,primary_key"`
		Name     string          `xdb:"name"`
		Metadata json.RawMessage `xdb:"metadata"`
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "xdb",
		NS:     "com.example",
		Schema: "users",
	})
	record, err := encoder.Encode(&User{
		ID:       "123",
		Name:     "John Doe",
		Metadata: json.RawMessage(`{"role":"admin"}`),
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("ID:", record.ID().String())
	fmt.Println("Name:", record.Get("name").ToString())
	fmt.Println("Metadata:", record.Get("metadata").ToString())

	// Output:
	// ID: 123
	// Name: John Doe
	// Metadata: {"role":"admin"}
}
