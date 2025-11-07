// Package core provides the fundamental data structures for XDB, a tuple-based database abstraction.
//
// XDB models data using tuples, where each tuple contains:
//   - ID: A hierarchical identifier (e.g., ["123"])
//   - Attr: An attribute name (e.g., ["name"], ["profile", "email"])
//   - Value: A typed value containing the actual data
//
// Core Types:
//
// ID represents a hierarchical identifier as an array of strings.
// Use NewID() to create IDs from strings or existing ID values.
//
// Attr represents an attribute name as an array of strings, supporting nested attributes.
// Use NewAttr() to create attributes from strings or existing Attr values.
//
// Tuple is the fundamental building block combining an ID, Attr, and Value.
// Tuples are immutable and represent a single fact about an entity.
//
// Record is a collection of tuples sharing the same ID.
// Records are mutable and thread-safe for concurrent access.
//
// URI provides unique references to repositories, records, and tuples.
//
// Value is a typed container supporting Go's basic types plus arrays and maps.
// Values provide type-safe casting methods and automatic type inference.
//
// Schema provides structure and validation for records.
// Schemas define field types, constraints, and validation rules similar to JSON Schema.
//
// Example usage:
//
//	// Create a record for user "123"
//	user := NewRecord("user", "123")
//	user.Set("name", "John Doe")
//	user.Set("age", 30)
//
//	// Access individual tuples
//	nameTuple := user.Get("name")
//	fmt.Println(nameTuple.ToString()) // "John Doe"
//
//	// Create tuples directly
//	emailTuple := NewTuple(
//		"user",
//		NewID("123"),
//		NewAttr("profile", "email"),
//		"john@example.com",
//	)
package core
