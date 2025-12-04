package api

import (
	"context"
	"errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
)

type SchemaAPI struct {
	store driver.SchemaDriver
}

func NewSchemaAPI(store driver.SchemaDriver) *SchemaAPI {
	return &SchemaAPI{store: store}
}

func (a *SchemaAPI) PutSchema() EndpointFunc[PutSchemaRequest, PutSchemaResponse] {
	return func(ctx context.Context, req *PutSchemaRequest) (*PutSchemaResponse, error) {
		uri, err := core.ParseURI(req.URI)
		if err != nil {
			return nil, err
		}

		err = a.store.PutSchema(ctx, uri, req.Schema)
		if err != nil {
			return nil, err
		}

		return &PutSchemaResponse{URI: uri}, nil
	}
}

func (a *SchemaAPI) GetSchema() EndpointFunc[GetSchemaRequest, GetSchemaResponse] {
	return func(ctx context.Context, req *GetSchemaRequest) (*GetSchemaResponse, error) {
		uri, err := core.ParseURI(req.URI)
		if err != nil {
			return nil, err
		}

		def, err := a.store.GetSchema(ctx, uri)
		if err != nil {
			return nil, err
		}

		return &GetSchemaResponse{Schema: def}, nil
	}
}

func (a *SchemaAPI) ListSchemas() EndpointFunc[ListSchemasRequest, ListSchemasResponse] {
	return func(ctx context.Context, req *ListSchemasRequest) (*ListSchemasResponse, error) {
		uri, err := core.ParseURI(req.URI)
		if err != nil {
			return nil, err
		}

		defs, err := a.store.ListSchemas(ctx, uri)
		if err != nil {
			return nil, err
		}

		return &ListSchemasResponse{Schemas: defs}, nil
	}
}

func (a *SchemaAPI) DeleteSchema() EndpointFunc[DeleteSchemaRequest, DeleteSchemaResponse] {
	return func(ctx context.Context, req *DeleteSchemaRequest) (*DeleteSchemaResponse, error) {
		uri, err := core.ParseURI(req.URI)
		if err != nil {
			return nil, err
		}

		err = a.store.DeleteSchema(ctx, uri)
		if err != nil {
			return nil, err
		}

		return &DeleteSchemaResponse{}, nil
	}
}

type PutSchemaRequest struct {
	URI    string      `json:"uri"`
	Schema *schema.Def `json:"schema"`
}

func (req *PutSchemaRequest) Validate() error {
	if req.URI == "" {
		return errors.New("uri is required")
	}
	if req.Schema == nil {
		return errors.New("schema is required")
	}
	return nil
}

type PutSchemaResponse struct {
	URI *core.URI `json:"uri"`
}

type GetSchemaRequest struct {
	URI string `json:"uri"`
}

func (req *GetSchemaRequest) Validate() error {
	if req.URI == "" {
		return errors.New("uri is required")
	}
	return nil
}

type GetSchemaResponse struct {
	Schema *schema.Def `json:"schema"`
}

type ListSchemasRequest struct {
	URI string `json:"uri"`
}

func (req *ListSchemasRequest) Validate() error {
	if req.URI == "" {
		return errors.New("uri is required")
	}
	return nil
}

type ListSchemasResponse struct {
	Schemas []*schema.Def `json:"schemas"`
}

type DeleteSchemaRequest struct {
	URI string `json:"uri"`
}

func (req *DeleteSchemaRequest) Validate() error {
	if req.URI == "" {
		return errors.New("uri is required")
	}
	return nil
}

type DeleteSchemaResponse struct {}
