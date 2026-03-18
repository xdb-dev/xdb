package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/store"
)

// RecordService provides record operations.
type RecordService struct {
	store store.RecordStore
}

// NewRecordService creates a [RecordService] backed by the given [store.RecordStore].
func NewRecordService(s store.RecordStore) *RecordService {
	return &RecordService{store: s}
}

// CreateRecordRequest is the request for records.create.
type CreateRecordRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// CreateRecordResponse is the response for records.create.
type CreateRecordResponse struct {
	Data *core.Record `json:"data"`
}

// Create creates a new record. Idempotent: returns existing if already exists.
func (s *RecordService) Create(ctx context.Context, req *CreateRecordRequest) (*CreateRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String(),
	)

	if len(req.Data) > 0 {
		dec := xdbjson.NewDefaultDecoder(uri.NS().String(), uri.Schema().String())

		if decErr := dec.ToExistingRecord(req.Data, record); decErr != nil {
			return nil, decErr
		}
	}

	err = s.store.CreateRecord(ctx, record)
	if errors.Is(err, store.ErrAlreadyExists) {
		existing, getErr := s.store.GetRecord(ctx, uri)
		if getErr != nil {
			return nil, getErr
		}

		return &CreateRecordResponse{Data: existing}, nil
	}

	if err != nil {
		return nil, err
	}

	return &CreateRecordResponse{Data: record}, nil
}

// GetRecordRequest is the request for records.get.
type GetRecordRequest struct {
	URI    string   `json:"uri"`
	Fields []string `json:"fields,omitempty"`
}

// GetRecordResponse is the response for records.get.
type GetRecordResponse struct {
	Data *core.Record `json:"data"`
}

// Get retrieves a single record by URI.
func (s *RecordService) Get(ctx context.Context, req *GetRecordRequest) (*GetRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	record, err := s.store.GetRecord(ctx, uri)
	if err != nil {
		return nil, err
	}

	return &GetRecordResponse{Data: record}, nil
}

// ListRecordsRequest is the request for records.list.
type ListRecordsRequest struct {
	URI    string   `json:"uri"`
	Filter string   `json:"filter,omitempty"`
	Fields []string `json:"fields,omitempty"`
	Limit  int      `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`
}

// ListRecordsResponse is the response for records.list.
type ListRecordsResponse struct {
	Items      []*core.Record `json:"items"`
	NextOffset int            `json:"next_offset,omitempty"`
	Total      int            `json:"total"`
}

// List lists records matching the given query.
func (s *RecordService) List(ctx context.Context, req *ListRecordsRequest) (*ListRecordsResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	query := &store.ListQuery{
		Filter: req.Filter,
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	page, err := s.store.ListRecords(ctx, uri, query)
	if err != nil {
		return nil, err
	}

	return &ListRecordsResponse{
		Items:      page.Items,
		NextOffset: page.NextOffset,
		Total:      page.Total,
	}, nil
}

// UpdateRecordRequest is the request for records.update (patch semantics).
type UpdateRecordRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// UpdateRecordResponse is the response for records.update.
type UpdateRecordResponse struct {
	Data *core.Record `json:"data"`
}

// Update updates an existing record using patch semantics.
func (s *RecordService) Update(ctx context.Context, req *UpdateRecordRequest) (*UpdateRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	existing, err := s.store.GetRecord(ctx, uri)
	if err != nil {
		return nil, err
	}

	dec := xdbjson.NewDefaultDecoder(uri.NS().String(), uri.Schema().String())

	if decErr := dec.ToExistingRecord(req.Data, existing); decErr != nil {
		return nil, decErr
	}

	if updateErr := s.store.UpdateRecord(ctx, existing); updateErr != nil {
		return nil, updateErr
	}

	return &UpdateRecordResponse{Data: existing}, nil
}

// UpsertRecordRequest is the request for records.upsert (full replace).
type UpsertRecordRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// UpsertRecordResponse is the response for records.upsert.
type UpsertRecordResponse struct {
	Data *core.Record `json:"data"`
}

// Upsert creates or replaces a record.
func (s *RecordService) Upsert(ctx context.Context, req *UpsertRecordRequest) (*UpsertRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String(),
	)

	if len(req.Data) > 0 {
		dec := xdbjson.NewDefaultDecoder(uri.NS().String(), uri.Schema().String())

		if decErr := dec.ToExistingRecord(req.Data, record); decErr != nil {
			return nil, decErr
		}
	}

	if upsertErr := s.store.UpsertRecord(ctx, record); upsertErr != nil {
		return nil, upsertErr
	}

	return &UpsertRecordResponse{Data: record}, nil
}

// DeleteRecordRequest is the request for records.delete.
type DeleteRecordRequest struct {
	URI string `json:"uri"`
}

// DeleteRecordResponse is the response for records.delete.
type DeleteRecordResponse struct{}

// Delete deletes a record by URI. Idempotent: succeeds even if not found.
func (s *RecordService) Delete(ctx context.Context, req *DeleteRecordRequest) (*DeleteRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	err = s.store.DeleteRecord(ctx, uri)
	if errors.Is(err, store.ErrNotFound) {
		return &DeleteRecordResponse{}, nil
	}

	if err != nil {
		return nil, err
	}

	return &DeleteRecordResponse{}, nil
}
