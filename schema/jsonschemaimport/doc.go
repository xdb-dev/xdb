// Package jsonschemaimport imports a documented subset of JSON Schema
// (draft 2020-12) into an XDB [schema.Def], and moves JSON documents to and
// from [core.Record] values via encoding/xdbjson.
//
// [Import] parses a schema document into a [schema.Def]. [Marshal] and
// [Unmarshal] are the typed data path: Marshal decodes a JSON document into a
// record (typing declared fields), Unmarshal encodes a record back to a JSON
// document. The three are exercised together by the shared round-trip harness
// (tests.RunRoundTrip).
//
// # Type mapping
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
// Every field records nothing by default; constraint keywords are captured in
// Field.Annotations under a "jsonschema." prefix (format, enum, const, pattern,
// minimum, maximum, exclusiveMinimum, exclusiveMaximum, minLength, maxLength).
// The Def records Annotations["source"]="jsonschema" and, when present,
// Annotations["jsonschema.id"] from $id.
//
// # Two nesting representations
//
// A single nested object always flattens to dotted attributes (profile.name),
// so filters can address it. An array of objects uses Field.Items, a separate
// namespace whose elements stay opaque. Inside an object-array element, a nested
// object is NOT flattened — it imports as an opaque JSON member — because the
// data path keeps element internals nested.
//
// A nested object's required members are enforced only when the object itself is
// required at its parent. An optional nested object may be omitted whole, so its
// members are conditionally required, which the flat IR cannot express and does
// not enforce.
//
// # Key grammar (wire-format commitment)
//
// A property name must parse as a single attribute segment and must not contain
// '.' (which is the path separator and would be ambiguous). Offending names are
// an import error ([ErrInvalidKey]) listing every offending key with its JSON
// pointer. There is no escaping in v1.
//
// # $ref
//
// Only same-document pointers ("#/$defs/Foo") are resolved. A cross-document
// $ref is [ErrCrossDocument]; a cyclic $ref is [ErrCyclicRef]. The escape hatch
// for a cycle is [WithJSON], which imports the named pointer as an opaque JSON
// field.
//
// # Rejections
//
// Every unsupported construct errors with the JSON pointer to the offending
// node:
//
//	anyOf / oneOf                      ErrUnion (unions are a non-goal)
//	allOf of conflicting keys          ErrConflict
//	allOf branch that is not an object ErrUnsupported
//	cross-document $ref                ErrCrossDocument
//	cyclic $ref (without WithJSON)     ErrCyclicRef
//	unresolved same-document $ref      ErrUnresolvedRef
//	invalid / dotted property name     ErrInvalidKey
//	root that is not an object         ErrUnsupported
//	typed additionalProperties at root ErrUnsupported
//	multiple non-null types            ErrUnsupported
//	arrays of arrays                   ErrUnsupported
//
// allOf of disjoint objects is the one composition that is supported: the branch
// field sets merge into one, and a key defined by more than one branch is
// [ErrConflict].
//
// # Null, absent, and zero handling
//
// The data path follows encoding/xdbjson: an absent property and an explicit
// null both decode to no tuple (the record simply omits the attribute), and
// encode back to an absent property — null and absent are indistinguishable
// after a round-trip. A zero value (0, "", false) is a present tuple and
// round-trips as itself. Object-array elements are opaque JSON, so a null member
// inside an element is preserved verbatim.
package jsonschemaimport
