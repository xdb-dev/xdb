package schema

import (
	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

// ErrInvalidMode is returned when an empty or unrecognized mode is encountered.
var ErrInvalidMode = errors.New("[xdb/schema] invalid mode")

// Mode controls how a schema validates undeclared attributes.
//
// Declared fields ALWAYS type-check, in every mode. Mode governs only
// attributes that are not declared in the schema:
//
//   - [ModeFlexible]: undeclared attributes are ignored.
//   - [ModeStrict]: undeclared attributes are rejected.
//   - [ModeDynamic]: undeclared attributes are inferred and the schema evolves.
type Mode string

const (
	// ModeFlexible type-checks declared fields and ignores
	// undeclared attributes.
	ModeFlexible Mode = "flexible"

	// ModeStrict type-checks declared fields and rejects
	// undeclared attributes.
	ModeStrict Mode = "strict"

	// ModeDynamic type-checks declared fields and infers new fields
	// for undeclared attributes.
	ModeDynamic Mode = "dynamic"
)

// validModes is the set of all valid mode values.
var validModes = map[Mode]struct{}{
	ModeFlexible: {},
	ModeStrict:   {},
	ModeDynamic:  {},
}

// Field describes a single field in a schema definition.
//
// Type is the full [core.Type], carrying the element type for arrays.
// Build scalar fields with [core.NewType] and array fields with
// [core.NewArrayType].
//
// Items declares the element object schema for an array of objects. It is set
// ONLY when Type is ARRAY<JSON>: each array element must be a JSON object whose
// members type-check against Items, using the same rules as top-level fields
// (including Required within the element and one level of further nesting).
// Items is arrays-of-objects only — a single nested object flattens to dotted
// attributes (e.g. profile.name) and never uses Items.
type Field struct {
	Annotations map[string]string
	Items       map[string]Field
	Type        core.Type
	Description string
	Required    bool
}

// Def represents a schema definition. It is the intermediate representation
// that schema importers produce and that stores validate records against.
type Def struct {
	URI         *core.URI
	Fields      map[string]Field
	Annotations map[string]string
	Description string
	Mode        Mode
	Revision    int64
}
