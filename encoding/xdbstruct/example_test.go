package xdbstruct_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

type User struct {
	ID    string `xdb:"id,primary_key"`
	Name  string `xdb:"name"`
	Email string `xdb:"email"`
}

type Account struct {
	ns      string
	schema  string
	id      string
	Name    string   `xdb:"name"`
	Tags    []string `xdb:"tags"`
	Score   int      `xdb:"score"`
	Address Address  `xdb:"address"`
}

func (a *Account) GetID() string      { return a.id }
func (a *Account) SetID(id string)    { a.id = id }
func (a *Account) GetNS() string      { return a.ns }
func (a *Account) GetSchema() string  { return a.schema }
func (a *Account) SetNS(ns string)    { a.ns = ns }
func (a *Account) SetSchema(s string) { a.schema = s }

// Address represents a postal address used in examples.
type Address struct {
	Street   string   `xdb:"street"`
	City     string   `xdb:"city"`
	Location Location `xdb:"location"`
}

type Location struct {
	Lat float64 `xdb:"lat"`
	Lon float64 `xdb:"lon"`
}

func ExampleEncoder_ToRecord() {
	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "xdb",
		NS:     "com.example",
		Schema: "users",
	})
	record, err := encoder.ToRecord(&User{
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

func ExampleEncoder_ToRecord_nestedStruct() {
	encoder := xdbstruct.NewEncoder(xdbstruct.Options{Tag: "xdb"})
	record, err := encoder.ToRecord(&Account{
		ns:     "com.example",
		schema: "users",
		id:     "123",
		Name:   "John Doe",
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
			Location: Location{
				Lat: 42.3601,
				Lon: -71.0589,
			},
		},
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("ID:", record.ID().String())
	fmt.Println("Name:", record.Get("name").ToString())
	fmt.Println("Street:", record.Get("address.street").ToString())
	fmt.Println("City:", record.Get("address.city").ToString())
	fmt.Println("Lat:", record.Get("address.location.lat").ToFloat())
	fmt.Println("Lon:", record.Get("address.location.lon").ToFloat())

	// Output:
	// ID: 123
	// Name: John Doe
	// Street: 123 Main St
	// City: Boston
	// Lat: 42.3601
	// Lon: -71.0589
}

func ExampleDecoder_FromRecord() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	if err := decoder.FromRecord(record, &user); err != nil {
		panic(err)
	}

	fmt.Println("ID:", user.ID)
	fmt.Println("Name:", user.Name)
	fmt.Println("Email:", user.Email)

	// Output:
	// ID: 123
	// Name: John Doe
	// Email: john@example.com
}

func ExampleDecoder_FromRecord_nestedStruct() {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.city", "Boston")

	decoder := xdbstruct.NewDefaultDecoder()

	var account Account
	if err := decoder.FromRecord(record, &account); err != nil {
		panic(err)
	}

	fmt.Println("ID:", account.GetID())
	fmt.Println("Name:", account.Name)
	fmt.Println("Street:", account.Address.Street)
	fmt.Println("City:", account.Address.City)

	// Output:
	// ID: 123
	// Name: John Doe
	// Street: 123 Main St
	// City: Boston
}

func Example_roundTrip() {
	original := Account{
		ns:     "com.example",
		schema: "users",
		id:     "user-789",
		Name:   "Alice",
		Tags:   []string{"admin", "developer"},
		Score:  100,
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{Tag: "xdb"})
	record, err := encoder.ToRecord(&original)
	if err != nil {
		panic(err)
	}

	decoder := xdbstruct.NewDefaultDecoder()
	var decoded Account
	if err := decoder.FromRecord(record, &decoded); err != nil {
		panic(err)
	}

	fmt.Println("ID:", decoded.GetID())
	fmt.Println("Name:", decoded.Name)
	fmt.Println("Tags:", decoded.Tags)
	fmt.Println("Score:", decoded.Score)

	// Output:
	// ID: user-789
	// Name: Alice
	// Tags: [admin developer]
	// Score: 100
}
