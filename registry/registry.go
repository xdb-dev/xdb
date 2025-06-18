// Package registry provides APIs to parse and manage schemas.
package registry

import "github.com/xdb-dev/xdb/types"

// Registry is a registry of schemas.
type Registry struct {
	schemas map[string]*types.Schema
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]*types.Schema)}
}

// All returns all schemas in the registry.
func (r *Registry) All() map[string]*types.Schema {
	return r.schemas
}

// Add adds a schema to the registry.
func (r *Registry) Add(schema *types.Schema) {
	r.schemas[schema.Kind] = schema
}

// Get gets a schema from the registry.
func (r *Registry) Get(kind string) *types.Schema {
	return r.schemas[kind]
}
