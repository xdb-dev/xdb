// Package xdbstruct imports Go struct types into a [schema.Def]. It also
// provides the typed read and write path, [Marshal] and [Unmarshal], so
// callers work with their own types instead of tuples.
//
// # Tag grammar
//
// Fields are mapped with the `xdb` struct tag. The tag obeys the
// encoding/json conventions:
//
//	`xdb:"name,required"`   // attribute "name", Required
//	`xdb:"name"`            // attribute "name", optional
//	`xdb:"-"`               // skip the field
//	`xdb:",required"`       // default name (the Go field name), Required
//	`xdb:"attrs,json"`      // store as a raw JSON value (maps, interfaces)
//
// An empty tag name defaults to the Go field name verbatim, as in
// encoding/json. Only exported fields are considered.
//
// # Type mapping
//
//   - bool/int*/uint*/float*/string → their scalar TIDs.
//   - time.Time → TIME. []byte → BYTES. json.RawMessage → JSON.
//   - A named scalar (type UserID string) → its underlying TID, with
//     Annotations["go.type"] set to the source type name. Unmarshal restores
//     the named type through the static field type of the destination.
//   - A nested struct (by value or pointer) flattens to dotted attributes
//     (Profile.Name → "profile.name"). A pointer is nullable. A nil pointer
//     marshals as absent (no tuple) and unmarshals back to nil.
//   - []Struct → an object array: ARRAY<JSON> with the fields of the element
//     struct as Items. []scalar → ARRAY<scalar>.
//   - Embedded (anonymous) structs obey Go field promotion.
//   - `xdb:"...,json"` maps any field (including maps and interfaces) to a
//     raw JSON value. It is the opt-in for recursive types.
//
// # Rejections
//
// Each of the following is an error that names the field and the fix:
//
//   - A map or interface without the `json` opt-in.
//   - A channel or function field.
//   - A recursive type (User → Manager → User), unless the recursive field
//     opts into `json`.
//
// # Limitations
//
// Required on a nested-struct field is not representable in the schema. The
// schema has no single field for a nested object, only its dotted leaves.
// As a result, the marker is not enforced. Mark the leaves Required instead.
package xdbstruct
