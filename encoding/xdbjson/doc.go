// Package xdbjson converts JSON to XDB records and records back to JSON.
//
// # Overview
//
// The xdbjson package converts in both directions between JSON and XDB
// records:
//   - Flat metadata fields (_id, _ns, _schema) with configurable names
//   - Nested JSON objects flattened to dot-notation attributes
//   - Configurable inclusion of metadata in the JSON output
//
// # Basic usage
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
// # JSON format
//
// With the default options, the JSON uses flat metadata fields:
//
//	{
//	    "_id": "user-123",
//	    "_ns": "com.example",
//	    "_schema": "users",
//	    "name": "John Doe",
//	    "email": "john@example.com"
//	}
//
// You can rename the metadata fields with functional options:
//
//	enc := xdbjson.New(
//	    xdbjson.WithIDField("id"),
//	    xdbjson.WithNSField("namespace"),
//	    xdbjson.WithSchemaField("type"),
//	)
//
// # Nested objects
//
// Nested JSON objects are flattened to dot-notation attributes.
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
// On encode, dot-notation attributes are unflattened back to nested objects.
//
// # Metadata in the output
//
// By default, the encoder includes only the ID field. To include the
// namespace and the schema in the JSON output:
//
//	encoder := xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())
//
// # Per-call options
//
// Use [EncodeOption] values to control one FromRecord call:
//
//	data, err := enc.FromRecord(record, xdbjson.WithIndent("", "  "))
//	data, err := enc.FromRecord(record, xdbjson.WithFields("name", "email"))
//
// # Custom field names
//
// Use functional options to rename the metadata fields on decode:
//
//	decoder := xdbjson.NewDecoder(
//	    xdbjson.WithNS("com.example"),
//	    xdbjson.WithSchema("users"),
//	    xdbjson.WithIDField("userId"),       // Look for "userId" instead of "_id"
//	    xdbjson.WithNSField("namespace"),    // Look for "namespace" instead of "_ns"
//	    xdbjson.WithSchemaField("type"),     // Look for "type" instead of "_schema"
//	)
//
// # Metadata resolution (decoding)
//
// On decode, the decoder resolves metadata in this order:
//  1. The JSON field, if present
//  2. The default value from the options
//
// For example, if the JSON contains "_ns", the decoder uses that value.
// Otherwise, the decoder uses the WithNS value.
//
// # Errors
//
// Encoding errors:
//   - The record is nil
//
// Decoding errors:
//   - Invalid JSON
//   - Missing ID field
//   - Empty ID value
//   - No namespace (not in the JSON and WithNS not set)
//   - No schema (not in the JSON and WithSchema not set)
//   - A declared field whose value cannot decode as the declared type
//     (wraps [core.ErrSchemaViolation])
package xdbjson
