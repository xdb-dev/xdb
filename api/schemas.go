package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xdb-dev/xdb/core"
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
// If the schema already exists, the existing definition is returned.
func (s *SchemaService) Create(ctx context.Context, req *CreateSchemaRequest) (*CreateSchemaResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}

	def, err := unmarshalSchemaDef(req.Data, uri)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}

	err = s.store.CreateSchema(ctx, uri, &def)
	if errors.Is(err, store.ErrAlreadyExists) {
		existing, getErr := s.store.GetSchema(ctx, uri)
		if getErr != nil {
			return nil, fmt.Errorf("api: schemas.create: %w", getErr)
		}
		return &CreateSchemaResponse{Data: existing}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}

	return &CreateSchemaResponse{Data: &def}, nil
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
func (s *SchemaService) Get(ctx context.Context, req *GetSchemaRequest) (*GetSchemaResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.get: %w", err)
	}

	def, err := s.store.GetSchema(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.get: %w", err)
	}

	return &GetSchemaResponse{Data: def}, nil
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
func (s *SchemaService) List(ctx context.Context, req *ListSchemasRequest) (*ListSchemasResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.list: %w", err)
	}

	q := &store.Query{
		URI:    uri,
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	page, err := s.store.ListSchemas(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.list: %w", err)
	}

	return &ListSchemasResponse{
		Items:      page.Items,
		NextOffset: page.NextOffset,
		Total:      page.Total,
	}, nil
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

// Update updates an existing schema definition with patch semantics.
// Patch fields are merged into the existing definition.
func (s *SchemaService) Update(ctx context.Context, req *UpdateSchemaRequest) (*UpdateSchemaResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	existing, err := s.store.GetSchema(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	patch, err := unmarshalSchemaDef(req.Data, uri)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	// Merge patch fields into existing.
	if existing.Fields == nil {
		existing.Fields = make(map[string]schema.FieldDef)
	}
	for name, field := range patch.Fields {
		existing.Fields[name] = field
	}

	if patch.Mode != "" {
		existing.Mode = patch.Mode
	}

	if err := s.store.UpdateSchema(ctx, uri, existing); err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	return &UpdateSchemaResponse{Data: existing}, nil
}

// DeleteSchemaRequest is the request for schemas.delete.
type DeleteSchemaRequest struct {
	URI     string `json:"uri"`
	Cascade bool   `json:"cascade,omitempty"`
}

// DeleteSchemaResponse is the response for schemas.delete.
type DeleteSchemaResponse struct{}

// schemaDefPayload is the JSON-safe subset of [schema.Def] used for
// Create and Update requests. It avoids unmarshaling the URI field
// (which is provided separately in the request envelope).
type schemaDefPayload struct {
	Fields map[string]schema.FieldDef `json:"fields,omitempty"`
	Mode   schema.Mode                `json:"mode,omitempty"`
}

// unmarshalSchemaDef decodes a schema definition payload and attaches
// the given URI. This avoids Go's JSON decoder calling
// [core.URI.UnmarshalJSON] on a missing or null URI field.
func unmarshalSchemaDef(data json.RawMessage, uri *core.URI) (schema.Def, error) {
	var p schemaDefPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return schema.Def{}, err
	}
	mode := p.Mode
	if mode == "" {
		mode = schema.ModeStrict
	}

	// Normalize type identifiers (e.g. "string" → "STRING").
	for name, field := range p.Fields {
		tid, err := core.ParseType(string(field.Type))
		if err != nil {
			return schema.Def{}, err
		}
		field.Type = tid
		p.Fields[name] = field
	}

	return schema.Def{
		URI:    uri,
		Fields: p.Fields,
		Mode:   mode,
	}, nil
}

// Delete deletes a schema by URI.
// If the schema does not exist, the operation is treated as successful (idempotent).
// When Cascade is true, all records belonging to the schema are deleted first.
// If the store supports [store.BatchExecutor], cascade + delete runs atomically.
func (s *SchemaService) Delete(ctx context.Context, req *DeleteSchemaRequest) (*DeleteSchemaResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.delete: %w", err)
	}

	if req.Cascade {
		if cascadeErr := s.cascadeDelete(ctx, uri); cascadeErr != nil {
			return nil, fmt.Errorf("api: schemas.delete: %w", cascadeErr)
		}
		return &DeleteSchemaResponse{}, nil
	}

	err = s.store.DeleteSchema(ctx, uri)
	if errors.Is(err, store.ErrNotFound) {
		return &DeleteSchemaResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("api: schemas.delete: %w", err)
	}

	return &DeleteSchemaResponse{}, nil
}

// cascadeDelete deletes all records and the schema itself.
// Uses [store.BatchExecutor] for atomicity when available,
// otherwise falls back to sequential operations.
func (s *SchemaService) cascadeDelete(ctx context.Context, uri *core.URI) error {
	if batch, ok := s.store.(store.BatchExecutor); ok {
		return batch.ExecuteBatch(ctx, func(tx store.Store) error {
			if err := tx.DeleteSchemaRecords(ctx, uri); err != nil {
				return err
			}
			err := tx.DeleteSchema(ctx, uri)
			if errors.Is(err, store.ErrNotFound) {
				return nil
			}
			return err
		})
	}

	// Fallback: sequential (non-atomic).
	if err := s.store.DeleteSchemaRecords(ctx, uri); err != nil {
		return err
	}
	err := s.store.DeleteSchema(ctx, uri)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	return err
}
