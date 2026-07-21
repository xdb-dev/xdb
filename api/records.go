package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/store"
)

// RecordService provides record operations.
type RecordService struct {
	store   store.Store
	schemas store.SchemaStore
	tuples  store.TupleStore
	tx      store.TX // nil when the store does not support transactions
	enc     *xdbjson.Encoder
}

// NewRecordService creates a [RecordService] backed by the given [store.Store].
func NewRecordService(s store.Store) *RecordService {
	svc := &RecordService{
		store:   s,
		schemas: s,
		tuples:  s,
		enc:     xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema()),
	}
	if tx, ok := s.(store.TX); ok {
		svc.tx = tx
	}
	return svc
}

// CreateRecordRequest is the request for records.create.
type CreateRecordRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// CreateRecordResponse is the response for records.create.
type CreateRecordResponse struct {
	Data json.RawMessage `json:"data"`
}

// Create creates a new record. Idempotent: returns existing if already exists.
func (s *RecordService) Create(ctx context.Context, req *CreateRecordRequest) (*CreateRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(
		uri.NS(),
		uri.Schema(),
		uri.ID(),
	)

	if len(req.Data) > 0 {
		decOpts, optsErr := decoderOpts(ctx, s.schemas, uri)
		if optsErr != nil {
			return nil, optsErr
		}

		dec := xdbjson.NewDecoder(decOpts...)
		if decErr := dec.ToExistingRecord(req.Data, record); decErr != nil {
			return nil, decErr
		}
	}

	err = s.store.CreateRecord(ctx, record)
	if errors.Is(err, core.ErrAlreadyExists) {
		existing, getErr := s.store.GetRecord(ctx, uri)
		if getErr != nil {
			return nil, getErr
		}

		return s.recordResponse(existing)
	}

	if err != nil {
		return nil, err
	}

	return s.recordResponse(record)
}

// GetRecordRequest is the request for records.get.
type GetRecordRequest struct {
	URI    string   `json:"uri"`
	Fields []string `json:"fields,omitempty"`
}

// GetRecordResponse is the response for records.get.
type GetRecordResponse struct {
	Data json.RawMessage `json:"data"`
}

// Get retrieves a single record by URI. An attr-level URI
// (xdb://ns/schema/id#attr) retrieves just that tuple.
func (s *RecordService) Get(ctx context.Context, req *GetRecordRequest) (*GetRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	var record *core.Record
	if uri.Attr() != "" {
		tuple, tupleErr := s.tuples.GetTuple(ctx, uri)
		if tupleErr != nil {
			return nil, fmt.Errorf("api: records.get %s: %w", uri, tupleErr)
		}
		record = core.NewRecord(uri.NS(), uri.Schema(), uri.ID())
		record.Set(tuple.Attr(), tuple.Value())
	} else {
		record, err = s.store.GetRecord(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("api: records.get %s: %w", uri, err)
		}
	}

	var encOpts []xdbjson.EncodeOption
	if len(req.Fields) > 0 {
		encOpts = append(encOpts, xdbjson.WithFields(req.Fields...))
	}

	data, encErr := s.enc.FromRecord(record, encOpts...)
	if encErr != nil {
		return nil, fmt.Errorf("api: encode record: %w", encErr)
	}

	return &GetRecordResponse{Data: data}, nil
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
	Items      []json.RawMessage `json:"items"`
	NextOffset int               `json:"next_offset,omitempty"`
	Total      int               `json:"total"`
}

// List lists records matching the given query.
func (s *RecordService) List(ctx context.Context, req *ListRecordsRequest) (*ListRecordsResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	query := &store.Query{
		URI:    uri,
		Filter: req.Filter,
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	page, err := s.store.ListRecords(ctx, query)
	if err != nil {
		return nil, err
	}

	var encOpts []xdbjson.EncodeOption
	if len(req.Fields) > 0 {
		encOpts = append(encOpts, xdbjson.WithFields(req.Fields...))
	}

	items := make([]json.RawMessage, len(page.Items))
	for i, rec := range page.Items {
		data, encErr := s.enc.FromRecord(rec, encOpts...)
		if encErr != nil {
			return nil, fmt.Errorf("api: encode record: %w", encErr)
		}

		items[i] = data
	}

	return &ListRecordsResponse{
		Items:      items,
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
	Data json.RawMessage `json:"data"`
}

// Update updates an existing record using patch semantics.
// The read-merge-write runs inside a transaction when the store
// supports [store.TX]; otherwise it falls back to sequential
// (non-atomic) operations.
func (s *RecordService) Update(ctx context.Context, req *UpdateRecordRequest) (*UpdateRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	var updated *core.Record
	err = runAtomic(ctx, s.tx, s.store, func(st store.Store) error {
		var patchErr error
		updated, patchErr = applyRecordPatch(ctx, st, st, uri, req.Data)
		return patchErr
	})
	if err != nil {
		return nil, err
	}

	data, encErr := s.enc.FromRecord(updated)
	if encErr != nil {
		return nil, fmt.Errorf("api: encode record: %w", encErr)
	}

	return &UpdateRecordResponse{Data: data}, nil
}

// runAtomic performs a read-modify-write transactionally when the
// store supports [store.TX], else directly against seq (non-atomic
// fallback). It centralizes the atomic-or-sequential choice so
// read-modify-write verbs don't each branch on it.
func runAtomic(
	ctx context.Context,
	tx store.TX,
	seq store.Store,
	fn func(store.Store) error,
) error {
	if tx != nil {
		return tx.Run(ctx, fn)
	}
	return fn(seq)
}

// applyRecordPatch fetches the record, merges the patch data into it,
// and writes it back through the given stores.
func applyRecordPatch(
	ctx context.Context,
	records store.RecordStore,
	schemas store.SchemaStore,
	uri *core.URI,
	data json.RawMessage,
) (*core.Record, error) {
	existing, err := records.GetRecord(ctx, uri)
	if err != nil {
		return nil, err
	}

	decOpts, optsErr := decoderOpts(ctx, schemas, uri)
	if optsErr != nil {
		return nil, optsErr
	}

	dec := xdbjson.NewDecoder(decOpts...)
	if decErr := dec.ToExistingRecord(data, existing); decErr != nil {
		return nil, decErr
	}

	// The preceding GetRecord already enforced existence within the
	// same (transactional) scope, so the write-back is an upsert.
	if updateErr := records.UpsertRecord(ctx, existing); updateErr != nil {
		return nil, updateErr
	}

	return existing, nil
}

// UpsertRecordRequest is the request for records.upsert (full replace).
type UpsertRecordRequest struct {
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data"`
}

// UpsertRecordResponse is the response for records.upsert.
type UpsertRecordResponse struct {
	Data json.RawMessage `json:"data"`
}

// Upsert creates or replaces a record.
func (s *RecordService) Upsert(ctx context.Context, req *UpsertRecordRequest) (*UpsertRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(
		uri.NS(),
		uri.Schema(),
		uri.ID(),
	)

	if len(req.Data) > 0 {
		decOpts, optsErr := decoderOpts(ctx, s.schemas, uri)
		if optsErr != nil {
			return nil, optsErr
		}

		dec := xdbjson.NewDecoder(decOpts...)
		if decErr := dec.ToExistingRecord(req.Data, record); decErr != nil {
			return nil, decErr
		}
	}

	if upsertErr := s.store.UpsertRecord(ctx, record); upsertErr != nil {
		return nil, upsertErr
	}

	data, encErr := s.enc.FromRecord(record)
	if encErr != nil {
		return nil, fmt.Errorf("api: encode record: %w", encErr)
	}

	return &UpsertRecordResponse{Data: data}, nil
}

// DeleteRecordRequest is the request for records.delete.
type DeleteRecordRequest struct {
	URI string `json:"uri"`
}

// DeleteRecordResponse is the response for records.delete.
type DeleteRecordResponse struct{}

// Delete deletes a record by URI. An attr-level URI
// (xdb://ns/schema/id#attr) deletes just that tuple. Idempotent:
// succeeds even if not found.
func (s *RecordService) Delete(ctx context.Context, req *DeleteRecordRequest) (*DeleteRecordResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, err
	}

	if uri.Attr() != "" {
		if deleteErr := s.tuples.DeleteTuples(ctx, uri); deleteErr != nil {
			return nil, deleteErr
		}
		return &DeleteRecordResponse{}, nil
	}

	err = s.store.DeleteRecord(ctx, uri)
	if errors.Is(err, core.ErrNotFound) {
		return &DeleteRecordResponse{}, nil
	}

	if err != nil {
		return nil, err
	}

	return &DeleteRecordResponse{}, nil
}

// decoderOpts returns decoder options for the given URI, including the schema
// definition for type-aware decoding when a schema exists.
// Returns an error if the schema lookup fails for reasons other than not-found.
func decoderOpts(
	ctx context.Context,
	schemas store.SchemaStore,
	uri *core.URI,
) ([]xdbjson.Option, error) {
	opts := []xdbjson.Option{
		xdbjson.WithNS(uri.NS()),
		xdbjson.WithSchema(uri.Schema()),
	}

	def, err := schemas.GetSchema(ctx, uri)
	switch {
	case err == nil && def != nil:
		opts = append(opts, xdbjson.WithDef(def))
	case errors.Is(err, core.ErrNotFound):
		// No schema — flexible mode, skip type coercion.
	case err != nil:
		return nil, fmt.Errorf("api: lookup schema %s: %w", uri, err)
	}

	return opts, nil
}

// recordResponse encodes a [core.Record] into a [CreateRecordResponse].
func (s *RecordService) recordResponse(rec *core.Record) (*CreateRecordResponse, error) {
	data, err := s.enc.FromRecord(rec)
	if err != nil {
		return nil, fmt.Errorf("api: encode record: %w", err)
	}

	return &CreateRecordResponse{Data: data}, nil
}
