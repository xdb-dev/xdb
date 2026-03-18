package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/redis/go-redis/v9"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
)

// OpenStore creates a [store.Store] based on the config's store backend.
// The caller must call Close on the returned store when done.
func OpenStore(cfg *Config) (store.Store, error) {
	switch cfg.Store.backendName() {
	case "memory":
		return xdbmemory.New(), nil
	case "sqlite":
		return openSQLiteStore(cfg)
	case "redis":
		return openRedisStore(cfg)
	case "fs":
		return openFSStore(cfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackend, cfg.Store.Backend)
	}
}

func openSQLiteStore(cfg *Config) (store.Store, error) {
	dbPath := cfg.Store.SQLite.Path
	if dbPath == "" {
		dbPath = filepath.Join(cfg.DataDir(), "xdb.db")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	dsn := cfg.Store.SQLite.DSN(dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s, err := xdbsqlite.New(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init sqlite store: %w", err)
	}

	return s, nil
}

func openRedisStore(cfg *Config) (store.Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Store.Redis.Addr,
		Password: cfg.Store.Redis.Password,
		DB:       cfg.Store.Redis.DB,
	})

	return xdbredis.New(client), nil
}

func openFSStore(cfg *Config) (store.Store, error) {
	dir := cfg.Store.FS.Dir
	if dir == "" {
		dir = cfg.DataDir()
	}

	s, err := xdbfs.New(dir, xdbfs.Options{})
	if err != nil {
		return nil, fmt.Errorf("init fs store: %w", err)
	}

	return s, nil
}
