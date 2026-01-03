// Package schema provides utilities for loading and parsing XDB schemas
// from various file formats including JSON and YAML.
//
// The package defines schema structures and supports loading schema definitions:
//   - [Def]: Schema definition with metadata, fields, and validation rules
//   - [FieldDef]: Individual field definition with type and constraints
//   - [Mode]: Validation mode (flexible or strict)
//
// Supported field types:
//   - Scalar types (STRING, INTEGER, FLOAT, BOOLEAN, etc.)
//   - Array types
//   - Map types
//   - Nested field paths
//
// Example JSON schema:
//
//	{
//	  "name": "User",
//	  "description": "User record schema",
//	  "version": "1.0.0",
//	  "fields": [
//	    {
//	      "name": "name",
//	      "description": "User's full name",
//	      "type": "STRING"
//	    },
//	    {
//	      "name": "tags",
//	      "type": "ARRAY",
//	      "array_of": "STRING"
//	    }
//	  ]
//	}
//
// Example YAML schema:
//
//	name: User
//	description: User record schema
//	version: 1.0.0
//	fields:
//	  - name: name
//	    description: User's full name
//	    type: STRING
//	  - name: tags
//	    type: ARRAY
//	    array_of: STRING
package schema
