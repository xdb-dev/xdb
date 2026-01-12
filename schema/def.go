package schema

import (
	"github.com/xdb-dev/xdb/core"
)

// Mode defines how records are validated against the schema.
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

// Def defines the structure and validation rules for records.
// It provides metadata, field definitions, and record-level constraints.
type Def struct {
	// NS is the namespace this schema belongs to.
	NS *core.NS

	// Name is the schema name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Version tracks schema evolution (e.g., "1.0.0").
	Version string

	// Mode defines how records are validated against the schema.
	Mode Mode

	// Fields is a list of field schemas.
	// Use hierarchical paths for nested fields (e.g., "profile.email").
	Fields []*FieldDef
}

// Clone returns a deep copy of the Def.
func (s *Def) Clone() *Def {
	clone := &Def{
		NS:          s.NS,
		Name:        s.Name,
		Description: s.Description,
		Version:     s.Version,
		Mode:        s.Mode,
		Fields:      make([]*FieldDef, 0, len(s.Fields)),
	}
	for _, field := range s.Fields {
		clone.Fields = append(clone.Fields, field.Clone())
	}
	return clone
}

// GetField returns the field definition for the given path.
func (s *Def) GetField(path string) *FieldDef {
	for _, field := range s.Fields {
		if field.Name == path {
			return field
		}
	}
	return nil
}

// AddFields adds field definitions to the schema, skipping duplicates.
func (s *Def) AddFields(fields ...*FieldDef) {
	for _, field := range fields {
		if s.GetField(field.Name) == nil {
			s.Fields = append(s.Fields, field)
		}
	}
}

// FieldDef defines the definition for a single field.
type FieldDef struct {
	// Name is the field name.
	Name string

	// Description provides human-readable documentation.
	Description string

	// Type specifies the field's full type
	Type core.Type
}

// Equals returns true if this FieldDef is equal to the other FieldDef.
func (f *FieldDef) Equals(other *FieldDef) bool {
	return f.Name == other.Name &&
		f.Description == other.Description &&
		f.Type.Equals(other.Type)
}

// Clone returns a deep copy of the FieldDef.
func (f *FieldDef) Clone() *FieldDef {
	return &FieldDef{
		Name:        f.Name,
		Description: f.Description,
		Type:        f.Type,
	}
}
