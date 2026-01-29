package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/store"
)

var (
	recordEncoder = xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})
	recordDecoder = xdbjson.NewDecoder(xdbjson.Options{})
)

// RecordJSON wraps a core.Record with xdbjson serialization.
type RecordJSON struct {
	record *core.Record
}

// NewRecordJSON creates a new RecordJSON wrapper.
func NewRecordJSON(r *core.Record) *RecordJSON {
	return &RecordJSON{record: r}
}

// Record returns the underlying core.Record.
func (r *RecordJSON) Record() *core.Record {
	return r.record
}

// MarshalJSON implements the json.Marshaler interface using xdbjson.
func (r *RecordJSON) MarshalJSON() ([]byte, error) {
	return recordEncoder.FromRecord(r.record)
}

// UnmarshalJSON implements the json.Unmarshaler interface using xdbjson.
func (r *RecordJSON) UnmarshalJSON(data []byte) error {
	record, err := recordDecoder.ToRecord(data)
	if err != nil {
		return err
	}
	r.record = record
	return nil
}

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
	Records  []*RecordJSON `json:"records"`
	NotFound []string      `json:"not_found,omitempty"`
}

// GetRecords retrieves records by URIs.
func (a *RecordAPI) GetRecords() EndpointFunc[GetRecordsRequest, GetRecordsResponse] {
	return func(ctx context.Context, req *GetRecordsRequest) (*GetRecordsResponse, error) {
		uris := make([]*core.URI, len(req.URIs))
		for i, uriStr := range req.URIs {
			uri, err := core.ParseURI(uriStr)
			if err != nil {
				return nil, err
			}
			uris[i] = uri
		}

		records, missing, err := a.store.GetRecords(ctx, uris)
		if err != nil {
			return nil, err
		}

		notFound := make([]string, len(missing))
		for i, uri := range missing {
			notFound[i] = uri.String()
		}

		jsonRecords := make([]*RecordJSON, len(records))
		for i, r := range records {
			jsonRecords[i] = NewRecordJSON(r)
		}

		return &GetRecordsResponse{
			Records:  jsonRecords,
			NotFound: notFound,
		}, nil
	}
}

// PutRecordsRequest is the request for PutRecords endpoint.
type PutRecordsRequest struct {
	Records []*RecordJSON `json:"records"`
}

// PutRecordsResponse is the response for PutRecords endpoint.
type PutRecordsResponse struct {
	Created []string `json:"created"`
	Updated []string `json:"updated"`
}

// PutRecords creates or updates records.
func (a *RecordAPI) PutRecords() EndpointFunc[PutRecordsRequest, PutRecordsResponse] {
	return func(ctx context.Context, req *PutRecordsRequest) (*PutRecordsResponse, error) {
		records := make([]*core.Record, len(req.Records))
		for i, r := range req.Records {
			records[i] = r.Record()
		}

		err := a.store.PutRecords(ctx, records)
		if err != nil {
			return nil, err
		}

		uris := make([]string, len(records))
		for i, record := range records {
			uris[i] = record.URI().String()
		}

		return &PutRecordsResponse{
			Created: uris,
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
		uris := make([]*core.URI, len(req.URIs))
		for i, uriStr := range req.URIs {
			uri, err := core.ParseURI(uriStr)
			if err != nil {
				return nil, err
			}
			uris[i] = uri
		}

		err := a.store.DeleteRecords(ctx, uris)
		if err != nil {
			return nil, err
		}

		return &DeleteRecordsResponse{
			Deleted: req.URIs,
		}, nil
	}
}
