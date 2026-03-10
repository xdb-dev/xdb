package schema

import "github.com/xdb-dev/xdb/core"

// Mode controls how a schema validates data.
type Mode int

const (
	// ModeOpen allows any fields (schema is advisory).
	ModeOpen Mode = iota

	// ModeStrict rejects fields not defined in the schema.
	ModeStrict
)

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
