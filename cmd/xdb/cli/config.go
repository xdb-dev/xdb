package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigDir  = "~/.xdb"
	defaultConfigFile = "config.json"
	defaultSocket     = "xdb.sock"
	defaultLogLevel   = "info"
	defaultBackend    = "sqlite"
)

// DaemonConfig holds daemon-specific configuration.
type DaemonConfig struct {
	Socket string `json:"socket"`
}

// StoreConfig holds store backend configuration.
type StoreConfig struct {
	Backend string       `json:"backend,omitempty"`
	FS      FSConfig     `json:"fs,omitzero"`
	Redis   RedisConfig  `json:"redis,omitzero"`
	SQLite  SQLiteConfig `json:"sqlite,omitzero"`
}

// SQLiteConfig holds SQLite-specific configuration.
type SQLiteConfig struct {
	// Path is the database file path. Default: <datadir>/xdb.db.
	Path string `json:"path,omitempty"`

	// Journal is the journal mode. Default: "wal".
	// Values: "wal", "delete", "truncate", "persist", "memory", "off".
	Journal string `json:"journal,omitempty"`

	// Sync is the synchronous mode. Default: "normal".
	// Values: "off", "normal", "full", "extra".
	Sync string `json:"sync,omitempty"`

	// CacheSize is the page cache size in KiB (negative) or pages (positive).
	// Default: -2000 (2 MiB). See PRAGMA cache_size.
	CacheSize int `json:"cache_size,omitempty"`

	// BusyTimeout is the busy timeout in milliseconds. Default: 5000.
	BusyTimeout int `json:"busy_timeout,omitempty"`
}

const (
	defaultJournal     = "wal"
	defaultSync        = "normal"
	defaultCacheSize   = -2000
	defaultBusyTimeout = 5000
)

// DSN builds a SQLite DSN from the config and the given file path.
// Pragmas are appended as query parameters understood by the ncruces driver.
func (sc SQLiteConfig) DSN(dbPath string) string {
	journal := sc.Journal
	if journal == "" {
		journal = defaultJournal
	}

	sync := sc.Sync
	if sync == "" {
		sync = defaultSync
	}

	busyTimeout := sc.BusyTimeout
	if busyTimeout == 0 {
		busyTimeout = defaultBusyTimeout
	}

	cacheSize := sc.CacheSize
	if cacheSize == 0 {
		cacheSize = defaultCacheSize
	}

	return fmt.Sprintf(
		"%s?_journal=%s&_sync=%s&_busy_timeout=%d&_cache_size=%d",
		dbPath, journal, sync, busyTimeout, cacheSize,
	)
}

// RedisConfig holds Redis-specific configuration.
type RedisConfig struct {
	// Addr is the Redis server address (host:port). Required when backend is "redis".
	Addr string `json:"addr,omitempty"`

	// Password is the Redis auth password. Optional.
	Password string `json:"password,omitempty"`

	// DB is the Redis database number. Default: 0.
	DB int `json:"db,omitempty"`
}

// FSConfig holds filesystem store configuration.
type FSConfig struct {
	// Dir is the root directory. Default: <datadir>.
	Dir string `json:"dir,omitempty"`
}

func (sc StoreConfig) backendName() string {
	if sc.Backend == "" {
		return defaultBackend
	}

	return sc.Backend
}

// Config holds the XDB application configuration.
type Config struct {
	Dir      string       `json:"dir"`
	Daemon   DaemonConfig `json:"daemon"`
	Store    StoreConfig  `json:"store,omitzero"`
	LogLevel string       `json:"log_level"`
}

// NewDefaultConfig creates a [Config] with default values.
func NewDefaultConfig() *Config {
	return &Config{
		Dir: defaultConfigDir,
		Daemon: DaemonConfig{
			Socket: defaultSocket,
		},
		LogLevel: defaultLogLevel,
		Store: StoreConfig{
			Backend: defaultBackend,
		},
	}
}

// DefaultConfigPath returns the default config file path (~/.xdb/config.json).
func DefaultConfigPath() string {
	return filepath.Join(expandTilde(defaultConfigDir), defaultConfigFile)
}

// ExpandedDir returns the config directory with ~ expanded to the home directory.
func (c *Config) ExpandedDir() string {
	return expandTilde(c.Dir)
}

// SocketPath returns the full path to the Unix socket.
func (c *Config) SocketPath() string {
	return filepath.Join(c.ExpandedDir(), c.Daemon.Socket)
}

// LogFile returns the full path to the log file.
func (c *Config) LogFile() string {
	return filepath.Join(c.ExpandedDir(), "xdb.log")
}

// PIDFile returns the full path to the PID file.
func (c *Config) PIDFile() string {
	return filepath.Join(c.ExpandedDir(), "xdb.pid")
}

// DataDir returns the full path to the data directory.
func (c *Config) DataDir() string {
	return filepath.Join(c.ExpandedDir(), "data")
}

// Validate checks that all config values are valid.
func (c *Config) Validate() error {
	if c.Dir == "" {
		return ErrConfigDirEmpty
	}

	expanded := expandTilde(c.Dir)
	if !filepath.IsAbs(expanded) && !strings.HasPrefix(c.Dir, "~") {
		return ErrConfigDirNotAbsolute
	}

	if strings.ContainsAny(c.Daemon.Socket, "/\\") {
		return ErrInvalidSocket
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return ErrInvalidLogLevel
	}

	switch c.Store.backendName() {
	case "memory", "sqlite", "fs":
	case "redis":
		if c.Store.Redis.Addr == "" {
			return ErrRedisAddrRequired
		}
	default:
		return ErrUnsupportedBackend
	}

	return nil
}

// EnsureConfigAt creates a config file at the given path with defaults if it
// doesn't exist. Returns true if a new file was created.
func EnsureConfigAt(configPath string) (bool, error) {
	if _, err := os.Stat(configPath); err == nil {
		return false, nil
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	cfg := NewDefaultConfig()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, append(data, '\n'), 0o600); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}

	return true, nil
}

// LoadConfig reads and validates the config from the given path.
// If configPath is empty, it uses [DefaultConfigPath] and creates defaults
// if the file doesn't exist.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = DefaultConfigPath()

		if _, err := EnsureConfigAt(configPath); err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(configPath) // #nosec G304 - configPath is from trusted CLI flag or hardcoded default
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := NewDefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}
