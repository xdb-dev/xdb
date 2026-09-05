// Package jsonschemaimport imports a documented subset of JSON Schema
// (draft 2020-12) into an XDB [schema.Def]. It also moves JSON documents to
// and from [core.Record] values through encoding/xdbjson.
//
// [Import] parses a schema document into a [schema.Def]. [Marshal] and
// [Unmarshal] are the typed data path. Marshal decodes a JSON document into
// a record and types the declared fields. Unmarshal encodes a record back to
// a JSON document. The shared round-trip harness (tests.RunRoundTrip)
// exercises the three together.
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
// A field has no annotations by default. Constraint keywords are captured in
// Field.Annotations under a "jsonschema." prefix (format, enum, const,
// pattern, minimum, maximum, exclusiveMinimum, exclusiveMaximum, minLength,
// maxLength). The Def records Annotations["source"]="jsonschema". When the
// document has an $id, the Def also records it in
// Annotations["jsonschema.id"].
//
// # Two nesting representations
//
// A single nested object always flattens to dotted attributes
// (profile.name), so filters can address it. An array of objects uses
// Field.Items, a separate namespace whose elements stay opaque. Inside an
// object-array element, a nested object is NOT flattened. It imports as an
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
// cycle, use [WithJSON], which imports the named pointer as an opaque JSON
// field.
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
//	cyclic $ref (without WithJSON)     ErrCyclicRef
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
// The data path obeys the rules of encoding/xdbjson. An absent property and
// an explicit null both decode to no tuple, so the record omits the
// attribute. Both encode back to an absent property. As a result, null and
// absent are indistinguishable after a round-trip. A zero value (0, "",
// false) is a present tuple and round-trips as itself. Object-array elements
// are opaque JSON, so a null member inside an element is preserved verbatim.
package jsonschemaimport
