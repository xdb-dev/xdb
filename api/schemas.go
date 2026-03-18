package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// SchemaService provides schema operations.
type SchemaService struct {
	store store.SchemaStore
}

// NewSchemaService creates a [SchemaService] backed by the given [store.SchemaStore].
func NewSchemaService(s store.SchemaStore) *SchemaService {
	return &SchemaService{store: s}
}

// CreateSchemaRequest is the request for schemas.create.
type CreateSchemaRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// CreateSchemaResponse is the response for schemas.create.
type CreateSchemaResponse struct {
	Data *schema.Def `json:"data"`
}

// Create creates a new schema definition.
func (s *SchemaService) Create(_ context.Context, _ *CreateSchemaRequest) (*CreateSchemaResponse, error) {
	return nil, fmt.Errorf("api: schemas.create not implemented")
}

// GetSchemaRequest is the request for schemas.get.
type GetSchemaRequest struct {
	URI string `json:"uri"`
}

// GetSchemaResponse is the response for schemas.get.
type GetSchemaResponse struct {
	Data *schema.Def `json:"data"`
}

// Get retrieves a schema definition by URI.
func (s *SchemaService) Get(_ context.Context, _ *GetSchemaRequest) (*GetSchemaResponse, error) {
	return nil, fmt.Errorf("api: schemas.get not implemented")
}

// ListSchemasRequest is the request for schemas.list.
type ListSchemasRequest struct {
	URI    string `json:"uri"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// ListSchemasResponse is the response for schemas.list.
type ListSchemasResponse struct {
	Items      []*schema.Def `json:"items"`
	NextOffset int           `json:"next_offset,omitempty"`
	Total      int           `json:"total"`
}

// List lists schema definitions.
func (s *SchemaService) List(_ context.Context, _ *ListSchemasRequest) (*ListSchemasResponse, error) {
	return nil, fmt.Errorf("api: schemas.list not implemented")
}

// UpdateSchemaRequest is the request for schemas.update (patch semantics).
type UpdateSchemaRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// UpdateSchemaResponse is the response for schemas.update.
type UpdateSchemaResponse struct {
	Data *schema.Def `json:"data"`
}

// Update updates an existing schema definition.
func (s *SchemaService) Update(_ context.Context, _ *UpdateSchemaRequest) (*UpdateSchemaResponse, error) {
	return nil, fmt.Errorf("api: schemas.update not implemented")
}

// DeleteSchemaRequest is the request for schemas.delete.
type DeleteSchemaRequest struct {
	URI     string `json:"uri"`
	Cascade bool   `json:"cascade,omitempty"`
}

// DeleteSchemaResponse is the response for schemas.delete.
type DeleteSchemaResponse struct{}

// Delete deletes a schema by URI.
func (s *SchemaService) Delete(_ context.Context, _ *DeleteSchemaRequest) (*DeleteSchemaResponse, error) {
	return nil, fmt.Errorf("api: schemas.delete not implemented")
}
