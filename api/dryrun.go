package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// DryRunResult reports the outcome of a validate-only request. It is
// present on a response only when the request set dry_run — clients use
// it as the marker that the daemon honored the flag.
type DryRunResult struct {
	Would string `json:"would"`
	Valid bool   `json:"valid"`
}

// validateRecord runs schema policy against the would-be write without
// writing. It requires the store to implement [store.Validator], which
// every facade-built store does.
func validateRecord(
	ctx context.Context,
	st store.Store,
	record *core.Record,
	op store.Op,
) error {
	v, ok := st.(store.Validator)
	if !ok {
		return fmt.Errorf("dry_run unsupported by this store: %w", core.ErrNotImplemented)
	}

	return v.ValidateRecord(ctx, record, op)
}

// dryRunCreate validates records.create without writing: a divergent
// existing record conflicts exactly like the real call, an identical
// one is a noop, and a missing one validates as a create.
func (s *RecordService) dryRunCreate(
	ctx context.Context,
	uri *core.URI,
	record *core.Record,
) (*CreateRecordResponse, error) {
	existing, err := s.store.GetRecord(ctx, uri)

	switch {
	case err == nil:
		equivalent, cmpErr := s.recordsEquivalent(record, existing)
		if cmpErr != nil {
			return nil, cmpErr
		}
		if !equivalent {
			return nil, createConflictError(uri)
		}

		resp, respErr := s.recordResponse(existing)
		if respErr != nil {
			return nil, respErr
		}
		resp.DryRun = &DryRunResult{Valid: true, Would: "noop"}

		return resp, nil

	case errors.Is(err, core.ErrNotFound):
		if vErr := validateRecord(ctx, s.store, record, store.OpCreate); vErr != nil {
			return nil, vErr
		}

		resp, respErr := s.recordResponse(record)
		if respErr != nil {
			return nil, respErr
		}
		resp.DryRun = &DryRunResult{Valid: true, Would: "create"}

		return resp, nil

	default:
		return nil, err
	}
}

// dryRunUpdate validates records.update without writing: the record
// must exist, and the patched result must satisfy the schema.
func (s *RecordService) dryRunUpdate(
	ctx context.Context,
	req *UpdateRecordRequest,
	uri *core.URI,
) (*UpdateRecordResponse, error) {
	merged, err := mergeRecordPatch(ctx, s.store, s.schemas, uri, req.Data)
	if err != nil {
		return nil, err
	}

	if vErr := validateRecord(ctx, s.store, merged, store.OpPut); vErr != nil {
		return nil, vErr
	}

	data, encErr := s.encode(merged)
	if encErr != nil {
		return nil, fmt.Errorf("api: encode record: %w", encErr)
	}

	return &UpdateRecordResponse{
		Data:   data,
		DryRun: &DryRunResult{Valid: true, Would: "update"},
	}, nil
}

// dryRunUpsert validates records.upsert without writing, reporting
// whether the real call would create or replace.
func (s *RecordService) dryRunUpsert(
	ctx context.Context,
	uri *core.URI,
	record *core.Record,
) (*UpsertRecordResponse, error) {
	if vErr := validateRecord(ctx, s.store, record, store.OpPut); vErr != nil {
		return nil, vErr
	}

	would := "replace"
	_, err := s.store.GetRecord(ctx, uri)
	if errors.Is(err, core.ErrNotFound) {
		would = "create"
	} else if err != nil {
		return nil, err
	}

	data, encErr := s.encode(record)
	if encErr != nil {
		return nil, fmt.Errorf("api: encode record: %w", encErr)
	}

	return &UpsertRecordResponse{
		Data:   data,
		DryRun: &DryRunResult{Valid: true, Would: would},
	}, nil
}

// dryRunDelete validates records.delete without deleting, reporting
// whether the real call would delete anything.
func (s *RecordService) dryRunDelete(
	ctx context.Context,
	uri *core.URI,
) (*DeleteRecordResponse, error) {
	v, ok := s.store.(store.Validator)
	if !ok {
		return nil, fmt.Errorf("dry_run unsupported by this store: %w", core.ErrNotImplemented)
	}

	if vErr := v.ValidateDeleteRecord(ctx, uri); vErr != nil {
		return nil, vErr
	}

	would := "delete"
	_, err := s.store.GetRecord(ctx, uri.RecordURI())
	if errors.Is(err, core.ErrNotFound) {
		would = "noop"
	} else if err != nil {
		return nil, err
	}

	return &DeleteRecordResponse{
		DryRun: &DryRunResult{Valid: true, Would: would},
	}, nil
}

// dryRunCreateSchema validates schemas.create without writing.
func (s *SchemaService) dryRunCreateSchema(
	ctx context.Context,
	uri *core.URI,
	def *schema.Def,
) (*CreateSchemaResponse, error) {
	if vErr := def.Validate(); vErr != nil {
		return nil, fmt.Errorf("api: schemas.create: %w: %w", core.ErrSchemaViolation, vErr)
	}

	existing, err := s.store.GetSchema(ctx, uri)

	switch {
	case err == nil:
		equivalent, cmpErr := schemasEquivalent(def, existing)
		if cmpErr != nil {
			return nil, fmt.Errorf("api: schemas.create: %w", cmpErr)
		}
		if !equivalent {
			return nil, schemaCreateConflictError(uri)
		}

		return &CreateSchemaResponse{
			Data:   existing,
			DryRun: &DryRunResult{Valid: true, Would: "noop"},
		}, nil

	case errors.Is(err, core.ErrNotFound):
		return &CreateSchemaResponse{
			Data:   def,
			DryRun: &DryRunResult{Valid: true, Would: "create"},
		}, nil

	default:
		return nil, fmt.Errorf("api: schemas.create: %w", err)
	}
}

// dryRunDeleteSchema validates schemas.delete without deleting.
func (s *SchemaService) dryRunDeleteSchema(
	ctx context.Context,
	uri *core.URI,
) (*DeleteSchemaResponse, error) {
	would := "delete"
	_, err := s.store.GetSchema(ctx, uri)
	if errors.Is(err, core.ErrNotFound) {
		would = "noop"
	} else if err != nil {
		return nil, fmt.Errorf("api: schemas.delete: %w", err)
	}

	return &DeleteSchemaResponse{
		DryRun: &DryRunResult{Valid: true, Would: would},
	}, nil
}
