package xdbsqlite

import (
	"fmt"
	"strings"
)

// Config holds SQLite-specific configuration options.
type Config struct {
	// Path is the path to the SQLite database file.
	// Supports special values like ":memory:" for in-memory databases.
	// Default: "xdb.db"
	Path string `env:"PATH"`

	// SchemaDir is the directory to store schema files.
	// Default: ".schema"
	SchemaDir string `env:"SCHEMA_DIR"`

	// Mode specifies the database access mode.
	// Options: "ro" (read-only), "rw" (read-write), "rwc" (read-write-create), "memory"
	// Default: "rwc"
	Mode string `env:"MODE"`

	// Cache specifies the cache mode.
	// Options: "shared", "private"
	// Default: "shared"
	Cache string `env:"CACHE"`

	// MaxOpenConns is the maximum number of open connections to the database.
	// Default: 25
	MaxOpenConns int `env:"MAX_OPEN_CONNS"`

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 10
	MaxIdleConns int `env:"MAX_IDLE_CONNS"`

	// ConnMaxLifetime is the maximum amount of time a connection may be reused (in seconds).
	// Default: 3600 (1 hour)
	ConnMaxLifetime int `env:"CONN_MAX_LIFETIME"`

	// BusyTimeout is the timeout for waiting on locks (in milliseconds).
	// Default: 5000 (5 seconds)
	BusyTimeout int `env:"BUSY_TIMEOUT"`

	// Pragmas is a map of SQLite pragmas to set.
	// Default: {"journal_mode": "WAL", "synchronous": "NORMAL"}
	Pragmas map[string]string `env:"PRAGMAS"`
}

// DefaultConfig creates a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Path:            "xdb.db",
		Mode:            "rwc",
		Cache:           "shared",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 3600,
		BusyTimeout:     5000,
		Pragmas: map[string]string{
			"journal_mode": "WAL",
			"synchronous":  "NORMAL",
		},
	}
}

func (cfg *Config) DSN() string {
	var params []string

	if cfg.Mode != "" {
		params = append(params, "mode="+cfg.Mode)
	}

	if cfg.Cache != "" {
		params = append(params, "cache="+cfg.Cache)
	}

	if cfg.BusyTimeout > 0 {
		params = append(params, fmt.Sprintf("_busy_timeout=%d", cfg.BusyTimeout))
	}

	dsn := cfg.Path
	if len(params) > 0 {
		dsn = fmt.Sprintf("file:%s?%s", cfg.Path, strings.Join(params, "&"))
	}

	return dsn
}
