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
type FieldDef struct {
	Type     core.TID
	Required bool
}

// Def represents a schema definition.
type Def struct {
	URI    *core.URI
	Fields map[string]FieldDef
	Mode   Mode
}
