package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/store"
)

// BatchService provides batch operations.
type BatchService struct {
	store  store.Store
	tx     store.TX // nil when the store does not support transactions
	events *Bus     // nil when change notifications are disabled
}

// NewBatchService creates a [BatchService] backed by the given [store.Store].
func NewBatchService(s store.Store, opts ...ServiceOption) *BatchService {
	o := applyServiceOptions(opts)

	svc := &BatchService{store: s, events: o.events}
	if tx, ok := s.(store.TX); ok {
		svc.tx = tx
	}

	return svc
}

// publishOp emits a change notification for one committed batch op.
// Batch ops run through event-less services (writes inside a
// transaction must not publish before commit), so the batch publishes
// itself after the commit.
func (s *BatchService) publishOp(o BatchOperation) {
	if s.events == nil {
		return
	}

	resource, action, ok := strings.Cut(o.Op, ".")
	if !ok {
		return
	}

	s.events.Publish(WatchEvent{
		Type: strings.TrimSuffix(resource, "s") + "." + action,
		URI:  o.URI,
		TS:   time.Now(),
	})
}

// BatchOperation is one operation in a batch: a dotted resource.action
// op name, the target URI, and the JSON payload for write ops.
type BatchOperation struct {
	Op   string          `json:"op"`
	URI  string          `json:"uri"`
	Data json.RawMessage `json:"data,omitempty"`
}

// BatchResult reports the outcome of one batch operation.
type BatchResult struct {
	Error  *rpc.Error    `json:"error,omitempty"`
	DryRun *DryRunResult `json:"dry_run,omitempty"`
	URI    string        `json:"uri"`
	Status string        `json:"status"`
	Index  int           `json:"index"`
}

// ExecuteBatchRequest is the request for batch.execute.
type ExecuteBatchRequest struct {
	Operations []BatchOperation `json:"operations"`
	DryRun     bool             `json:"dry_run,omitempty"`
	NonAtomic  bool             `json:"non_atomic,omitempty"`
}

// ExecuteBatchResponse is the response for batch.execute.
type ExecuteBatchResponse struct {
	Results    []BatchResult `json:"results"`
	Total      int           `json:"total"`
	Succeeded  int           `json:"succeeded"`
	Failed     int           `json:"failed"`
	RolledBack bool          `json:"rolled_back,omitempty"`
}

// batchDispatch runs one operation through the appropriate service.
type batchDispatch func(
	ctx context.Context,
	records *RecordService,
	schemas *SchemaService,
	o BatchOperation,
	dryRun bool,
) (*DryRunResult, error)

// batchOps maps every allowed op name to its dispatch function.
var batchOps = map[string]batchDispatch{
	"records.create": func(ctx context.Context, records *RecordService, _ *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := records.Create(ctx, &CreateRecordRequest{URI: o.URI, Data: o.Data, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
	"records.update": func(ctx context.Context, records *RecordService, _ *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := records.Update(ctx, &UpdateRecordRequest{URI: o.URI, Data: o.Data, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
	"records.upsert": func(ctx context.Context, records *RecordService, _ *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := records.Upsert(ctx, &UpsertRecordRequest{URI: o.URI, Data: o.Data, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
	"records.delete": func(ctx context.Context, records *RecordService, _ *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := records.Delete(ctx, &DeleteRecordRequest{URI: o.URI, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
	"schemas.create": func(ctx context.Context, _ *RecordService, schemas *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := schemas.Create(ctx, &CreateSchemaRequest{URI: o.URI, Data: o.Data, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
	"schemas.update": func(ctx context.Context, _ *RecordService, schemas *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		if dryRun {
			return nil, fmt.Errorf("%w: dry_run for schemas.update", core.ErrNotImplemented)
		}
		_, err := schemas.Update(ctx, &UpdateSchemaRequest{URI: o.URI, Data: o.Data})
		return nil, err
	},
	"schemas.delete": func(ctx context.Context, _ *RecordService, schemas *SchemaService, o BatchOperation, dryRun bool) (*DryRunResult, error) {
		resp, err := schemas.Delete(ctx, &DeleteSchemaRequest{URI: o.URI, DryRun: dryRun})
		if err != nil {
			return nil, err
		}
		return resp.DryRun, nil
	},
}

// allowedBatchOps returns the sorted op vocabulary for error messages.
func allowedBatchOps() string {
	names := make([]string, 0, len(batchOps))
	for name := range batchOps {
		names = append(names, name)
	}
	slices.Sort(names)

	return strings.Join(names, ", ")
}

// errBatchAborted aborts the batch transaction after a per-op failure
// so the driver rolls back; the failure itself is reported per-op.
var errBatchAborted = errors.New("[xdb/api] batch aborted")

// Execute runs a batch of operations. On transactional backends the
// batch is atomic: any failure rolls back every operation. On
// non-transactional backends it refuses unless non_atomic is set, in
// which case operations run sequentially with per-op error attribution.
func (s *BatchService) Execute(ctx context.Context, req *ExecuteBatchRequest) (*ExecuteBatchResponse, error) {
	if len(req.Operations) == 0 {
		return nil, rpc.InvalidParams("batch requires at least one operation")
	}

	for i, o := range req.Operations {
		if _, ok := batchOps[o.Op]; !ok {
			return nil, rpc.InvalidParams(fmt.Sprintf(
				"batch: unknown op %q at index %d; allowed: %s",
				o.Op, i, allowedBatchOps(),
			))
		}
	}

	if req.DryRun {
		return s.executeSequential(ctx, req.Operations, true), nil
	}

	if s.tx != nil {
		return s.executeAtomic(ctx, req.Operations)
	}

	if !req.NonAtomic {
		return nil, fmt.Errorf(
			"%w: batch.execute requires a transactional backend; "+
				"pass non_atomic:true for sequential best-effort execution",
			core.ErrNotImplemented,
		)
	}

	return s.executeSequential(ctx, req.Operations, false), nil
}

// executeAtomic runs every operation inside one transaction. The first
// failure aborts and rolls back; later operations are reported skipped.
func (s *BatchService) executeAtomic(
	ctx context.Context,
	ops []BatchOperation,
) (*ExecuteBatchResponse, error) {
	results := make([]BatchResult, len(ops))
	failedAt := -1

	runErr := s.tx.Run(ctx, func(txStore store.Store) error {
		records := NewRecordService(txStore)
		schemas := NewSchemaService(txStore)

		for i, o := range ops {
			_, opErr := batchOps[o.Op](ctx, records, schemas, o, false)
			if opErr != nil {
				failedAt = i
				results[i] = BatchResult{
					Index:  i,
					URI:    o.URI,
					Status: "error",
					Error:  rpc.MapError(opErr),
				}
				return errBatchAborted
			}

			results[i] = BatchResult{Index: i, URI: o.URI, Status: "ok"}
		}

		return nil
	})

	if runErr != nil && failedAt < 0 {
		return nil, runErr
	}

	resp := &ExecuteBatchResponse{
		Results: results,
		Total:   len(ops),
	}

	if failedAt >= 0 {
		resp.RolledBack = true
		resp.Failed = 1
		for i := failedAt + 1; i < len(ops); i++ {
			results[i] = BatchResult{Index: i, URI: ops[i].URI, Status: "skipped"}
		}
		return resp, nil
	}

	resp.Succeeded = len(ops)

	for _, o := range ops {
		s.publishOp(o)
	}

	return resp, nil
}

// executeSequential runs operations one by one, continuing past
// failures. Used for dry-run validation and non_atomic execution.
func (s *BatchService) executeSequential(
	ctx context.Context,
	ops []BatchOperation,
	dryRun bool,
) *ExecuteBatchResponse {
	records := NewRecordService(s.store)
	schemas := NewSchemaService(s.store)

	resp := &ExecuteBatchResponse{
		Results: make([]BatchResult, len(ops)),
		Total:   len(ops),
	}

	for i, o := range ops {
		dry, opErr := batchOps[o.Op](ctx, records, schemas, o, dryRun)
		if opErr != nil {
			resp.Failed++
			resp.Results[i] = BatchResult{
				Index:  i,
				URI:    o.URI,
				Status: "error",
				Error:  rpc.MapError(opErr),
			}
			continue
		}

		resp.Succeeded++
		resp.Results[i] = BatchResult{
			Index:  i,
			URI:    o.URI,
			Status: "ok",
			DryRun: dry,
		}

		if !dryRun {
			s.publishOp(o)
		}
	}

	return resp
}
