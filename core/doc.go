// Package core provides the fundamental data structures for XDB — an agent-first data layer.
//
// XDB Data Model:
//
// XDB models data as a tree of Namespaces, Schemas, Records, and Tuples:
//
//	┌─────────────────────────────────┐
//	│            Namespace            │
//	└────────────────┬────────────────┘
//	                 ↓
//	┌─────────────────────────────────┐
//	│             Schema              │
//	└────────────────┬────────────────┘
//	                 ↓
//	┌─────────────────────────────────┐
//	│             Record              │
//	└────────────────┬────────────────┘
//	                 ↓
//	┌─────────────────────────────────┐
//	│             Tuple               │
//	├─────────────────────────────────┤
//	│   ID | Attr | Value | Options   │
//	└─────────────────────────────────┘
//
// Core Types:
//
// Tuple is the fundamental building block in XDB — an addressable fact,
// xdb://ns/schema/id#attr = value. Each tuple contains:
//   - Path: A URI identifying the record (NS + SCHEMA + ID)
//   - Attr: An attribute name (e.g., "name", "profile.email")
//   - Value: A typed value containing the actual data
//
// Record is the set of tuples that share the same path; it groups tuples and
// adds no data of its own. A record exists exactly when at least one tuple
// exists at its path. Records are similar to objects, structs, or rows in a
// database, and typically represent a single entity of domain data.
//
// Schema defines the structure of records and groups them together.
// Schemas can be "strict" or "flexible" and are uniquely identified by name within a namespace.
//
// Namespace (NS) groups one or more Schemas.
// Namespaces are typically used to organize schemas by domain, application, or tenant.
//
// Schema is a definition of your domain entities and their relationships.
// Schemas can be "strict" or "flexible". Strict schemas enforce a predefined structure
// on the data, while flexible schemas allow for arbitrary data.
//
// URI provides unique references to namespaces, schemas, records, and attributes.
// The general format is:
//
//	xdb:// NS [ / SCHEMA ] [ / ID ] [ #ATTRIBUTE ]
//
// Examples:
//
//	Namespace:  xdb://com.example
//	Schema:     xdb://com.example/posts
//	Record:     xdb://com.example/posts/123-456-789
//	Attribute:  xdb://com.example/posts/123-456-789#author.id
//
// NS identifies the namespace.
// SCHEMA is the schema name.
// ID is the record identifier
// ATTRIBUTE is a specific attribute of a record (supports nesting like "profile.email").
// Path: NS, SCHEMA, and ID combined uniquely identify a record (URI without xdb://).
//
// Value is a typed container supporting Go's basic types plus arrays and maps.
// Values provide type-safe casting methods and automatic type inference.
//
// Example usage:
//
//	// A record is the set of tuples sharing a path; Set adds one tuple each.
//	record := NewRecord("com.example", "posts", "123-456-789").
//		Set("title", "Hello World").
//		Set("author.id", "user-001")
//
//	// Read a tuple back; As* is safe to chain on a missing attribute.
//	title, err := record.Get("title").AsStr()
//
//	// A standalone tuple, addressed by its path and attribute.
//	tuple := NewTuple("com.example/posts/123-456-789", "title", "Hello World")
//	uri := tuple.URI() // xdb://com.example/posts/123-456-789#title
package core
