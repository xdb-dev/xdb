package schema_test

import (
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func ExampleLoadFromJSON() {
	data := []byte(`{
		"name": "User",
		"description": "User record schema",
		"version": "1.0.0",
		"fields": [
			{
				"name": "name",
				"description": "User's full name",
				"type": "STRING"
			},
			{
				"name": "email",
				"type": "STRING"
			},
			{
				"name": "age",
				"type": "INTEGER"
			}
		],
		"required": ["name", "email"]
	}`)

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(s.Name)
	fmt.Println(s.Version)
	fmt.Println(len(s.Fields))
	fmt.Println(len(s.Required))

	// Output:
	// User
	// 1.0.0
	// 3
	// 2
}

func ExampleLoadFromYAML() {
	data := []byte(`
name: Product
description: Product schema
version: 2.0.0
fields:
  - name: sku
    type: STRING
  - name: name
    type: STRING
  - name: price
    type: FLOAT
required:
  - sku
  - name
`)

	s, err := schema.LoadFromYAML(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(s.Name)
	fmt.Println(s.Version)
	fmt.Println(len(s.Fields))

	// Output:
	// Product
	// 2.0.0
	// 3
}

func ExampleLoadFromJSON_withArrays() {
	data := []byte(`{
		"name": "Post",
		"fields": [
			{
				"name": "title",
				"type": "STRING"
			},
			{
				"name": "tags",
				"type": "ARRAY",
				"array_of": "STRING"
			},
			{
				"name": "scores",
				"type": "ARRAY",
				"array_of": "INTEGER"
			}
		]
	}`)

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(s.Fields[1].Name)
	fmt.Println(s.Fields[1].Type.ID())
	fmt.Println(s.Fields[1].Type.ValueType())

	// Output:
	// tags
	// TypeID(ARRAY)
	// TypeID(STRING)
}

func ExampleLoadFromJSON_withMaps() {
	data := []byte(`{
		"name": "Config",
		"fields": [
			{
				"name": "settings",
				"type": "MAP",
				"map_key": "STRING",
				"map_value": "STRING"
			},
			{
				"name": "counts",
				"type": "MAP",
				"map_key": "STRING",
				"map_value": "INTEGER"
			}
		]
	}`)

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(s.Fields[0].Name)
	fmt.Println(s.Fields[0].Type.ID())
	fmt.Println(s.Fields[0].Type.KeyType())
	fmt.Println(s.Fields[0].Type.ValueType())

	// Output:
	// settings
	// TypeID(MAP)
	// TypeID(STRING)
	// TypeID(STRING)
}

func ExampleLoadFromYAML_validation() {
	data := []byte(`
name: User
fields:
  - name: name
    type: STRING
  - name: email
    type: STRING
  - name: age
    type: INTEGER
required:
  - name
  - email
`)

	s, err := schema.LoadFromYAML(data)
	if err != nil {
		panic(err)
	}

	record := core.NewRecord("User", "123")
	record.Set("name", "John Doe")
	record.Set("email", "john@example.com")
	record.Set("age", 25)

	err = s.ValidateRecord(record)
	fmt.Println(err == nil)

	// Output:
	// true
}

func ExampleLoadFromJSON_nestedFields() {
	data := []byte(`{
		"name": "User",
		"fields": [
			{
				"name": "name",
				"type": "STRING"
			},
			{
				"name": "profile.bio",
				"type": "STRING"
			},
			{
				"name": "profile.age",
				"type": "INTEGER"
			},
			{
				"name": "settings.notifications.email",
				"type": "BOOLEAN"
			}
		]
	}`)

	s, err := schema.LoadFromJSON(data)
	if err != nil {
		panic(err)
	}

	fmt.Println(s.Fields[0].Name)
	fmt.Println(s.Fields[1].Name)
	fmt.Println(s.Fields[2].Name)
	fmt.Println(s.Fields[3].Name)

	// Output:
	// name
	// profile.bio
	// profile.age
	// settings.notifications.email
}
