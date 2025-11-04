package core

// Schema is a definition of a record's schema.
type Schema struct {
	Name        string         `json:"name,omitempty" yaml:"name,omitempty"`
	Type        string         `json:"type,omitempty" yaml:"type,omitempty"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      []*Schema      `json:"fields,omitempty" yaml:"fields,omitempty"`
	Items       *Schema        `json:"items,omitempty" yaml:"items,omitempty"`
	Options     map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
	Required    bool           `json:"required,omitempty" yaml:"required,omitempty"`
}
