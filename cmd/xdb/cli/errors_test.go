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
	"github.com/urfave/cli/v3"
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
			name:     "conflict",
			in:       rpc.Conflict("record exists with different data"),
			wantCode: CodeConflict,
		},
		{
			name:     "not implemented",
			in:       rpc.NotImplemented("batch requires a transactional store"),
			wantCode: CodeNotImplemented,
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
		{"conflict", &output.ErrorEnvelope{Code: CodeConflict}, ExitAppError},
		{"not implemented", &output.ErrorEnvelope{Code: CodeNotImplemented}, ExitAppError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitCodeFor(tc.err))
		})
	}
}

func TestExitCodeFor_ExitCoder(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"bare cli.ExitCoder", cli.Exit("no help topic for 'upsert'", 3), ExitInvalidArgs},
		{"wrapped cli.ExitCoder", fmt.Errorf("wrap: %w", cli.Exit("boom", 1)), ExitInvalidArgs},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExitCodeFor(tc.err))
		})
	}
}

func TestNormalizeError(t *testing.T) {
	t.Run("nil passes through", func(t *testing.T) {
		assert.Nil(t, normalizeError(nil))
	})

	t.Run("envelope passes through unchanged", func(t *testing.T) {
		env := &output.ErrorEnvelope{Code: CodeNotFound, Message: "gone"}
		assert.Same(t, env, normalizeError(env))
	})

	t.Run("cli.ExitCoder becomes an INVALID_ARGUMENT envelope", func(t *testing.T) {
		got := normalizeError(cli.Exit("no help topic for 'upsert'", 3))

		var env *output.ErrorEnvelope
		require.ErrorAs(t, got, &env)
		assert.Equal(t, CodeInvalidArgument, env.Code)
		assert.Equal(t, "no help topic for 'upsert'", env.Message)
		assert.Equal(t, "run 'xdb --help' to list commands", env.Hint)
	})

	t.Run("other errors pass through unchanged", func(t *testing.T) {
		err := errors.New("something else")
		assert.Same(t, err, normalizeError(err))
	})
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

func TestHintFor(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"not found", CodeNotFound},
		{"already exists", CodeAlreadyExists},
		{"schema violation", CodeSchemaViolation},
		{"conflict", CodeConflict},
		{"invalid argument", CodeInvalidArgument},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, hintFor(tc.code, "records", "get", ""))
		})
	}
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

func TestHintFor_SchemaViolationOnSchemasCreate(t *testing.T) {
	hint := hintFor(CodeSchemaViolation, "schemas", "create", "")
	assert.Contains(t, hint, "--schema-format", "must not point at describing a schema that failed to create")

	recordHint := hintFor(CodeSchemaViolation, "records", "create", "xdb://ns/s/r")
	assert.Contains(t, recordHint, "describe --uri")
}

func TestInvalidArgError_SetsHint(t *testing.T) {
	err := invalidArgError("records", "create", assert.AnError)

	var env *output.ErrorEnvelope
	require.ErrorAs(t, err, &env)
	assert.NotEmpty(t, env.Hint)
}

func TestHintFor_UniqueViolationIsNotConflictAdvice(t *testing.T) {
	unique := hintFor(CodeUniqueViolation, "records", "create", "xdb://ns/members/m2")
	assert.NotContains(t, unique, "upsert",
		"upsert writes the same duplicate value, so it is not a way out")
	assert.Contains(t, unique, "value")

	conflict := hintFor(CodeConflict, "records", "update", "xdb://ns/members/m1")
	assert.Contains(t, conflict, "upsert")
}

func TestCodeFromRPC_UniqueViolation(t *testing.T) {
	assert.Equal(t, CodeUniqueViolation, codeFromRPC(rpc.CodeUniqueViolation))
	assert.Equal(t, CodeConflict, codeFromRPC(rpc.CodeConflict))
}
