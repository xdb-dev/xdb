// Package xdbstruct imports Go struct types into [schema.Def] and provides the
// typed read/write path — [Marshal] and [Unmarshal] — so callers work with
// their own types instead of tuples.
//
// # Tag grammar
//
// Fields are mapped with the `xdb` struct tag, following encoding/json
// conventions:
//
//	`xdb:"name,required"`   // attribute "name", Required
//	`xdb:"name"`            // attribute "name", optional
//	`xdb:"-"`               // skip the field
//	`xdb:",required"`       // default name (the Go field name), Required
//	`xdb:"attrs,json"`      // store as a raw JSON value (maps, interfaces)
//
// An empty tag name defaults to the Go field name verbatim, matching
// encoding/json. Only exported fields are considered.
//
// # Type mapping
//
//   - bool/int*/uint*/float*/string → their scalar TIDs.
//   - time.Time → TIME; []byte → BYTES; json.RawMessage → JSON.
//   - a named scalar (type UserID string) → its underlying TID, with
//     Annotations["go.type"] recording the source type name. Unmarshal restores
//     the named type through the destination's static field type.
//   - a nested struct (by value or pointer) flattens to dotted attributes
//     (Profile.Name → "profile.name"). A pointer is nullable: a nil pointer
//     marshals as absent (no tuple) and unmarshals back to nil.
//   - []Struct → an object array: ARRAY<JSON> with the element struct's fields
//     as Items. []scalar → ARRAY<scalar>.
//   - embedded (anonymous) structs follow Go field promotion.
//   - `xdb:"...,json"` maps any field (including maps and interfaces) to a raw
//     JSON value and is the escape hatch for recursive types.
//
// # Rejections
//
// Each of the following is an error naming the field and the fix:
//
//   - a map or interface without the `json` opt-in;
//   - a channel or function field;
//   - a recursive type (User → Manager → User), unless the recursive field
//     opts into `json`.
//
// # Limitations
//
// Required on a nested-struct field is not representable in the schema (the
// schema has no single field for a nested object, only its dotted leaves), so
// it is not enforced; mark the leaves Required instead.
package xdbstruct
