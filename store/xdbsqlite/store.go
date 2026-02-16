package xdbsqlite

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ncruces/go-sqlite3/driver"

	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

var (
	_ store.HealthChecker = (*Store)(nil)
	_ store.SchemaStore   = (*Store)(nil)
	_ store.SchemaReader  = (*Store)(nil)
	_ store.RecordStore   = (*Store)(nil)
	_ store.TupleStore    = (*Store)(nil)
)

// Store is a SQLite-backed implementation of the XDB store interfaces.
type Store struct {
	cfg Config
	db  *sql.DB

	mu      sync.RWMutex
	schemas map[string]map[string]*schema.Def
}

// New creates a new SQLite Store with the given configuration.
func New(cfg Config) (*Store, error) {
	ctx := context.Background()

	s := &Store{cfg: cfg}

	db, err := s.newDB()
	if err != nil {
		return nil, err
	}

	s.db = db

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	driver := NewSchemaStoreTx(tx)
	if err := driver.Migrate(ctx); err != nil {
		_ = tx.Rollback()
		_ = db.Close()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := s.loadSchemas(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Health checks if the SQLite database connection is alive.
func (s *Store) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) newDB() (*sql.DB, error) {
	db, err := driver.Open(s.cfg.DSN())
	if err != nil {
		return nil, err
	}

	if s.cfg.InMemory {
		db.SetMaxOpenConns(1)
	}

	return db, nil
}
