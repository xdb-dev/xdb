// Package schema provides utilities for loading and parsing XDB schemas
// from various file formats including JSON and YAML.
//
// The package supports loading schema definitions that include:
//   - Scalar types (STRING, INTEGER, FLOAT, BOOLEAN, etc.)
//   - Array types
//   - Map types
//   - Nested field paths
//   - Required field validation
//   - Default values
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
//	  ],
//	  "required": ["name"]
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
//	required:
//	  - name
package schema
