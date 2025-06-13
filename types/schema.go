package types

// Schema defines the structure of a record.
type Schema struct {
	Kind       string      `json:"kind" yaml:"kind"`
	Attributes []Attribute `json:"attributes" yaml:"attributes"`
}

// GetAttribute returns the attribute with the given name.
func (s *Schema) GetAttribute(name string) *Attribute {
	for _, attr := range s.Attributes {
		if attr.Name == name {
			return &attr
		}
	}

	return nil
}

// Attribute defines a single attribute in a record.
type Attribute struct {
	Name       string `json:"name" yaml:"name"`
	Type       Type   `json:"type" yaml:"type"`
	PrimaryKey bool   `json:"primary_key" yaml:"primary_key"`
}
