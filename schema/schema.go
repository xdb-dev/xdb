package schema

import (
	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

// ErrInvalidMode is returned when an unknown mode string is encountered.
var ErrInvalidMode = errors.New("[xdb/schema] invalid mode")

// Mode controls how a schema validates data.
type Mode string

const (
	// ModeFlexible allows records to have arbitrary attributes
	// without predefined structure.
	ModeFlexible Mode = "flexible"

	// ModeStrict requires records to have attributes
	// defined in the schema.
	ModeStrict Mode = "strict"

	// ModeDynamic automatically infers and adds new fields.
	ModeDynamic Mode = "dynamic"
)

// validModes is the set of all valid mode values.
var validModes = map[Mode]struct{}{
	ModeFlexible: {},
	ModeStrict:   {},
	ModeDynamic:  {},
}

// FieldDef describes a single field in a schema definition.
// ElemType is only set when Type is [core.TIDArray]; it records the
// element type so array values can be decoded back into typed arrays.
type FieldDef struct {
	Type     core.TID `json:"type"`
	ElemType core.TID `json:"elem_type,omitempty"`
	Required bool     `json:"required,omitempty"`
}

// CoreType returns the full [core.Type] for this field, preserving
// the element type for arrays.
func (f FieldDef) CoreType() core.Type {
	if f.Type == core.TIDArray {
		return core.NewArrayType(f.ElemType)
	}
	return core.NewType(f.Type)
}

// Def represents a schema definition.
type Def struct {
	URI    *core.URI
	Fields map[string]FieldDef
	Mode   Mode
}
