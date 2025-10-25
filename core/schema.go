package core

// Schema is a definition of a record's schema.
type Schema struct {
	Name        string
	Type        string
	Description string
	Fields      []*Schema
	Items       *Schema
	Options     map[string]any
	Required    bool
}
