package core

// Schema is a definition of a record's schema.
type Schema struct {
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      []*Field `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// Field defines a field in a schema.
type Field struct {
	Name        string         `json:"name,omitempty" yaml:"name,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Type        Type           `json:"type,omitempty" yaml:"type,omitempty"`
	Required    bool           `json:"required,omitempty" yaml:"required,omitempty"`
	Options     map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}
