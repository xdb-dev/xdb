package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// RecordAPI provides HTTP endpoints for record operations.
type RecordAPI struct {
	store store.RecordStore
}

// NewRecordAPI creates a new RecordAPI.
func NewRecordAPI(store store.RecordStore) *RecordAPI {
	return &RecordAPI{store: store}
}

// GetRecordsRequest is the request for GetRecords endpoint.
type GetRecordsRequest struct {
	URIs []string `json:"uris"`
}

// GetRecordsResponse is the response for GetRecords endpoint.
type GetRecordsResponse struct {
	Records  []*core.Record `json:"records"`
	NotFound []string       `json:"not_found,omitempty"`
}

// GetRecords retrieves records by URIs.
func (a *RecordAPI) GetRecords() EndpointFunc[GetRecordsRequest, GetRecordsResponse] {
	return func(ctx context.Context, req *GetRecordsRequest) (*GetRecordsResponse, error) {
		// TODO: Implement in future iteration
		return &GetRecordsResponse{
			Records: []*core.Record{},
		}, nil
	}
}

// PutRecordsRequest is the request for PutRecords endpoint.
type PutRecordsRequest struct {
	Records []*core.Record `json:"records"`
}

// PutRecordsResponse is the response for PutRecords endpoint.
type PutRecordsResponse struct {
	Created []string `json:"created"`
	Updated []string `json:"updated"`
}

// PutRecords creates or updates records.
func (a *RecordAPI) PutRecords() EndpointFunc[PutRecordsRequest, PutRecordsResponse] {
	return func(ctx context.Context, req *PutRecordsRequest) (*PutRecordsResponse, error) {
		// TODO: Implement in future iteration
		return &PutRecordsResponse{
			Created: []string{},
		}, nil
	}
}

// DeleteRecordsRequest is the request for DeleteRecords endpoint.
type DeleteRecordsRequest struct {
	URIs []string `json:"uris"`
}

// DeleteRecordsResponse is the response for DeleteRecords endpoint.
type DeleteRecordsResponse struct {
	Deleted []string `json:"deleted"`
}

// DeleteRecords deletes records by URIs.
func (a *RecordAPI) DeleteRecords() EndpointFunc[DeleteRecordsRequest, DeleteRecordsResponse] {
	return func(ctx context.Context, req *DeleteRecordsRequest) (*DeleteRecordsResponse, error) {
		// TODO: Implement in future iteration
		return &DeleteRecordsResponse{
			Deleted: []string{},
		}, nil
	}
}
