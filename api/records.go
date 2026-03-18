package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/core"
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

// Create creates a new record.
func (s *RecordService) Create(_ context.Context, _ *CreateRecordRequest) (*CreateRecordResponse, error) {
	return nil, fmt.Errorf("api: records.create not implemented")
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
func (s *RecordService) Get(_ context.Context, _ *GetRecordRequest) (*GetRecordResponse, error) {
	return nil, fmt.Errorf("api: records.get not implemented")
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
func (s *RecordService) List(_ context.Context, _ *ListRecordsRequest) (*ListRecordsResponse, error) {
	return nil, fmt.Errorf("api: records.list not implemented")
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

// Update updates an existing record.
func (s *RecordService) Update(_ context.Context, _ *UpdateRecordRequest) (*UpdateRecordResponse, error) {
	return nil, fmt.Errorf("api: records.update not implemented")
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
func (s *RecordService) Upsert(_ context.Context, _ *UpsertRecordRequest) (*UpsertRecordResponse, error) {
	return nil, fmt.Errorf("api: records.upsert not implemented")
}

// DeleteRecordRequest is the request for records.delete.
type DeleteRecordRequest struct {
	URI string `json:"uri"`
}

// DeleteRecordResponse is the response for records.delete.
type DeleteRecordResponse struct{}

// Delete deletes a record by URI.
func (s *RecordService) Delete(_ context.Context, _ *DeleteRecordRequest) (*DeleteRecordResponse, error) {
	return nil, fmt.Errorf("api: records.delete not implemented")
}
