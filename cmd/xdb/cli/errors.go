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
	CodeConflict          = "CONFLICT"
	CodeUniqueViolation   = "UNIQUE_VIOLATION"
	CodeNotImplemented    = "NOT_IMPLEMENTED"
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
			Hint:     hintFor(code, resource, action, uri),
		}
	}

	if isConnectionError(err) {
		return &output.ErrorEnvelope{
			Code:     CodeConnectionRefused,
			Message:  err.Error(),
			Resource: resource,
			Action:   action,
			URI:      uri,
			Hint:     "the daemon is not running. Start it with: xdb daemon start",
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
		Hint:     hintFor(CodeInvalidArgument, resource, action, ""),
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
	case rpc.CodeConflict:
		return CodeConflict
	case rpc.CodeUniqueViolation:
		return CodeUniqueViolation
	case rpc.CodeNotImplemented:
		return CodeNotImplemented
	case rpc.CodeInvalidParams, rpc.CodeInvalidRequest, rpc.CodeParseError:
		return CodeInvalidArgument
	default:
		return CodeInternal
	}
}

// hintFor returns the next action to try for an error code.
func hintFor(code, resource, action, uri string) string {
	switch code {
	case CodeNotFound:
		if resource == "records" && uri != "" {
			return "try: xdb records list " + parentOf(uri)
		}

		return "run xdb describe --actions to see available actions"
	case CodeAlreadyExists:
		return "use update or upsert instead of create"
	case CodeSchemaViolation:
		// A schema that failed to create cannot be described — point at
		// the format doc instead of a schema that does not exist.
		if resource == "schemas" && action == "create" {
			return "run xdb describe --schema-format for the schema definition format"
		}

		return "run xdb describe --uri <schema-uri> to inspect the schema"
	case CodeConflict:
		return "the resource exists with different data. Use update or upsert for records, or schemas update for schemas"
	case CodeUniqueViolation:
		// update and upsert write the same duplicate value and fail
		// again, so the CONFLICT advice would send the caller in a
		// circle. The only way out is a different value.
		return "another record already holds this value. Change the duplicate value, or delete the record that holds it"
	case CodeNotImplemented:
		return "this operation is not available in this daemon build"
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

// silentExitError carries an exit code with no rendered output, for
// commands whose non-zero exit is a status signal, not a failure.
type silentExitError struct {
	code int
}

func (e *silentExitError) Error() string { return "" }

// ExitCodeFor returns the shell exit code for an error.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}

	var silent *silentExitError
	if errors.As(err, &silent) {
		return silent.code
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
	case CodeConflict, CodeUniqueViolation, CodeNotImplemented:
		// Same as the default. Listed so the app-error codes are
		// visible here.
		return ExitAppError
	default:
		return ExitAppError
	}
}

// normalizeError converts a bare [cli.ExitCoder] — for example urfave's own
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
// (for example "json", "yaml", "table", or "" for auto-detect).
// Raw (non-envelope) errors fall back to a short "error: ..." line.
func WriteError(w io.Writer, format string, err error) {
	if err == nil {
		return
	}

	var silent *silentExitError
	if errors.As(err, &silent) {
		return
	}

	err = normalizeError(err)

	// An unusable --output value is itself one of the errors this renders,
	// so fall back to the default rather than failing to report anything.
	resolved, derr := output.Detect(format, isTerminalWriter(w))
	if derr != nil {
		resolved, _ = output.Detect("", isTerminalWriter(w))
	}

	_ = output.New(resolved).FormatError(w, err)
}

// FinalizeError renders errors that [cli.Command.Run] returns without
// calling ExitErrHandler, such as an unknown help topic. Error envelopes
// pass through because the command already rendered them.
//
// Call FinalizeError after Run and before computing the exit code with
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
