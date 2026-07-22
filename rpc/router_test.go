package rpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		wantCode       int
		wantDataReason string
	}{
		{
			name:     "ErrNotFound maps to CodeNotFound",
			err:      core.ErrNotFound,
			wantCode: CodeNotFound,
		},
		{
			name:     "wrapped ErrNotFound maps to CodeNotFound",
			err:      fmt.Errorf("api: records.get xdb://ns/s/id: %w", core.ErrNotFound),
			wantCode: CodeNotFound,
		},
		{
			name:     "ErrAlreadyExists maps to CodeAlreadyExists",
			err:      core.ErrAlreadyExists,
			wantCode: CodeAlreadyExists,
		},
		{
			name:     "ErrSchemaViolation maps to CodeSchemaViolation",
			err:      core.ErrSchemaViolation,
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "wrapped ErrSchemaViolation maps to CodeSchemaViolation",
			err:      fmt.Errorf("%w: field age: expected INTEGER, got STRING", core.ErrSchemaViolation),
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "unknown error maps to CodeInternalError",
			err:      errors.New("something unexpected"),
			wantCode: CodeInternalError,
		},
		{
			name:     "RPC error passes through",
			err:      NotFound("already an rpc error"),
			wantCode: CodeNotFound,
		},
		{
			name:     "ErrConflict maps to CodeConflict",
			err:      core.ErrConflict,
			wantCode: CodeConflict,
		},
		{
			name:     "wrapped ErrConflict maps to CodeConflict",
			err:      fmt.Errorf("schema update: %w", core.ErrConflict),
			wantCode: CodeConflict,
		},
		{
			name:     "ErrNotImplemented maps to CodeNotImplemented",
			err:      core.ErrNotImplemented,
			wantCode: CodeNotImplemented,
		},
		{
			name:     "wrapped ErrNotImplemented maps to CodeNotImplemented",
			err:      fmt.Errorf("api: batch.execute: %w", core.ErrNotImplemented),
			wantCode: CodeNotImplemented,
		},
		{
			name:           "ErrInvalidURI maps to CodeInvalidParams with reason",
			err:            core.ErrInvalidURI,
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_uri",
		},
		{
			name:           "wrapped ErrInvalidURI maps to CodeInvalidParams with reason",
			err:            fmt.Errorf("parse uri: %w", core.ErrInvalidURI),
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_uri",
		},
		{
			name:           "ErrInvalidFilter maps to CodeInvalidParams with reason",
			err:            core.ErrInvalidFilter,
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_filter",
		},
		{
			name:           "wrapped ErrInvalidFilter maps to CodeInvalidParams with reason",
			err:            fmt.Errorf("compile filter: %w", core.ErrInvalidFilter),
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_filter",
		},
		{
			name:     "ErrUnknownType maps to CodeSchemaViolation",
			err:      core.ErrUnknownType,
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "wrapped ErrUnknownType maps to CodeSchemaViolation",
			err:      fmt.Errorf("parse type: %w", core.ErrUnknownType),
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "schema.ErrInvalidMode maps to CodeSchemaViolation",
			err:      schema.ErrInvalidMode,
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "wrapped schema.ErrInvalidMode maps to CodeSchemaViolation",
			err:      fmt.Errorf("validate def: %w", schema.ErrInvalidMode),
			wantCode: CodeSchemaViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := MapError(tt.err)
			assert.Equal(t, tt.wantCode, rpcErr.Code)

			if tt.wantDataReason != "" {
				data, ok := rpcErr.Data.(map[string]any)
				require.True(t, ok, "expected Data to be map[string]any, got %T", rpcErr.Data)
				assert.Equal(t, tt.wantDataReason, data["reason"])
			}
		})
	}
}
