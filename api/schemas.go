package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// SchemaService provides schema operations.
type SchemaService struct {
	store store.Store
	tx    store.TX // nil when the store does not support transactions
}

// NewSchemaService creates a [SchemaService] backed by the given [store.Store].
func NewSchemaService(s store.Store) *SchemaService {
	svc := &SchemaService{store: s}
	if tx, ok := s.(store.TX); ok {
		svc.tx = tx
	}
	return svc
}

// CreateSchemaRequest is the request for schemas.create.
type CreateSchemaRequest struct {
	URI    string          `json:"uri"`
	Data   json.RawMessage `json:"data"`
	DryRun bool            `json:"dry_run,omitempty"`
}

// CreateSchemaResponse is the response for schemas.create.
type CreateSchemaResponse struct {
	Data   *schema.Def   `json:"data"`
	DryRun *DryRunResult `json:"dry_run,omitempty"`
}

// Create creates a new schema definition. Identical re-create (same
// definition) is an idempotent success; creating over an existing schema
// with a different definition fails with [core.ErrConflict].
func (s *SchemaService) Create(ctx context.Context, req *CreateSchemaRequest) (*CreateSchemaResponse, error) {
	uri, err := parseURI(req.URI, "schemas.create", 2, 2, false)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}

	def, err := unmarshalSchemaDef(req.Data, uri)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}
	if def.Mode == "" {
		def.Mode = schema.ModeStrict
	}

	if req.DryRun {
		return s.dryRunCreateSchema(ctx, uri, &def)
	}

	err = s.store.CreateSchema(ctx, uri, &def)
	if errors.Is(err, core.ErrAlreadyExists) {
		existing, getErr := s.store.GetSchema(ctx, uri)
		if getErr != nil {
			return nil, fmt.Errorf("api: schemas.create: %w", getErr)
		}

		equivalent, cmpErr := schemasEquivalent(&def, existing)
		if cmpErr != nil {
			return nil, fmt.Errorf("api: schemas.create: %w", cmpErr)
		}
		if !equivalent {
			return nil, schemaCreateConflictError(uri)
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
	uri, err := parseURI(req.URI, "schemas.get", 2, 2, false)
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
	uri, err := parseURI(req.URI, "schemas.list", 1, 1, false)
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
// Patch fields (including Items) are added or replaced in the existing
// definition; field removal is not supported. A non-zero Revision in the
// patch is checked as an optimistic-concurrency CAS against the store's
// current revision ([core.ErrConflict] on mismatch); omitting it (zero)
// updates unconditionally.
// The read-merge-write runs inside a transaction when the store
// supports [store.TX]; otherwise it falls back to sequential
// (non-atomic) operations.
func (s *SchemaService) Update(ctx context.Context, req *UpdateSchemaRequest) (*UpdateSchemaResponse, error) {
	uri, err := parseURI(req.URI, "schemas.update", 2, 2, false)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	var updated *schema.Def
	err = runAtomic(ctx, s.tx, s.store, func(st store.Store) error {
		var patchErr error
		updated, patchErr = applySchemaPatch(ctx, st, uri, req.Data)
		return patchErr
	})
	if err != nil {
		return nil, fmt.Errorf("api: schemas.update: %w", err)
	}

	return &UpdateSchemaResponse{Data: updated}, nil
}

// applySchemaPatch fetches the schema, merges the patch data into it,
// and writes it back through the given store.
func applySchemaPatch(
	ctx context.Context,
	schemas store.SchemaStore,
	uri *core.URI,
	data json.RawMessage,
) (*schema.Def, error) {
	existing, err := schemas.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	patch, err := unmarshalSchemaDef(data, uri)
	if err != nil {
		return nil, err
	}

	// Build the merged definition as a fresh value rather than mutating
	// existing in place: some stores (e.g. xdbmemory) return the same
	// *schema.Def pointer they hold internally, so an in-place edit would
	// corrupt the stored definition (in particular its Revision) before
	// the write path below re-validates it as a CAS base.
	merged := &schema.Def{
		URI:         uri,
		Fields:      make(map[string]schema.Field, len(existing.Fields)+len(patch.Fields)),
		Annotations: existing.Annotations,
		Mode:        existing.Mode,
		Description: existing.Description,
		Revision:    existing.Revision,
	}
	for name, field := range existing.Fields {
		merged.Fields[name] = field
	}
	// Fields (including Items) are added or replaced; removal is not
	// supported.
	for name, field := range patch.Fields {
		merged.Fields[name] = field
	}

	if patch.Mode != "" {
		merged.Mode = patch.Mode
	}

	// A non-zero patch revision is the caller's expected base revision:
	// CAS it against the store's current revision. Zero (omitted) keeps
	// the freshly fetched current revision, which is always accepted
	// (unconditional update).
	if patch.Revision != 0 {
		merged.Revision = patch.Revision
	}

	if err := schemas.UpdateSchema(ctx, uri, merged); err != nil {
		return nil, err
	}

	return merged, nil
}

// schemasEquivalent reports whether a and b are the same schema
// definition, ignoring Revision (which legitimately differs between a
// freshly built create payload and the currently stored definition).
// schemaCreateConflictError reports a create over an existing schema
// with a different definition.
func schemaCreateConflictError(uri *core.URI) error {
	return fmt.Errorf(
		"api: schemas.create %s: schema exists with a different definition "+
			"(run schemas.get to inspect; use schemas.update to evolve): %w",
		uri, core.ErrConflict,
	)
}

func schemasEquivalent(a, b *schema.Def) (bool, error) {
	am, err := schemaDefMap(a)
	if err != nil {
		return false, err
	}

	bm, err := schemaDefMap(b)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(am, bm), nil
}

// schemaDefMap encodes a [schema.Def] to its canonical JSON form and
// decodes it into a map, with the revision key removed, for comparison.
func schemaDefMap(d *schema.Def) (map[string]any, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	delete(m, "revision")

	return m, nil
}

// DeleteSchemaRequest is the request for schemas.delete.
type DeleteSchemaRequest struct {
	URI     string `json:"uri"`
	Cascade bool   `json:"cascade,omitempty"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

// DeleteSchemaResponse is the response for schemas.delete.
type DeleteSchemaResponse struct {
	DryRun *DryRunResult `json:"dry_run,omitempty"`
}

// schemaFieldPayload is the wire representation of a [schema.Field], mirroring
// the schema package's own JSON format ({type, elem_type, items, ...}).
// Items is recursive: it declares the element object schema for an
// ARRAY<JSON> field, using the same field payload shape one level deeper.
type schemaFieldPayload struct {
	Annotations map[string]string             `json:"annotations,omitempty"`
	Items       map[string]schemaFieldPayload `json:"items,omitempty"`
	Type        string                        `json:"type"`
	ElemType    string                        `json:"elem_type,omitempty"`
	Description string                        `json:"description,omitempty"`
	Required    bool                          `json:"required,omitempty"`
}

// schemaDefPayload is the JSON-safe subset of [schema.Def] used for
// Create and Update requests. It avoids unmarshaling the URI field
// (which is provided separately in the request envelope).
type schemaDefPayload struct {
	Fields      map[string]schemaFieldPayload `json:"fields,omitempty"`
	Annotations map[string]string             `json:"annotations,omitempty"`
	Mode        schema.Mode                   `json:"mode,omitempty"`
	Description string                        `json:"description,omitempty"`
	Revision    int64                         `json:"revision,omitempty"`
}

// unmarshalSchemaDef decodes a schema definition payload and attaches
// the given URI. This avoids Go's JSON decoder calling
// [core.URI.UnmarshalJSON] on a missing or null URI field. The Mode is
// left as provided (possibly empty); callers normalize as needed.
func unmarshalSchemaDef(data json.RawMessage, uri *core.URI) (schema.Def, error) {
	var p schemaDefPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return schema.Def{}, err
	}

	var fields map[string]schema.Field
	if len(p.Fields) > 0 {
		var err error
		fields, err = payloadFields(p.Fields)
		if err != nil {
			return schema.Def{}, err
		}
	}

	return schema.Def{
		URI:         uri,
		Fields:      fields,
		Mode:        p.Mode,
		Description: p.Description,
		Annotations: p.Annotations,
		Revision:    p.Revision,
	}, nil
}

// payloadFields converts a set of wire field payloads into [schema.Field]
// values, recursing into Items.
func payloadFields(payload map[string]schemaFieldPayload) (map[string]schema.Field, error) {
	fields := make(map[string]schema.Field, len(payload))
	for name, fp := range payload {
		field, err := payloadToField(fp)
		if err != nil {
			return nil, err
		}
		fields[name] = field
	}
	return fields, nil
}

// payloadToField reconstructs a [schema.Field] from a wire field payload,
// recursing into Items for ARRAY<JSON> member schemas.
func payloadToField(fp schemaFieldPayload) (schema.Field, error) {
	t, err := fieldPayloadType(fp)
	if err != nil {
		return schema.Field{}, err
	}

	field := schema.Field{
		Type:        t,
		Required:    fp.Required,
		Description: fp.Description,
		Annotations: fp.Annotations,
	}

	if len(fp.Items) > 0 {
		items, err := payloadFields(fp.Items)
		if err != nil {
			return schema.Field{}, err
		}
		field.Items = items
	}

	return field, nil
}

// fieldPayloadType reconstructs a [core.Type] from a wire field payload.
func fieldPayloadType(fp schemaFieldPayload) (core.Type, error) {
	tid, err := core.ParseType(fp.Type)
	if err != nil {
		return core.Type{}, err
	}
	if tid != core.TIDArray {
		return core.NewType(tid), nil
	}
	if fp.ElemType == "" {
		return core.NewArrayType(""), nil
	}
	elemTID, err := core.ParseType(fp.ElemType)
	if err != nil {
		return core.Type{}, err
	}
	return core.NewArrayType(elemTID), nil
}

// Delete deletes a schema by URI.
// If the schema does not exist, the operation is treated as successful (idempotent).
// When Cascade is true, all records belonging to the schema are deleted first.
// If the store supports [store.TX], cascade + delete runs atomically.
func (s *SchemaService) Delete(ctx context.Context, req *DeleteSchemaRequest) (*DeleteSchemaResponse, error) {
	uri, err := parseURI(req.URI, "schemas.delete", 2, 2, false)
	if err != nil {
		return nil, fmt.Errorf("api: schemas.delete: %w", err)
	}

	if req.DryRun {
		return s.dryRunDeleteSchema(ctx, uri)
	}

	if req.Cascade {
		if cascadeErr := s.cascadeDelete(ctx, uri); cascadeErr != nil {
			return nil, fmt.Errorf("api: schemas.delete: %w", cascadeErr)
		}
		return &DeleteSchemaResponse{}, nil
	}

	err = s.store.DeleteSchema(ctx, uri)
	if errors.Is(err, core.ErrNotFound) {
		return &DeleteSchemaResponse{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("api: schemas.delete: %w", err)
	}

	return &DeleteSchemaResponse{}, nil
}

// cascadeDelete deletes all records and the schema itself. Atomic when
// the store supports [store.TX], sequential (non-atomic) otherwise.
func (s *SchemaService) cascadeDelete(ctx context.Context, uri *core.URI) error {
	return runAtomic(ctx, s.tx, s.store, func(st store.Store) error {
		if err := st.DeleteSchemaRecords(ctx, uri); err != nil {
			return err
		}
		err := st.DeleteSchema(ctx, uri)
		if errors.Is(err, core.ErrNotFound) {
			return nil
		}
		return err
	})
}
