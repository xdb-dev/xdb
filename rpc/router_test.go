package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	xerrors "github.com/gojekfarm/xtools/errors"
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
		wantData       map[string]string
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
			err:      fmt.Errorf("[xdb/api] records.get xdb://ns/s/id: %w", core.ErrNotFound),
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
			err:      fmt.Errorf("[xdb/api] batch.execute: %w", core.ErrNotImplemented),
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
		{
			name: "tags reach Data",
			err: xerrors.Wrap(core.ErrSchemaViolation,
				"field", "age",
				"expected", "INTEGER",
				"got", "STRING",
			),
			wantCode: CodeSchemaViolation,
			wantData: map[string]string{
				"field":    "age",
				"expected": "INTEGER",
				"got":      "STRING",
			},
		},
		{
			name: "tags survive an outer fmt.Errorf",
			err: fmt.Errorf("[xdb/api] records.put: %w", xerrors.Wrap(
				core.ErrSchemaViolation, "field", "age",
			)),
			wantCode: CodeSchemaViolation,
			wantData: map[string]string{"field": "age"},
		},
		{
			name:           "malformed JSON maps to CodeInvalidParams",
			err:            fmt.Errorf("[xdb/api] schemas.create: %w", jsonSyntaxError()),
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_payload",
		},
		{
			name:           "a JSON type mismatch maps to CodeInvalidParams",
			err:            fmt.Errorf("[xdb/api] schemas.create: %w", jsonTypeError()),
			wantCode:       CodeInvalidParams,
			wantDataReason: "invalid_payload",
		},
		{
			name: "a tagged reason wins over the sentinel default",
			err: xerrors.Wrap(core.ErrInvalidURI,
				"reason", "attr_not_allowed",
				"parts", "a/b/c",
			),
			wantCode: CodeInvalidParams,
			wantData: map[string]string{
				"reason": "attr_not_allowed",
				"parts":  "a/b/c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := MapError(tt.err)
			assert.Equal(t, tt.wantCode, rpcErr.Code)

			if tt.wantDataReason != "" {
				data, ok := rpcErr.Data.(map[string]string)
				require.True(t, ok, "expected Data to be map[string]string, got %T", rpcErr.Data)
				assert.Equal(t, tt.wantDataReason, data["reason"])
			}

			if tt.wantData != nil {
				assert.Equal(t, tt.wantData, rpcErr.Data)
			} else if tt.wantDataReason == "" {
				assert.Nil(t, rpcErr.Data)
			}
		})
	}
}

func TestInvokeRejectsStreamingMethod(t *testing.T) {
	r := NewRouter()

	type pong struct {
		Msg string `json:"msg"`
	}

	RegisterHandler(r, "unary.ping", func(ctx context.Context, req *struct{}) (*pong, error) {
		return &pong{Msg: "pong"}, nil
	})
	RegisterStream(r, "events.watch", func(ctx context.Context, req *struct{}, send func(string, json.RawMessage)) error {
		send("event", json.RawMessage(`{}`))
		return nil
	})

	t.Run("unary", func(t *testing.T) {
		res, err := r.Invoke(context.Background(), "unary.ping", nil)
		require.NoError(t, err)
		assert.JSONEq(t, `{"msg":"pong"}`, string(res))
	})

	t.Run("streaming", func(t *testing.T) {
		_, err := r.Invoke(context.Background(), "events.watch", nil)
		require.Error(t, err)

		var rpcErr *Error
		require.ErrorAs(t, err, &rpcErr)
		assert.Equal(t, CodeInvalidRequest, rpcErr.Code)
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := r.Invoke(context.Background(), "nope.nope", nil)
		require.Error(t, err)
	})
}

// jsonSyntaxError returns the error the standard decoder gives for
// malformed JSON.
func jsonSyntaxError() error {
	var v map[string]any
	return json.Unmarshal([]byte(`{"fields":`), &v)
}

// jsonTypeError returns the error the standard decoder gives when a value
// has the wrong JSON type for its target field.
func jsonTypeError() error {
	var v struct {
		Fields map[string]string `json:"fields"`
	}
	return json.Unmarshal([]byte(`{"fields":{"items":1}}`), &v)
}
