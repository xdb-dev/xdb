package schema

// Record defines the schema of a record.
type Record struct {
	Kind       string      `yaml:"kind"`
	Attributes []Attribute `yaml:"attributes"`
	Table      string      `yaml:"table"`
}

// Get returns the attribute with the given name.
func (r *Record) Get(name string) *Attribute {
	for _, attr := range r.Attributes {
		if attr.Name == name {
			return &attr
		}
	}

	return nil
}

// Attribute defines the attribute metadata.
type Attribute struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Repeated bool   `yaml:"repeated"`
}

// Schema is the registry of all kinds of records.
type Schema struct {
	Records []Record `yaml:"records"`
}

// Get returns the record with the given kind.
func (s *Schema) Get(kind string) *Record {
	for _, record := range s.Records {
		if record.Kind == kind {
			return &record
		}
	}

	return nil
}
