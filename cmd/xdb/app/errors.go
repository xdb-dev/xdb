package app

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrURIRequired is returned when a URI argument is required but not provided.
	ErrURIRequired = errors.New("[xdb/app] URI required")

	// ErrInvalidURI is returned when a URI cannot be parsed.
	ErrInvalidURI = errors.New("[xdb/app] invalid URI")

	// ErrRecordIDRequired is returned when a record ID is required in the URI.
	ErrRecordIDRequired = errors.New("[xdb/app] record ID required in URI")

	// ErrUnsupportedFormat is returned when an unsupported format is specified.
	ErrUnsupportedFormat = errors.New("[xdb/app] unsupported format")

	// ErrTupleNotFound is returned when a tuple is not found.
	ErrTupleNotFound = errors.New("[xdb/app] tuple not found")

	// ErrRecordNotFound is returned when a record is not found.
	ErrRecordNotFound = errors.New("[xdb/app] record not found")

	// ErrNoTuplesReturned is returned when no tuples are returned from the store.
	ErrNoTuplesReturned = errors.New("[xdb/app] no tuples returned")

	// ErrNoRecordsReturned is returned when no records are returned from the store.
	ErrNoRecordsReturned = errors.New("[xdb/app] no records returned")

	// ErrURIMustSpecifySchema is returned when a URI does not specify a schema.
	ErrURIMustSpecifySchema = errors.New("[xdb/app] URI must specify at least a schema")

	// ErrListingRecordsNotImplemented is returned when listing records is not yet supported.
	ErrListingRecordsNotImplemented = errors.New("[xdb/app] listing records not yet implemented in store layer")

	// ErrDeletionCancelled is returned when a deletion is cancelled by the user.
	ErrDeletionCancelled = errors.New("[xdb/app] deletion cancelled by user")

	// ErrDaemonAlreadyRunning is returned when attempting to start a daemon that is already running.
	ErrDaemonAlreadyRunning = errors.New("[xdb/app] daemon already running")

	// ErrDaemonNotRunning is returned when attempting to operate on a daemon that is not running.
	ErrDaemonNotRunning = errors.New("[xdb/app] daemon not running")

	// ErrHealthCheckTimeout is returned when a daemon health check times out.
	ErrHealthCheckTimeout = errors.New("[xdb/app] health check timeout")

	// ErrStopTimeout is returned when stopping a daemon times out.
	ErrStopTimeout = errors.New("[xdb/app] daemon stop timeout")

	// ErrLogFileNotFound is returned when a log file is not found.
	ErrLogFileNotFound = errors.New("[xdb/app] log file not found")

	// ErrConfigDirEmpty is returned when the config directory is empty.
	ErrConfigDirEmpty = errors.New("[xdb/app] config dir cannot be empty")

	// ErrConfigDirNotAbsolute is returned when the config directory is not absolute.
	ErrConfigDirNotAbsolute = errors.New("[xdb/app] config dir must be absolute or start with ~")

	// ErrNoListenerConfigured is returned when no listener is configured.
	ErrNoListenerConfigured = errors.New("[xdb/app] at least one listener must be configured")

	// ErrInvalidDaemonAddr is returned when the daemon address format is invalid.
	ErrInvalidDaemonAddr = errors.New("[xdb/app] invalid daemon.addr format")

	// ErrInvalidSocketPath is returned when the socket path is invalid.
	ErrInvalidSocketPath = errors.New("[xdb/app] daemon.socket must be filename not path")

	// ErrInvalidLogLevel is returned when the log level is invalid.
	ErrInvalidLogLevel = errors.New("[xdb/app] invalid log_level")

	// ErrPathNotAbsolute is returned when a path is expected to be absolute but is not.
	ErrPathNotAbsolute = errors.New("[xdb/app] path is not absolute")
)
