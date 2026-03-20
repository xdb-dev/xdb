// Package xdbjson provides utilities for converting JSON to XDB records and vice versa.
//
// # Overview
//
// The xdbjson package provides bidirectional conversion between JSON and XDB records:
//   - Flat metadata fields (_id, _ns, _schema) with customizable field names
//   - Nested JSON objects flattened to dot-notation attributes
//   - Configurable metadata inclusion in JSON output
//
// # Basic Usage
//
// Create an encoder to convert records to JSON:
//
//	record := core.NewRecord("com.example", "users", "123").
//	    Set("name", "John Doe").
//	    Set("email", "john@example.com")
//
//	encoder := xdbjson.New()
//	data, err := encoder.FromRecord(record)
//	// {"_id":"123","email":"john@example.com","name":"John Doe"}
//
// Create a decoder to convert JSON to records:
//
//	data := []byte(`{"_id":"123","name":"John Doe"}`)
//
//	decoder := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("users"))
//	record, err := decoder.ToRecord(data)
//	// record.URI() -> xdb://com.example/users/123
//
// # JSON Format
//
// With default options, JSON uses flat metadata fields:
//
//	{
//	    "_id": "user-123",
//	    "_ns": "com.example",
//	    "_schema": "users",
//	    "name": "John Doe",
//	    "email": "john@example.com"
//	}
//
// Metadata fields can be customized via functional options:
//
//	enc := xdbjson.New(
//	    xdbjson.WithIDField("id"),
//	    xdbjson.WithNSField("namespace"),
//	    xdbjson.WithSchemaField("type"),
//	)
//
// # Nested Objects
//
// Nested JSON objects are flattened to dot-notation attributes:
//
// Input JSON:
//
//	{
//	    "_id": "123",
//	    "address": {
//	        "street": "123 Main St",
//	        "city": "Boston"
//	    }
//	}
//
// Record attributes:
//   - address.street: "123 Main St"
//   - address.city: "Boston"
//
// When encoding, dot-notation attributes are unflattened back to nested objects.
//
// # Including Metadata in Output
//
// By default, the encoder only includes the ID field. To include namespace
// and schema in the JSON output:
//
//	encoder := xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())
//
// # Per-Call Options
//
// Use [EncodeOption] values to control individual FromRecord calls:
//
//	data, err := enc.FromRecord(record, xdbjson.WithIndent("", "  "))
//	data, err := enc.FromRecord(record, xdbjson.WithFields("name", "email"))
//
// # Custom Field Names
//
// Use functional options to customize metadata field names:
//
//	decoder := xdbjson.NewDecoder(
//	    xdbjson.WithNS("com.example"),
//	    xdbjson.WithSchema("users"),
//	    xdbjson.WithIDField("userId"),       // Look for "userId" instead of "_id"
//	    xdbjson.WithNSField("namespace"),    // Look for "namespace" instead of "_ns"
//	    xdbjson.WithSchemaField("type"),     // Look for "type" instead of "_schema"
//	)
//
// # Metadata Resolution (Decoding)
//
// When decoding, metadata is resolved in this order:
//  1. JSON field (if present)
//  2. Options default value
//
// For example, if JSON contains "_ns", that value is used.
// Otherwise, the WithNS value is used.
//
// # Error Handling
//
// Encoding errors:
//   - Record is nil
//
// Decoding errors:
//   - Invalid JSON
//   - Missing ID field
//   - Empty ID value
//   - Cannot determine namespace (not in JSON and WithNS not set)
//   - Cannot determine schema (not in JSON and WithSchema not set)
package xdbjson
