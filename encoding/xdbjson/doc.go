// Package xdbjson is the JSON adapter: it imports JSON Schema documents into
// XDB schemas and converts JSON documents to and from XDB records.
//
// # Overview
//
// The package has two paths that share one option set:
//
//   - The schema path. [ImportSchema] parses a JSON Schema document (a
//     documented subset of draft 2020-12) into a [schema.Def].
//   - The data path. [Marshal] and [MarshalInto] decode a JSON document into a
//     [core.Record]; [Unmarshal] encodes a record back to a JSON document.
//
// The two compose: import a schema once, then pass the resulting def to the
// data path with [WithDef] so declared fields decode as their declared types.
// The shared round-trip harness (tests.RunRoundTrip) exercises them together.
//
// # Data path
//
// Encode a record:
//
//	record := core.NewRecord("com.example", "users", "123").
//	    Set("name", "John Doe").
//	    Set("email", "john@example.com")
//
//	data, err := xdbjson.Unmarshal(record)
//	// {"_id":"123","email":"john@example.com","name":"John Doe"}
//
// Decode a document:
//
//	data := []byte(`{"_id":"123","name":"John Doe"}`)
//
//	record, err := xdbjson.Marshal(data,
//	    xdbjson.WithNS("com.example"),
//	    xdbjson.WithSchema("users"),
//	)
//	// record.URI() -> xdb://com.example/users/123
//
// [Marshal] builds a new record; [MarshalInto] populates one the caller
// already owns, preserving its identity. Unlike encoding/xdbstruct and
// encoding/xdbproto, neither takes a URI: a JSON document carries its own
// identity.
//
// Per-call output options control one [Unmarshal]:
//
//	data, err := xdbjson.Unmarshal(record, xdbjson.WithIndent("", "  "))
//	data, err := xdbjson.Unmarshal(record, xdbjson.WithFields("name", "email"))
//
// # JSON format
//
// With [WithIncludeNS] and [WithIncludeSchema], encoded JSON includes
// the namespace and schema alongside the record ID:
//
//	{
//	    "_id": "user-123",
//	    "_ns": "com.example",
//	    "_schema": "users",
//	    "name": "John Doe",
//	    "email": "john@example.com"
//	}
//
// Rename them with [WithIDField], [WithNSField], and [WithSchemaField]. By
// default [Unmarshal] emits only the ID; add [WithIncludeNS] and
// [WithIncludeSchema] for the rest.
//
// # Nested objects
//
// Nested JSON objects flatten to dot-notation attributes.
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
// On encode, dot-notation attributes unflatten back to nested objects.
//
// # Metadata resolution (decoding)
//
// [Marshal] resolves metadata in this order:
//  1. The JSON field, if present
//  2. The default from [WithNS] / [WithSchema]
//
// # Schema path: type mapping
//
//	JSON Schema construct              XDB
//	--------------------------------   ------------------------------------------
//	type: object (root)                schema (Def); properties -> fields
//	type: object (nested property)     flattened to dotted attributes
//	type: object (inside an element)   opaque JSON member (element not flattened)
//	object with typed                  JSON field (map-like); annotation
//	  additionalProperties               jsonschema.additionalProperties=schema
//	items: {type: object}              object array (ARRAY<JSON> + Field.Items)
//	items: {type: scalar}              ARRAY<scalar>
//	type: string                       STRING
//	type: string, format: date-time    TIME
//	type: integer                      INTEGER
//	type: number                       FLOAT
//	type: boolean                      BOOLEAN
//	type: ["T", "null"]                T (nullable; not required unless listed)
//	required: [...]                    Field.Required on the named members
//	additionalProperties: false        Mode strict
//	additionalProperties: true|absent  Mode flexible
//	$ref: "#/..." (same document)      resolved and imported inline
//	enum / const / pattern / min /     annotations only (XDB does not enforce);
//	  max / minLength / maxLength         an enum's scalar type still maps
//	description (schema and fields)    Def.Description / Field.Description
//
// A field has no annotations by default. Constraint keywords are captured in
// Field.Annotations under a "jsonschema." prefix (format, enum, const,
// pattern, minimum, maximum, exclusiveMinimum, exclusiveMaximum, minLength,
// maxLength). The Def records Annotations["source"]="jsonschema". When the
// document has an $id, the Def also records it in
// Annotations["jsonschema.id"].
//
// # Two nesting representations
//
// Nested objects flatten to dotted attributes (profile.name), unless
// [WithDef] declares the field as JSON and preserves it as one value.
// An array of objects uses Field.Items, a separate namespace whose elements
// stay opaque. Inside an object-array element, a nested object imports as an
// opaque JSON member, because the data path keeps element internals nested.
//
// The required members of a nested object are enforced only when the object
// itself is required at its parent. An optional nested object can be omitted
// whole, so its members are conditionally required. The flat IR cannot
// express this condition and does not enforce it.
//
// # Key grammar (wire-format commitment)
//
// A property name must parse as a single attribute segment. It must not
// contain '.', which is the path separator and is ambiguous in a name. An
// offending name is an import error ([ErrInvalidKey]). The error lists every
// offending key with its JSON pointer. There is no escaping.
//
// # $ref
//
// Only same-document pointers ("#/$defs/Foo") are resolved. A cross-document
// $ref is [ErrCrossDocument]. A cyclic $ref is [ErrCyclicRef]. To break a
// cycle, use [WithOpaqueJSON], which imports the named pointer as an opaque
// JSON field.
//
// # Rejections
//
// Every unsupported construct is an error that names the JSON pointer to the
// offending node:
//
//	anyOf / oneOf                      ErrUnion (unions are a non-goal)
//	allOf of conflicting keys          ErrConflict
//	allOf branch that is not an object ErrUnsupported
//	cross-document $ref                ErrCrossDocument
//	cyclic $ref (without WithOpaqueJSON) ErrCyclicRef
//	unresolved same-document $ref      ErrUnresolvedRef
//	invalid / dotted property name     ErrInvalidKey
//	root that is not an object         ErrUnsupported
//	typed additionalProperties at root ErrUnsupported
//	multiple non-null types            ErrUnsupported
//	arrays of arrays                   ErrUnsupported
//
// allOf of disjoint objects is the one composition that is supported. The
// field sets of the branches merge into one. A key that more than one branch
// defines is [ErrConflict].
//
// # Null, absent, and zero handling
//
// An absent property and an explicit null both decode to no tuple, so the
// record omits the attribute. Both encode back to an absent property. As a
// result, null and absent are indistinguishable after a round-trip. A zero
// value (0, "", false) is a present tuple and round-trips as itself.
// Object-array elements are opaque JSON, so a null member inside an element is
// preserved verbatim.
//
// # Errors
//
// Encoding errors:
//   - The record is nil ([ErrNilRecord])
//
// Decoding errors:
//   - Invalid JSON ([ErrInvalidJSON])
//   - Missing or empty ID field ([ErrMissingID], [ErrEmptyID])
//   - No namespace or schema ([ErrMissingNamespace], [ErrMissingSchema])
//   - A declared field whose value cannot decode as the declared type
//     (wraps [core.ErrSchemaViolation])
package xdbjson
