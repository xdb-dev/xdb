// Package registry provides APIs to parse and manage schemas.
package registry

import "github.com/xdb-dev/xdb/core"

// Registry is a registry of schemas.
type Registry struct {
	schemas map[string]*core.Schema
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]*core.Schema)}
}

// All returns all schemas in the registry.
func (r *Registry) All() map[string]*core.Schema {
	return r.schemas
}

// Add adds a schema to the registry.
func (r *Registry) Add(schema *core.Schema) {
	r.schemas[schema.Kind] = schema
}

// Get gets a schema from the registry.
func (r *Registry) Get(kind string) *core.Schema {
	return r.schemas[kind]
}
