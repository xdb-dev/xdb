// Package core provides the fundamental data structures for XDB, a tuple-based database abstraction.
//
// XDB Data Model:
//
// XDB models data as a tree of Repositories, Collections, Records, and Tuples:
//
//	┌─────────────────────────────────┐
//	│           Repository            │
//	└────────────────┬────────────────┘
//	                 ↓
//	┌─────────────────────────────────┐
//	│           Collection            │
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
// Tuple is the fundamental building block in XDB. Each tuple contains:
//   - RKey: A URI identifying the record (NS + SCHEMA + ID)
//   - Attr: An attribute name (e.g., "name", "profile.email")
//   - Value: A typed value containing the actual data
//
// Record is a collection of tuples sharing the same RKey.
// Records are similar to objects, structs, or rows in a database.
// Records typically represent a single entity or object of domain data.
//
// Collection is a collection of records with the same Schema.
// Collections are identified by their schema name and are unique within a repository.
//
// Repository is a data repository holding one or more Collections.
// Repositories are typically used to group collections by domain, application, or tenant.
// Repositories are identified by their NS (namespace).
//
// Schema is a definition of your domain entities and their relationships.
// Schemas can be "strict" or "flexible". Strict schemas enforce a predefined structure
// on the data, while flexible schemas allow for arbitrary data.
//
// URI provides unique references to repositories, collections, records, and attributes.
// The general format is:
//
//	xdb:// NS [ / SCHEMA ] [ / ID ] [ #ATTRIBUTE ]
//
// Examples:
//
//	Repository: xdb://com.example
//	Collection: xdb://com.example/posts
//	Record:     xdb://com.example/posts/123-456-789
//	Attribute:  xdb://com.example/posts/123-456-789#author.id
//
// NS identifies the data repository.
// SCHEMA is the collection name.
// ID is the record identifier
// ATTRIBUTE is a specific attribute of a record (supports nesting like "profile.email").
// RECORD KEY (rkey): NS, SCHEMA, and ID combined uniquely identify a record.
//
// Value is a typed container supporting Go's basic types plus arrays and maps.
// Values provide type-safe casting methods and automatic type inference.
//
// Example usage:
//
//	// Create a repository URI
//	repo := NewURI("com.example")
//
//	// Create a record key (repository + collection + id)
//	rkey := repo.WithSchema("posts").WithID("123-456-789")
//
//	// Create tuples
//	title := NewTuple(rkey, NewAttr("title"), "Hello World")
//	author := NewTuple(rkey, NewAttr("author", "id"), "user-001")
//
//	// Get tuple URI
//	uri := title.URI() // xdb://com.example/posts/123-456-789#title
package core
