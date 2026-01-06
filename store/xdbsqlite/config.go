package xdbsqlite

import "path/filepath"

// Config holds SQLite-specific configuration options.
type Config struct {
	// Dir is the directory to store the schema files and the SQLite database file.
	// Default: "."
	Dir string `env:"DIR"`

	// Name is the name of the SQLite database file.
	// Default: "xdb.db"
	Name string `env:"NAME"`

	// InMemory specifies if the SQLite database should be in memory.
	// Default: false
	InMemory bool `env:"IN_MEMORY"`
}

// DefaultConfig creates a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Dir:      ".xdb",
		Name:     "xdb.db",
		InMemory: false,
	}
}

func (cfg *Config) DSN() string {
	if cfg.InMemory {
		return ":memory:"
	}

	return filepath.Join(cfg.Dir, cfg.Name)
}
