package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// RecordService provides record operations.
type RecordService struct {
	store   store.Store
	schemas store.SchemaStore
	tuples  store.TupleStore
	tx      store.TX
	events  *Bus
	encOpts []xdbjson.Option
}

// NewRecordService creates a [RecordService] backed by the given [store.Store].
func NewRecordService(s store.Store, opts ...ServiceOption) *RecordService {
	o := applyServiceOptions(opts)

	svc := &RecordService{
		store:   s,
		schemas: s,
		tuples:  s,
		encOpts: []xdbjson.Option{xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema()},
		events:  o.events,
	}
	if tx, ok := s.(store.TX); ok {
		svc.tx = tx
	}
	return svc
}

// publish emits a change notification when an event bus is wired.
// version is the record's version after the change, or 0 when unknown.
func (s *RecordService) publish(
	eventType, uri string,
	data json.RawMessage,
	version int64,
) {
	if s.events == nil {
		return
	}

	s.events.Publish(WatchEvent{
		Type:    eventType,
		URI:     uri,
		Data:    data,
		Version: version,
		TS:      time.Now(),
	})
}

// stored re-reads a record after a write. The version and timestamp the
// store stamps are not visible on the record the caller handed in, so
// responses and events are built from the stored state — otherwise a
// write response would disagree with a subsequent read.
func (s *RecordService) stored(
	ctx context.Context,
	uri *core.URI,
) (json.RawMessage, int64, error) {
	record, err := s.store.GetRecord(ctx, uri)
	if err != nil {
		return nil, 0, err
	}

	data, err := s.encode(record)
	if err != nil {
		return nil, 0, fmt.Errorf("[xdb/api] encode record: %w", err)
	}

	return data, recordVersion(record), nil
}

// recordVersion reads a record's stamped version, or 0 when absent.
func recordVersion(record *core.Record) int64 {
	tuple := record.Get(schema.FieldVersion)
	if tuple == nil {
		return 0
	}

	version, err := tuple.AsInt()
	if err != nil {
		return 0
	}

	return version
}

// CreateRecordRequest is the request for records.create.
type CreateRecordRequest struct {
	URI    string          `json:"uri"`
	Data   json.RawMessage `json:"data"`
	DryRun bool            `json:"dry_run,omitempty"`
}

// CreateRecordResponse is the response for records.create.
type CreateRecordResponse struct {
	DryRun *DryRunResult   `json:"dry_run,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// Create creates a new record. Identical re-create (same data) is an
// idempotent success; creating over an existing record with different
// data fails with [core.ErrConflict].
func (s *RecordService) Create(ctx context.Context, req *CreateRecordRequest) (*CreateRecordResponse, error) {
	uri, err := parseURI(req.URI, "records.create", 3, 3, false)
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

		if decErr := xdbjson.MarshalInto(req.Data, record, decOpts...); decErr != nil {
			return nil, decErr
		}
	}

	if req.DryRun {
		return s.dryRunCreate(ctx, uri, record)
	}

	err = s.store.CreateRecord(ctx, record)
	if errors.Is(err, core.ErrAlreadyExists) {
		existing, getErr := s.store.GetRecord(ctx, uri)
		if getErr != nil {
			return nil, getErr
		}

		equivalent, cmpErr := s.recordsEquivalent(record, existing)
		if cmpErr != nil {
			return nil, cmpErr
		}
		if !equivalent {
			return nil, createConflictError(uri)
		}

		return s.recordResponse(existing)
	}

	if err != nil {
		return nil, err
	}

	data, version, storedErr := s.stored(ctx, uri)
	if storedErr != nil {
		return nil, storedErr
	}
	s.publish("record.create", uri.String(), data, version)

	return &CreateRecordResponse{Data: data}, nil
}

// createConflictError reports a create over an existing record with
// different data.
func createConflictError(uri *core.URI) error {
	return fmt.Errorf(
		"records.create %s: record exists with different data "+
			"(use records.update to patch or records.upsert to replace): %w",
		uri, core.ErrConflict,
	)
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
	uri, err := parseURI(req.URI, "records.get", 3, 3, true)
	if err != nil {
		return nil, err
	}

	var record *core.Record
	if uri.Attr() != "" {
		tuple, tupleErr := s.tuples.GetTuple(ctx, uri)
		if tupleErr != nil {
			return nil, fmt.Errorf("[xdb/api] records.get %s: %w", uri, tupleErr)
		}
		record = core.NewRecord(uri.NS(), uri.Schema(), uri.ID())
		record.Set(tuple.Attr(), tuple.Value())
	} else {
		record, err = s.store.GetRecord(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("[xdb/api] records.get %s: %w", uri, err)
		}
	}

	var encOpts []xdbjson.Option
	if len(req.Fields) > 0 {
		encOpts = append(encOpts, xdbjson.WithFields(req.Fields...))
	}

	data, encErr := s.encode(record, encOpts...)
	if encErr != nil {
		return nil, fmt.Errorf("[xdb/api] encode record: %w", encErr)
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
	uri, err := parseURI(req.URI, "records.list", 1, 2, false)
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

	var encOpts []xdbjson.Option
	if len(req.Fields) > 0 {
		encOpts = append(encOpts, xdbjson.WithFields(req.Fields...))
	}

	items := make([]json.RawMessage, len(page.Items))
	for i, rec := range page.Items {
		data, encErr := s.encode(rec, encOpts...)
		if encErr != nil {
			return nil, fmt.Errorf("[xdb/api] encode record: %w", encErr)
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
	URI    string          `json:"uri"`
	Data   json.RawMessage `json:"data"`
	DryRun bool            `json:"dry_run,omitempty"`
}

// UpdateRecordResponse is the response for records.update.
type UpdateRecordResponse struct {
	DryRun *DryRunResult   `json:"dry_run,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// Update updates an existing record using patch semantics.
// The read and the write-back run inside a transaction when the store
// supports [store.TX]. Otherwise they run sequentially (non-atomic).
func (s *RecordService) Update(ctx context.Context, req *UpdateRecordRequest) (*UpdateRecordResponse, error) {
	uri, err := parseURI(req.URI, "records.update", 3, 3, false)
	if err != nil {
		return nil, err
	}

	if req.DryRun {
		return s.dryRunUpdate(ctx, req, uri)
	}

	err = runAtomic(ctx, s.tx, s.store, func(st store.Store) error {
		_, patchErr := applyRecordPatch(ctx, st, st, uri, req.Data)
		return patchErr
	})
	if err != nil {
		return nil, err
	}

	data, version, storedErr := s.stored(ctx, uri)
	if storedErr != nil {
		return nil, storedErr
	}

	s.publish("record.update", uri.String(), data, version)

	return &UpdateRecordResponse{Data: data}, nil
}

// deleteChecked runs del against the record at uri, enforcing want as an
// optimistic-concurrency precondition when it is non-zero. It returns
// the version the record held when it was removed, so the change event
// can be ordered against the writes before it.
//
// The check and the delete run in one transaction where the store
// supports it, so the precondition cannot go stale in between.
func (s *RecordService) deleteChecked(
	ctx context.Context,
	uri *core.URI,
	want int64,
	del func(store.Store) error,
) (int64, error) {
	var version int64

	err := runAtomic(ctx, s.tx, s.store, func(st store.Store) error {
		record, getErr := st.GetRecord(ctx, uri)
		if getErr != nil {
			return getErr
		}

		version = recordVersion(record)
		if want != 0 && want != version {
			return fmt.Errorf(
				"records.delete %s: record is at version %d, not %d: %w",
				uri, version, want, core.ErrConflict,
			)
		}

		return del(st)
	})

	return version, err
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

// applyRecordPatch fetches the record, applies the patch data to it,
// and writes it back through the given stores.
func applyRecordPatch(
	ctx context.Context,
	records store.RecordStore,
	schemas store.SchemaStore,
	uri *core.URI,
	data json.RawMessage,
) (*core.Record, error) {
	existing, err := mergeRecordPatch(ctx, records, schemas, uri, data)
	if err != nil {
		return nil, err
	}

	// The preceding GetRecord already enforced existence within the
	// same (transactional) scope, so the write-back is an upsert.
	if updateErr := records.UpsertRecord(ctx, existing); updateErr != nil {
		return nil, updateErr
	}

	return existing, nil
}

// mergeRecordPatch fetches the record and applies the patch data to
// it without writing anything back.
func mergeRecordPatch(
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

	if decErr := xdbjson.MarshalInto(data, existing, decOpts...); decErr != nil {
		return nil, decErr
	}

	return existing, nil
}

// UpsertRecordRequest is the request for records.upsert (full replace).
type UpsertRecordRequest struct {
	URI    string          `json:"uri"`
	Data   json.RawMessage `json:"data"`
	DryRun bool            `json:"dry_run,omitempty"`
}

// UpsertRecordResponse is the response for records.upsert.
type UpsertRecordResponse struct {
	DryRun *DryRunResult   `json:"dry_run,omitempty"`
	Data   json.RawMessage `json:"data"`
}

// Upsert creates or replaces a record.
func (s *RecordService) Upsert(ctx context.Context, req *UpsertRecordRequest) (*UpsertRecordResponse, error) {
	uri, err := parseURI(req.URI, "records.upsert", 3, 3, false)
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

		if decErr := xdbjson.MarshalInto(req.Data, record, decOpts...); decErr != nil {
			return nil, decErr
		}
	}

	if req.DryRun {
		return s.dryRunUpsert(ctx, uri, record)
	}

	if upsertErr := s.store.UpsertRecord(ctx, record); upsertErr != nil {
		return nil, upsertErr
	}

	data, version, storedErr := s.stored(ctx, uri)
	if storedErr != nil {
		return nil, storedErr
	}

	s.publish("record.upsert", uri.String(), data, version)

	return &UpsertRecordResponse{Data: data}, nil
}

// DeleteRecordRequest is the request for records.delete.
//
// Version is an optional optimistic-concurrency precondition: the delete
// proceeds only if the record is at that version, and fails with
// [core.ErrConflict] otherwise. Zero deletes unconditionally. Every
// other write verb carries its precondition inside the record payload;
// delete has none, so it takes one here.
type DeleteRecordRequest struct {
	URI     string `json:"uri"`
	Version int64  `json:"version,omitempty"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

// DeleteRecordResponse is the response for records.delete.
type DeleteRecordResponse struct {
	DryRun *DryRunResult `json:"dry_run,omitempty"`
}

// Delete deletes a record by URI. An attr-level URI
// (xdb://ns/schema/id#attr) deletes just that tuple. A whole-record
// delete is idempotent: a missing record is a success. An attr-level
// delete requires the record to exist and returns [core.ErrNotFound]
// otherwise. A missing attr on an existing record is a success.
func (s *RecordService) Delete(ctx context.Context, req *DeleteRecordRequest) (*DeleteRecordResponse, error) {
	uri, err := parseURI(req.URI, "records.delete", 3, 3, true)
	if err != nil {
		return nil, err
	}

	if req.DryRun {
		return s.dryRunDelete(ctx, uri)
	}

	if uri.Attr() != "" {
		version, delErr := s.deleteChecked(ctx, uri.RecordURI(), req.Version,
			func(st store.Store) error { return st.DeleteTuples(ctx, uri) },
		)
		if delErr != nil {
			return nil, delErr
		}
		s.publish("record.delete", uri.String(), nil, version)
		return &DeleteRecordResponse{}, nil
	}

	version, err := s.deleteChecked(ctx, uri, req.Version,
		func(st store.Store) error { return st.DeleteRecord(ctx, uri) },
	)
	if errors.Is(err, core.ErrNotFound) {
		return &DeleteRecordResponse{}, nil
	}

	if err != nil {
		return nil, err
	}

	s.publish("record.delete", uri.String(), nil, version)

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

	def, err := schemas.GetSchema(ctx, uri.SchemaURI())
	switch {
	case err == nil && def != nil:
		opts = append(opts, xdbjson.WithDef(def))
	case errors.Is(err, core.ErrNotFound):
		// No schema: the record is schema-free, so skip type coercion.
	case err != nil:
		return nil, fmt.Errorf("[xdb/api] lookup schema %s: %w", uri, err)
	}

	return opts, nil
}

// encode renders a record as a JSON document using the service's base options
// plus extra.
func (s *RecordService) encode(rec *core.Record, extra ...xdbjson.Option) ([]byte, error) {
	opts := make([]xdbjson.Option, 0, len(s.encOpts)+len(extra))
	opts = append(opts, s.encOpts...)
	opts = append(opts, extra...)

	return xdbjson.Unmarshal(rec, opts...)
}

// recordsEquivalent reports whether a and b encode to the same canonical
// JSON representation. It distinguishes an idempotent re-create (same
// data) from a conflicting one (different data) on [core.ErrAlreadyExists].
// The system attrs that the store derives are excluded from the comparison.
func (s *RecordService) recordsEquivalent(a, b *core.Record) (bool, error) {
	aData, err := s.encode(a)
	if err != nil {
		return false, err
	}

	bData, err := s.encode(b)
	if err != nil {
		return false, err
	}

	var am, bm map[string]any
	if err := json.Unmarshal(aData, &am); err != nil {
		return false, err
	}
	if err := json.Unmarshal(bData, &bm); err != nil {
		return false, err
	}

	// The store derives these on every write, so a caller's payload
	// never carries them and they say nothing about whether two records
	// hold the same facts.
	for _, m := range []map[string]any{am, bm} {
		delete(m, schema.FieldVersion)
		delete(m, schema.FieldUpdated)
	}

	return reflect.DeepEqual(am, bm), nil
}

// recordResponse encodes a [core.Record] into a [CreateRecordResponse].
func (s *RecordService) recordResponse(rec *core.Record) (*CreateRecordResponse, error) {
	data, err := s.encode(rec)
	if err != nil {
		return nil, fmt.Errorf("[xdb/api] encode record: %w", err)
	}

	return &CreateRecordResponse{Data: data}, nil
}
