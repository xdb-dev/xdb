package cli

import "errors"

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
