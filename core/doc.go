// Package core provides the fundamental data structures for XDB, an
// agent-first data layer.
//
// # Data model
//
// XDB models data as a tree of namespaces, schemas, records, and tuples:
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
//	│      Path | Attr | Value        │
//	└─────────────────────────────────┘
//
// # Core types
//
// [Tuple] is the fundamental building block in XDB. A tuple is an
// addressable fact: xdb://ns/schema/id#attr = value. Each tuple contains:
//   - Path: a [URI] that identifies the record (NS + SCHEMA + ID).
//   - Attr: an attribute name, for example "name" or "profile.email".
//   - Value: a typed [Value] that holds the data.
//
// [Record] is the set of tuples that share the same path. A record groups
// tuples and adds no data of its own. A record exists exactly when at least
// one tuple exists at its path. Records are similar to objects, structs, or
// rows in a database. A record usually represents one entity of domain data.
//
// Schema defines the structure of records and groups them together. A schema
// is identified by name within a namespace. A schema has one of three modes:
// strict, flexible, or dynamic. Declared fields type-check in every mode.
// The mode controls only undeclared attributes. Strict rejects them.
// Flexible accepts them as-is. Dynamic infers a field for each of them and
// adds it to the schema. The schema package defines the modes.
//
// Namespace (NS) groups one or more schemas. Namespaces usually organize
// schemas by domain, application, or tenant.
//
// [URI] gives a unique reference to a namespace, schema, record, or
// attribute. The general format is:
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
// ID is the record identifier.
// ATTRIBUTE is one attribute of a record. Attribute names can nest, for
// example "profile.email".
// Path is NS, SCHEMA, and ID combined. The path identifies one record. It is
// the URI without the xdb:// scheme.
//
// [Value] is a typed container for the basic Go types, time.Time, []byte,
// json.RawMessage, and arrays of these. Maps are not supported and are
// rejected with [ErrUnsupportedValue]. Values provide typed As* accessors and
// automatic type inference.
//
// # Example
//
//	// A record is the set of tuples that share a path. Each Set adds one tuple.
//	record := NewRecord("com.example", "posts", "123-456-789").
//		Set("title", "Hello World").
//		Set("author.id", "user-001")
//
//	// Read a tuple back. As* is safe to chain on a missing attribute.
//	title, err := record.Get("title").AsStr()
//
//	// A standalone tuple, addressed by its path and attribute.
//	tuple := NewTuple("com.example/posts/123-456-789", "title", "Hello World")
//	uri := tuple.URI() // xdb://com.example/posts/123-456-789#title
package core
