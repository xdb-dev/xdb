// Package xdbstruct provides utilities for converting Go structs to XDB records and vice versa.
//
// # Overview
//
// The xdbstruct package follows GORM-like conventions for encoding and decoding Go structs:
//   - Use interfaces for record-level configuration (namespace, schema, ID)
//   - Use struct tags for field-level configuration
//   - Simple by default with zero-config for common cases
//   - Explicit configuration through Options for full control
//
// # Basic Usage
//
// Create an encoder to convert structs to records:
//
//	type User struct {
//	    ID    string `xdb:"id,primary_key"`
//	    Name  string `xdb:"name"`
//	    Email string `xdb:"email"`
//	}
//
//	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
//	    Tag:    "xdb",
//	    NS:     "com.example",
//	    Schema: "users",
//	})
//
//	record, err := encoder.Encode(&User{
//	    ID:    "123",
//	    Name:  "John Doe",
//	    Email: "john@example.com",
//	})
//	// record.URI() -> xdb://com.example/users/123
//
// Create a decoder to populate structs from records:
//
//	decoder := xdbstruct.NewDecoder(xdbstruct.Options{Tag: "xdb"})
//
//	var user User
//	err := decoder.Decode(record, &user)
//	// user is now populated with data from record
//
// # Struct Tags
//
// Fields are mapped using struct tags. The tag format is:
//
//	`xdb:"field_name[,option1,option2,...]"`
//
// Available options:
//   - primary_key: Mark field as the record's primary key (ID)
//   - -: Skip field during encoding/decoding
//
// Example:
//
//	type User struct {
//	    ID       string `xdb:"id,primary_key"`
//	    Name     string `xdb:"name"`
//	    Internal string `xdb:"-"`  // skipped
//	}
//
// # Interfaces for Record-Level Configuration
//
// Structs can implement interfaces to control record-level metadata.
// These interfaces provide the highest priority configuration.
//
// For encoding:
//   - NSGetter: Provide custom namespace via NS() string
//   - SchemaGetter: Provide custom schema via Schema() string
//   - IDGetter: Provide custom ID via ID() string
//
// For decoding:
//   - NSSetter: Receive namespace via SetNS(ns string)
//   - SchemaSetter: Receive schema via SetSchema(schema string)
//   - IDSetter: Receive ID via SetID(id string)
//
// Example with interfaces:
//
//	type User struct {
//	    ID   string `xdb:"id,primary_key"`
//	    Name string `xdb:"name"`
//	}
//
//	func (u User) NS() string     { return "com.example" }
//	func (u User) Schema() string { return "users" }
//
//	encoder := xdbstruct.NewDefaultEncoder()
//	record, _ := encoder.Encode(&user)
//	// record.URI() -> xdb://com.example/users/123
//
// Symmetric setters for decoding:
//
//	type User struct {
//	    id     string
//	    ns     string
//	    schema string
//	    Name   string `xdb:"name"`
//	}
//
//	func (u *User) SetID(id string)       { u.id = id }
//	func (u *User) SetNS(ns string)       { u.ns = ns }
//	func (u *User) SetSchema(schema string) { u.schema = schema }
//
//	decoder := xdbstruct.NewDefaultDecoder()
//	var user User
//	decoder.Decode(record, &user)
//	// user.id, user.ns, user.schema are now populated
//
// # Configuration Priority
//
// For encoding (highest to lowest priority):
//  1. Struct interfaces (NSGetter, SchemaGetter, IDGetter)
//  2. Struct tags (field-level only)
//  3. Encoder Options (defaults)
//
// For ID specifically:
//  1. IDGetter interface
//  2. Field with primary_key tag
//  3. Error (ID is required)
//
// # Nested Structs
//
// Nested structs are flattened using dot-separated paths:
//
//	type Address struct {
//	    Street string `xdb:"street"`
//	    City   string `xdb:"city"`
//	}
//
//	type User struct {
//	    ID      string  `xdb:"id,primary_key"`
//	    Address Address `xdb:"address"`
//	}
//
// This encodes to tuples:
//   - id: "123"
//   - address.street: "123 Main St"
//   - address.city: "Boston"
//
// Deeply nested structs work the same way:
//
//	address.location.lat: 42.3601
//	address.location.lon: -71.0589
//
// # Arrays and Slices
//
// Arrays and slices of basic types are encoded directly:
//
//	type User struct {
//	    Tags []string `xdb:"tags"`
//	}
//	// tags: ["go", "rust", "python"]
//
// Byte slices have special handling:
//
//	type User struct {
//	    Data []byte `xdb:"data"`
//	}
//	// data: stored as []byte
//
// # Custom Marshalers
//
// Types implementing json.Marshaler or encoding.BinaryMarshaler are encoded as []byte:
//
//	type User struct {
//	    Metadata json.RawMessage `xdb:"metadata"`
//	}
//
// Arrays of types with marshalers become [][]byte:
//
//	type Post struct {
//	    Timestamps []time.Time `xdb:"timestamps"`
//	}
//	// timestamps: [binary, binary, binary] as [][]byte
//
// # Default Encoder and Decoder
//
// For simple cases, use the default constructors:
//
//	encoder := xdbstruct.NewDefaultEncoder()  // Tag: "xdb"
//	decoder := xdbstruct.NewDefaultDecoder()  // Tag: "xdb"
//
// # Custom Options
//
// For full control, create encoders/decoders with custom options:
//
//	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
//	    Tag:    "db",           // Use "db" tag instead of "xdb"
//	    NS:     "com.myapp",    // Default namespace
//	    Schema: "users",        // Default schema
//	})
//
// # Error Handling
//
// Encoding errors:
//   - Input is not a struct or pointer to struct
//   - Cannot determine ID (no IDGetter interface, no primary_key tag)
//   - ID is empty
//   - Namespace is empty
//   - Schema is empty
//   - Type conversion fails
//   - Custom marshaler fails
//
// Decoding errors:
//   - Input is not a pointer to struct
//   - Type mismatch between record value and struct field
//   - Custom unmarshaler fails
//
// The encoder/decoder do NOT validate business rules or missing fields.
// Zero values are encoded/decoded as-is.
package xdbstruct
