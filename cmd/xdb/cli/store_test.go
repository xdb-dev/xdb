package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenStore(t *testing.T) {
	t.Run("memory backend", func(t *testing.T) {
		cfg := &Config{
			Dir:   t.TempDir(),
			Store: StoreConfig{Backend: "memory"},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("sqlite backend with defaults", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{
			Dir:   dir,
			Store: StoreConfig{Backend: "sqlite"},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("sqlite backend with custom path", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "custom.db")
		cfg := &Config{
			Dir: dir,
			Store: StoreConfig{
				Backend: "sqlite",
				SQLite:  SQLiteConfig{Path: dbPath},
			},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("fs backend with defaults", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{
			Dir:   dir,
			Store: StoreConfig{Backend: "fs"},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("fs backend with custom dir", func(t *testing.T) {
		dir := t.TempDir()
		dataDir := filepath.Join(dir, "custom-data")
		cfg := &Config{
			Dir: dir,
			Store: StoreConfig{
				Backend: "fs",
				FS:      FSConfig{Dir: dataDir},
			},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("empty backend defaults to sqlite", func(t *testing.T) {
		dir := t.TempDir()
		cfg := &Config{
			Dir:   dir,
			Store: StoreConfig{},
		}

		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("redis backend creates store", func(t *testing.T) {
		cfg := &Config{
			Dir: t.TempDir(),
			Store: StoreConfig{
				Backend: "redis",
				Redis:   RedisConfig{Addr: "localhost:6379"},
			},
		}

		// OpenStore succeeds (creates the client) even if Redis isn't running.
		// The connection is lazy — it only fails on first operation.
		s, err := OpenStore(cfg)
		require.NoError(t, err)
		assert.NotNil(t, s)
		require.NoError(t, s.Close())
	})

	t.Run("unsupported backend", func(t *testing.T) {
		cfg := &Config{
			Dir:   t.TempDir(),
			Store: StoreConfig{Backend: "postgres"},
		}

		_, err := OpenStore(cfg)
		require.ErrorIs(t, err, ErrUnsupportedBackend)
	})
}
