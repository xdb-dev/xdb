package app

import (
	"log/slog"
	"os"

	"github.com/gojekfarm/xtools/errors"
	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
)

type storeSet struct {
	schema  store.SchemaStore
	tuple   store.TupleStore
	record  store.RecordStore
	health  store.HealthChecker
	cleanup []func() error
}

func initStoreFromConfig(cfg *Config) (*storeSet, error) {
	switch cfg.Store.backendName() {
	case "memory":
		return initMemoryStore()
	case "sqlite":
		return initSQLiteStore(cfg)
	case "redis":
		return initRedisStore(cfg)
	case "fs":
		return initFSStore(cfg)
	default:
		return nil, errors.Wrap(ErrUnsupportedBackend, "backend", cfg.Store.Backend)
	}
}

func initMemoryStore() (*storeSet, error) {
	slog.Info("Initializing in-memory store")

	st := xdbmemory.New()

	return &storeSet{
		schema: st,
		tuple:  st,
		record: st,
		health: st,
	}, nil
}

func initSQLiteStore(cfg *Config) (*storeSet, error) {
	sqliteCfg := xdbsqlite.Config{
		Dir:      cfg.Store.SQLite.Dir,
		Name:     cfg.Store.SQLite.Name,
		InMemory: cfg.Store.SQLite.InMemory,
	}

	if sqliteCfg.Dir == "" {
		sqliteCfg.Dir = cfg.DataDir()
	}

	if sqliteCfg.Name == "" {
		sqliteCfg.Name = "xdb.db"
	}

	if err := os.MkdirAll(sqliteCfg.Dir, 0o700); err != nil {
		return nil, errors.Wrap(err, "path", sqliteCfg.Dir)
	}

	slog.Info("Initializing SQLite store", "dir", sqliteCfg.Dir, "name", sqliteCfg.Name)

	st, err := xdbsqlite.New(sqliteCfg)
	if err != nil {
		return nil, err
	}

	return &storeSet{
		schema:  st,
		tuple:   st,
		record:  st,
		health:  st,
		cleanup: []func() error{st.Close},
	}, nil
}

func initRedisStore(cfg *Config) (*storeSet, error) {
	slog.Info("Initializing Redis store", "addr", cfg.Store.Redis.Addr)

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Store.Redis.Addr,
		Password: cfg.Store.Redis.Password,
		DB:       cfg.Store.Redis.DB,
	})

	st, err := xdbredis.NewStore(client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	return &storeSet{
		schema:  st,
		tuple:   st,
		record:  st,
		health:  st,
		cleanup: []func() error{st.Close},
	}, nil
}

func initFSStore(cfg *Config) (*storeSet, error) {
	dir := cfg.Store.FS.Dir
	if dir == "" {
		dir = cfg.DataDir()
	}

	slog.Info("Initializing filesystem store", "dir", dir)

	st, err := xdbfs.New(dir)
	if err != nil {
		return nil, err
	}

	return &storeSet{
		schema: st,
		tuple:  st,
		record: st,
		health: xdbmemory.New(),
	}, nil
}
