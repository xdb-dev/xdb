package types

// Schema defines the structure of a record.
type Schema struct {
	Kind       string      `json:"kind" yaml:"kind"`
	Attributes []Attribute `json:"attributes" yaml:"attributes"`
}

// Attribute defines a single attribute in a record.
type Attribute struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"`
	PrimaryKey bool   `json:"primary_key" yaml:"primary_key"`
	Repeated   bool   `json:"repeated" yaml:"repeated"`
}

// Registry is a registry of schemas.
type Registry struct {
	schemas map[string]*Schema
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]*Schema)}
}

// All returns all schemas in the registry.
func (r *Registry) All() map[string]*Schema {
	return r.schemas
}

// Add adds a schema to the registry.
func (r *Registry) Add(schema *Schema) {
	r.schemas[schema.Kind] = schema
}

// Get gets a schema from the registry.
func (r *Registry) Get(kind string) *Schema {
	return r.schemas[kind]
}
