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
//	encoder := xdbjson.NewDefaultEncoder()
//	data, err := encoder.FromRecord(record)
//	// {"_id":"123","email":"john@example.com","name":"John Doe"}
//
// Create a decoder to convert JSON to records:
//
//	data := []byte(`{"_id":"123","name":"John Doe"}`)
//
//	decoder := xdbjson.NewDefaultDecoder("com.example", "users")
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
// Metadata fields can be customized via Options:
//
//	opts := xdbjson.Options{
//	    IDField:     "id",
//	    NSField:     "namespace",
//	    SchemaField: "type",
//	}
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
//	encoder := xdbjson.NewEncoder(xdbjson.Options{
//	    IncludeNS:     true,
//	    IncludeSchema: true,
//	})
//
// # Custom Field Names
//
// Use Options to customize metadata field names:
//
//	decoder := xdbjson.NewDecoder(xdbjson.Options{
//	    NS:          "com.example",
//	    Schema:      "users",
//	    IDField:     "userId",    // Look for "userId" instead of "_id"
//	    NSField:     "namespace", // Look for "namespace" instead of "_ns"
//	    SchemaField: "type",      // Look for "type" instead of "_schema"
//	})
//
// # Metadata Resolution (Decoding)
//
// When decoding, metadata is resolved in this order:
//  1. JSON field (if present)
//  2. Options default value
//
// For example, if JSON contains "_ns", that value is used.
// Otherwise, Options.NS is used.
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
//   - Cannot determine namespace (not in JSON and Options.NS is empty)
//   - Cannot determine schema (not in JSON and Options.Schema is empty)
package xdbjson
