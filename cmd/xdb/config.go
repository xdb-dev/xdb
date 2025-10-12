package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gojekfarm/xtools/xload"
	"github.com/gojekfarm/xtools/xload/providers/yaml"
)

type Config struct {
	Addr   string       `env:"ADDR"`
	SQLite SQLiteConfig `env:"SQLITE"`
}

// SQLiteConfig holds SQLite-specific configuration options.
type SQLiteConfig struct {
	// Path is the path to the SQLite database file.
	// Supports special values like ":memory:" for in-memory databases.
	// Default: "xdb.db"
	Path string `env:"PATH"`

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

	// JournalMode specifies the journal mode.
	// Options: "DELETE", "TRUNCATE", "PERSIST", "MEMORY", "WAL", "OFF"
	// Default: "WAL"
	JournalMode string `env:"JOURNAL_MODE"`

	// Synchronous specifies the synchronous mode.
	// Options: "OFF", "NORMAL", "FULL", "EXTRA"
	// Default: "NORMAL"
	Synchronous string `env:"SYNCHRONOUS"`
}

func NewDefaultConfig() Config {
	cfg := Config{}

	cfg.Addr = ":8080"
	cfg.SQLite = NewDefaultSQLiteConfig()

	return cfg
}

// NewDefaultSQLiteConfig creates a SQLiteConfig with sensible defaults.
func NewDefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		Path:            "xdb.db",
		Mode:            "rwc",
		Cache:           "shared",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 3600,
		BusyTimeout:     5000,
		JournalMode:     "WAL",
		Synchronous:     "NORMAL",
	}
}

func LoadConfig(ctx context.Context, configPath string) (*Config, error) {
	cfg := NewDefaultConfig()

	loader, err := createLoader(configPath)
	if err != nil {
		return &cfg, err
	}

	err = xload.Load(ctx, &cfg, xload.WithLoader(loader))
	return &cfg, err
}

// createLoader creates the appropriate loader chain based on config file availability
func createLoader(configPath string) (xload.Loader, error) {
	var yamlPath string

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		yamlPath = configPath
	} else {
		yamlPath = findDefaultConfigFile()
	}

	if yamlPath != "" {
		slog.Info("[HTTP] Using YAML config file", "path", yamlPath)

		yamlLoader, err := yaml.NewFileLoader(yamlPath, "_")
		if err != nil {
			return nil, fmt.Errorf("failed to create YAML loader: %w", err)
		}

		return xload.SerialLoader(yamlLoader, xload.OSLoader()), nil
	}

	return xload.SerialLoader(xload.OSLoader()), nil
}

// findDefaultConfigFile checks for default config files and returns the first one found
func findDefaultConfigFile() string {
	candidates := []string{"xdb.yaml", "xdb.yml"}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
