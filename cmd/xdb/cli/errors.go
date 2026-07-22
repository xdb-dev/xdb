package cli

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/rpc"
)

var (
	// ErrConfigDirEmpty is returned when the config directory is empty.
	ErrConfigDirEmpty = errors.New("config dir cannot be empty")

	// ErrConfigDirNotAbsolute is returned when the config directory is not absolute.
	ErrConfigDirNotAbsolute = errors.New("config dir must be absolute or start with ~")

	// ErrInvalidSocket is returned when the socket path contains directory separators.
	ErrInvalidSocket = errors.New("daemon.socket must be a filename, not a path")

	// ErrInvalidLogLevel is returned when the log level is not recognized.
	ErrInvalidLogLevel = errors.New("invalid log_level")

	// ErrUnsupportedBackend is returned when the store backend is not recognized.
	ErrUnsupportedBackend = errors.New("unsupported store backend")

	// ErrRedisAddrRequired is returned when the redis backend is configured without an address.
	ErrRedisAddrRequired = errors.New("store.redis.addr is required when backend is redis")
)

// Error codes rendered in [output.ErrorEnvelope].
const (
	CodeNotFound          = "NOT_FOUND"
	CodeAlreadyExists     = "ALREADY_EXISTS"
	CodeSchemaViolation   = "SCHEMA_VIOLATION"
	CodeInvalidArgument   = "INVALID_ARGUMENT"
	CodeConnectionRefused = "CONNECTION_REFUSED"
	CodeInternal          = "INTERNAL"
)

// Exit codes — stable values the CLI returns to the shell.
const (
	ExitOK          = 0
	ExitAppError    = 1
	ExitConnection  = 2
	ExitInvalidArgs = 3
	ExitInternal    = 4
)

// wrapRPCError converts an RPC or transport error into an [*output.ErrorEnvelope].
// Returns nil if err is nil, and passes through envelopes that are already wrapped.
func wrapRPCError(resource, action, uri string, err error) error {
	if err == nil {
		return nil
	}

	var env *output.ErrorEnvelope
	if errors.As(err, &env) {
		return env
	}

	var rpcErr *rpc.Error
	if errors.As(err, &rpcErr) {
		code := codeFromRPC(rpcErr.Code)
		return &output.ErrorEnvelope{
			Code:     code,
			Message:  rpcErr.Message,
			Resource: resource,
			Action:   action,
			URI:      uri,
			Hint:     hintFor(code, resource, uri),
		}
	}

	if isConnectionError(err) {
		return &output.ErrorEnvelope{
			Code:     CodeConnectionRefused,
			Message:  err.Error(),
			Resource: resource,
			Action:   action,
			URI:      uri,
			Hint:     "is the daemon running? try: xdb daemon start",
		}
	}

	return err
}

// invalidArgError wraps a user input error into an envelope.
func invalidArgError(resource, action string, err error) error {
	if err == nil {
		return nil
	}

	return &output.ErrorEnvelope{
		Code:     CodeInvalidArgument,
		Message:  err.Error(),
		Resource: resource,
		Action:   action,
	}
}

func codeFromRPC(code int) string {
	switch code {
	case rpc.CodeNotFound:
		return CodeNotFound
	case rpc.CodeAlreadyExists:
		return CodeAlreadyExists
	case rpc.CodeSchemaViolation:
		return CodeSchemaViolation
	case rpc.CodeInvalidParams, rpc.CodeInvalidRequest, rpc.CodeParseError:
		return CodeInvalidArgument
	default:
		return CodeInternal
	}
}

func hintFor(code, resource, uri string) string {
	switch code {
	case CodeNotFound:
		if resource == "records" && uri != "" {
			return "try: xdb records list " + parentOf(uri)
		}

		return "run xdb describe --actions to see available operations"
	case CodeAlreadyExists:
		return "use update or upsert instead of create"
	case CodeSchemaViolation:
		return "run xdb describe --uri <schema-uri> to inspect the schema"
	case CodeInvalidArgument:
		return "run xdb describe <resource>.<action> to see expected parameters"
	default:
		return ""
	}
}

// parentOf returns the URI of the parent resource, or the input if no parent exists.
func parentOf(uri string) string {
	hash := strings.LastIndex(uri, "#")
	if hash >= 0 {
		uri = uri[:hash]
	}

	slash := strings.LastIndex(uri, "/")
	if slash < len("xdb://") {
		return uri
	}

	return uri[:slash]
}

// isConnectionError returns true when err looks like a daemon-connection failure:
// the socket refused the connection, the socket file does not exist, or any
// other Unix-level refusal wrapped by the HTTP/RPC client.
func isConnectionError(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	if errors.Is(err, syscall.ENOENT) || errors.Is(err, os.ErrNotExist) {
		return true
	}

	return false
}

// ExitCodeFor returns the shell exit code for an error.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}

	var env *output.ErrorEnvelope
	if !errors.As(err, &env) {
		var exitErr cli.ExitCoder
		if errors.As(err, &exitErr) {
			return ExitInvalidArgs
		}

		return ExitAppError
	}

	switch env.Code {
	case CodeConnectionRefused:
		return ExitConnection
	case CodeInvalidArgument:
		return ExitInvalidArgs
	case CodeInternal:
		return ExitInternal
	default:
		return ExitAppError
	}
}

// normalizeError converts a bare [cli.ExitCoder] — e.g. urfave's own
// "No help topic for X" error, raised when --help targets an unknown
// subcommand name — into an INVALID_ARGUMENT [output.ErrorEnvelope].
// Envelopes pass through unchanged; any other error is returned as-is.
func normalizeError(err error) error {
	if err == nil {
		return nil
	}

	var env *output.ErrorEnvelope
	if errors.As(err, &env) {
		return env
	}

	var exitErr cli.ExitCoder
	if errors.As(err, &exitErr) {
		return &output.ErrorEnvelope{
			Code:    CodeInvalidArgument,
			Message: err.Error(),
			Hint:    "run 'xdb --help' to list commands",
		}
	}

	return err
}

// WriteError renders an error to w. format is the output-format name
// (e.g. "json", "yaml", "table", or "" for auto-detect).
// Raw (non-envelope) errors fall back to a short "error: ..." line.
func WriteError(w io.Writer, format string, err error) {
	if err == nil {
		return
	}

	err = normalizeError(err)

	resolved := output.Detect(format, isTerminalWriter(w))
	formatter := output.New(resolved)
	_ = formatter.FormatError(w, err)
}

// FinalizeError renders err as a fallback when [*cli.Command.Run] returns it
// without having already rendered it via the root command's ExitErrHandler.
// Every error produced by CLI actions or [installUsageErrorHandler] is an
// *[output.ErrorEnvelope] by the time Run returns, and envelopes were already
// rendered during Run(); those pass through here untouched. A handful of
// urfave-internal errors (e.g. "No help topic for X", raised when --help
// targets an unknown subcommand name) bypass Before/Action/ExitErrHandler
// entirely and reach the caller unrendered — this renders those, once.
//
// Call this immediately after Run(), before computing the exit code with
// [ExitCodeFor].
func FinalizeError(cmd *cli.Command, err error) {
	if err == nil {
		return
	}

	var env *output.ErrorEnvelope
	if errors.As(err, &env) {
		return
	}

	WriteError(cmd.Root().ErrWriter, cmd.Root().String("output"), err)
}

// isTerminalWriter returns true when w is a terminal file.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	stat, err := f.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) != 0
}
