package cli

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/rpc"
)

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ECONNREFUSED wrapped", fmt.Errorf("dial: %w", syscall.ECONNREFUSED), true},
		{"ENOENT wrapped", fmt.Errorf("open: %w", syscall.ENOENT), true},
		{"os.ErrNotExist wrapped", fmt.Errorf("stat: %w", os.ErrNotExist), true},
		{"net.OpError with ECONNREFUSED", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, true},
		{"unrelated error", errors.New("something else"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isConnectionError(tc.err))
		})
	}
}

func TestWrapRPCError(t *testing.T) {
	tests := []struct {
		name     string
		in       error
		wantCode string
	}{
		{
			name:     "not found",
			in:       rpc.NotFound("record not found"),
			wantCode: CodeNotFound,
		},
		{
			name:     "already exists",
			in:       rpc.AlreadyExists("record exists"),
			wantCode: CodeAlreadyExists,
		},
		{
			name:     "schema violation",
			in:       rpc.SchemaViolation("field missing"),
			wantCode: CodeSchemaViolation,
		},
		{
			name:     "invalid params maps to invalid argument",
			in:       rpc.InvalidParams("bad"),
			wantCode: CodeInvalidArgument,
		},
		{
			name:     "internal error",
			in:       rpc.InternalError("boom"),
			wantCode: CodeInternal,
		},
		{
			name:     "connection refused",
			in:       fmt.Errorf("dial unix /x: %w", syscall.ECONNREFUSED),
			wantCode: CodeConnectionRefused,
		},
		{
			name:     "socket missing",
			in:       fmt.Errorf("dial unix /x: %w", syscall.ENOENT),
			wantCode: CodeConnectionRefused,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := wrapRPCError("records", "get", "xdb://ns/s/id", tc.in)

			var env *output.ErrorEnvelope
			require.ErrorAs(t, wrapped, &env)
			assert.Equal(t, tc.wantCode, env.Code)
			assert.Equal(t, "records", env.Resource)
			assert.Equal(t, "get", env.Action)
			assert.Equal(t, "xdb://ns/s/id", env.URI)
		})
	}
}

func TestWrapRPCError_NilPassesThrough(t *testing.T) {
	assert.Nil(t, wrapRPCError("records", "get", "", nil))
}

func TestWrapRPCError_NonRPCErrorPassesThrough(t *testing.T) {
	err := errors.New("something else")
	wrapped := wrapRPCError("records", "get", "", err)

	var env *output.ErrorEnvelope
	assert.False(t, errors.As(wrapped, &env), "non-RPC, non-connection errors should not be wrapped")
	assert.Equal(t, err, wrapped)
}

func TestWrapRPCError_DoubleWrapIsIdempotent(t *testing.T) {
	first := wrapRPCError("records", "get", "xdb://x/y/z", rpc.NotFound("nope"))
	second := wrapRPCError("other", "other", "other", first)
	assert.Same(t, first, second, "already-wrapped envelopes should not be re-wrapped")
}

func TestInvalidArgError(t *testing.T) {
	e := invalidArgError("records", "update", errors.New("URI required"))

	var env *output.ErrorEnvelope
	require.ErrorAs(t, e, &env)
	assert.Equal(t, CodeInvalidArgument, env.Code)
	assert.Equal(t, "records", env.Resource)
	assert.Equal(t, "update", env.Action)
	assert.Equal(t, "URI required", env.Message)
}

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"plain error", errors.New("x"), ExitAppError},
		{"not found", &output.ErrorEnvelope{Code: CodeNotFound}, ExitAppError},
		{"already exists", &output.ErrorEnvelope{Code: CodeAlreadyExists}, ExitAppError},
		{"connection refused", &output.ErrorEnvelope{Code: CodeConnectionRefused}, ExitConnection},
		{"invalid argument", &output.ErrorEnvelope{Code: CodeInvalidArgument}, ExitInvalidArgs},
		{"internal", &output.ErrorEnvelope{Code: CodeInternal}, ExitInternal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitCodeFor(tc.err))
		})
	}
}

func TestWriteError_UsesEnvelopeShape(t *testing.T) {
	env := &output.ErrorEnvelope{
		Code:     "NOT_FOUND",
		Message:  "record not found",
		Resource: "records",
		Action:   "get",
		URI:      "xdb://ns/s/id",
	}

	var buf bytes.Buffer
	WriteError(&buf, "json", env)

	s := buf.String()
	assert.Contains(t, s, "\"code\": \"NOT_FOUND\"")
	assert.Contains(t, s, "\"resource\": \"records\"")
	assert.Contains(t, s, "\"action\": \"get\"")
}

func TestParentOf(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"xdb://ns/schema/id", "xdb://ns/schema"},
		{"xdb://ns/schema", "xdb://ns"},
		{"xdb://ns/schema/id#attr", "xdb://ns/schema"},
		{"xdb://ns", "xdb://ns"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, parentOf(tc.in))
		})
	}
}
