package rpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestMapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantCode int
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := MapError(tt.err)
			assert.Equal(t, tt.wantCode, rpcErr.Code)
		})
	}
}
