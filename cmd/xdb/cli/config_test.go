package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.Equal(t, "~/.xdb", cfg.Dir)
	assert.Equal(t, "xdb.sock", cfg.Daemon.Socket)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "sqlite", cfg.Store.Backend)
}

func TestDefaultConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	want := filepath.Join(home, ".xdb", "config.json")
	assert.Equal(t, want, DefaultConfigPath())
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
		want string
	}{
		{"tilde only", "~", home},
		{"tilde slash", "~/foo", filepath.Join(home, "foo")},
		{"absolute", "/tmp/foo", "/tmp/foo"},
		{"relative", "foo/bar", "foo/bar"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expandTilde(tt.path))
		})
	}
}

func TestConfig_PathHelpers(t *testing.T) {
	cfg := &Config{
		Dir:    "/tmp/xdb-test",
		Daemon: DaemonConfig{Socket: "test.sock"},
	}

	assert.Equal(t, "/tmp/xdb-test", cfg.ExpandedDir())
	assert.Equal(t, "/tmp/xdb-test/test.sock", cfg.SocketPath())
	assert.Equal(t, "/tmp/xdb-test/xdb.log", cfg.LogFile())
	assert.Equal(t, "/tmp/xdb-test/xdb.pid", cfg.PIDFile())
	assert.Equal(t, "/tmp/xdb-test/data", cfg.DataDir())
}

func TestConfig_Validate(t *testing.T) {
	valid := func() *Config {
		return &Config{
			Dir:      "/tmp/xdb",
			Daemon:   DaemonConfig{Socket: "xdb.sock"},
			LogLevel: "info",
			Store:    StoreConfig{Backend: "memory"},
		}
	}

	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr error
	}{
		{
			name:   "valid config",
			modify: func(_ *Config) {},
		},
		{
			name:   "tilde dir is valid",
			modify: func(c *Config) { c.Dir = "~/.xdb" },
		},
		{
			name:    "empty dir",
			modify:  func(c *Config) { c.Dir = "" },
			wantErr: ErrConfigDirEmpty,
		},
		{
			name:    "relative dir",
			modify:  func(c *Config) { c.Dir = "relative/path" },
			wantErr: ErrConfigDirNotAbsolute,
		},
		{
			name:    "socket with slash",
			modify:  func(c *Config) { c.Daemon.Socket = "path/to.sock" },
			wantErr: ErrInvalidSocket,
		},
		{
			name:    "socket with backslash",
			modify:  func(c *Config) { c.Daemon.Socket = "path\\to.sock" },
			wantErr: ErrInvalidSocket,
		},
		{
			name:    "invalid log level",
			modify:  func(c *Config) { c.LogLevel = "trace" },
			wantErr: ErrInvalidLogLevel,
		},
		{
			name:    "unsupported backend",
			modify:  func(c *Config) { c.Store.Backend = "postgres" },
			wantErr: ErrUnsupportedBackend,
		},
		{
			name:   "empty backend defaults to sqlite",
			modify: func(c *Config) { c.Store.Backend = "" },
		},
		{
			name:   "memory backend",
			modify: func(c *Config) { c.Store.Backend = "memory" },
		},
		{
			name:   "sqlite backend",
			modify: func(c *Config) { c.Store.Backend = "sqlite" },
		},
		{
			name:   "fs backend",
			modify: func(c *Config) { c.Store.Backend = "fs" },
		},
		{
			name: "redis backend with addr",
			modify: func(c *Config) {
				c.Store.Backend = "redis"
				c.Store.Redis.Addr = "localhost:6379"
			},
		},
		{
			name:    "redis backend without addr",
			modify:  func(c *Config) { c.Store.Backend = "redis" },
			wantErr: ErrRedisAddrRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.modify(cfg)

			err := cfg.Validate()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEnsureConfig(t *testing.T) {
	t.Run("creates config when missing", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		created, err := EnsureConfigAt(configPath)
		require.NoError(t, err)
		assert.True(t, created)

		// File should exist with valid JSON.
		data, readErr := os.ReadFile(configPath)
		require.NoError(t, readErr)

		var cfg Config
		require.NoError(t, json.Unmarshal(data, &cfg))
		assert.Equal(t, "~/.xdb", cfg.Dir)
	})

	t.Run("skips when config exists", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		// Create the file first.
		require.NoError(t, os.WriteFile(configPath, []byte(`{}`), 0o600))

		created, err := EnsureConfigAt(configPath)
		require.NoError(t, err)
		assert.False(t, created)

		// File should be unchanged.
		data, readErr := os.ReadFile(configPath)
		require.NoError(t, readErr)
		assert.Equal(t, "{}", string(data))
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "nested", "deep", "config.json")

		created, err := EnsureConfigAt(configPath)
		require.NoError(t, err)
		assert.True(t, created)
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		cfg := &Config{
			Dir:      "/opt/xdb",
			Daemon:   DaemonConfig{Socket: "my.sock"},
			LogLevel: "debug",
			Store: StoreConfig{
				Backend: "sqlite",
				SQLite:  SQLiteConfig{Path: "/opt/xdb/data/my.db"},
			},
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0o600))

		loaded, loadErr := LoadConfig(configPath)
		require.NoError(t, loadErr)

		assert.Equal(t, "/opt/xdb", loaded.Dir)
		assert.Equal(t, "my.sock", loaded.Daemon.Socket)
		assert.Equal(t, "debug", loaded.LogLevel)
		assert.Equal(t, "sqlite", loaded.Store.Backend)
		assert.Equal(t, "/opt/xdb/data/my.db", loaded.Store.SQLite.Path)
	})

	t.Run("loads fs store config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		require.NoError(t, os.WriteFile(
			configPath,
			[]byte(`{"dir": "/opt/xdb", "store": {"backend": "fs", "fs": {"dir": "/data/xdb"}}}`),
			0o600,
		))

		loaded, err := LoadConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "fs", loaded.Store.Backend)
		assert.Equal(t, "/data/xdb", loaded.Store.FS.Dir)
	})

	t.Run("missing fields get defaults", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		// Only set dir — everything else should default.
		require.NoError(t, os.WriteFile(
			configPath,
			[]byte(`{"dir": "/tmp/xdb"}`),
			0o600,
		))

		loaded, err := LoadConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "xdb.sock", loaded.Daemon.Socket)
		assert.Equal(t, "info", loaded.LogLevel)
		assert.Equal(t, "sqlite", loaded.Store.Backend)
	})

	t.Run("returns error for invalid config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		require.NoError(t, os.WriteFile(
			configPath,
			[]byte(`{"dir": "relative/bad"}`),
			0o600,
		))

		_, err := LoadConfig(configPath)
		require.ErrorIs(t, err, ErrConfigDirNotAbsolute)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path/config.json")
		require.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		require.NoError(t, os.WriteFile(configPath, []byte(`{not json`), 0o600))

		_, err := LoadConfig(configPath)
		require.Error(t, err)
	})

	t.Run("loads sqlite options", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")

		require.NoError(t, os.WriteFile(
			configPath,
			[]byte(`{
				"dir": "/tmp/xdb",
				"store": {
					"backend": "sqlite",
					"sqlite": {
						"journal": "delete",
						"sync": "full",
						"cache_size": -4000,
						"busy_timeout": 10000
					}
				}
			}`),
			0o600,
		))

		loaded, err := LoadConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "delete", loaded.Store.SQLite.Journal)
		assert.Equal(t, "full", loaded.Store.SQLite.Sync)
		assert.Equal(t, -4000, loaded.Store.SQLite.CacheSize)
		assert.Equal(t, 10000, loaded.Store.SQLite.BusyTimeout)
	})
}

func TestSQLiteConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  SQLiteConfig
		path string
		want string
	}{
		{
			name: "defaults",
			cfg:  SQLiteConfig{},
			path: "/data/xdb.db",
			want: "/data/xdb.db?_journal=wal&_sync=normal&_busy_timeout=5000&_cache_size=-2000",
		},
		{
			name: "custom values",
			cfg: SQLiteConfig{
				Journal:     "delete",
				Sync:        "full",
				CacheSize:   -4000,
				BusyTimeout: 10000,
			},
			path: "/data/xdb.db",
			want: "/data/xdb.db?_journal=delete&_sync=full&_busy_timeout=10000&_cache_size=-4000",
		},
		{
			name: "partial override",
			cfg: SQLiteConfig{
				Journal: "truncate",
			},
			path: "/tmp/test.db",
			want: "/tmp/test.db?_journal=truncate&_sync=normal&_busy_timeout=5000&_cache_size=-2000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.DSN(tt.path))
		})
	}
}
