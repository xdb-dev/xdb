package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/store"
)

// BatchService provides batch operations.
type BatchService struct {
	store store.Store
}

// NewBatchService creates a [BatchService] backed by the given [store.Store].
func NewBatchService(s store.Store) *BatchService {
	return &BatchService{store: s}
}

// ExecuteBatchRequest is the request for batch.execute.
type ExecuteBatchRequest struct {
	Operations json.RawMessage `json:"operations"`
	DryRun     bool            `json:"dry_run,omitempty"`
}

// ExecuteBatchResponse is the response for batch.execute.
type ExecuteBatchResponse struct {
	Results    []json.RawMessage `json:"results"`
	Total      int               `json:"total"`
	Succeeded  int               `json:"succeeded"`
	Failed     int               `json:"failed"`
	RolledBack bool              `json:"rolled_back,omitempty"`
}

// Execute runs a batch of operations.
func (s *BatchService) Execute(_ context.Context, _ *ExecuteBatchRequest) (*ExecuteBatchResponse, error) {
	return nil, fmt.Errorf("api: batch.execute not implemented")
}
