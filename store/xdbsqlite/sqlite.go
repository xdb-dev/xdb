package xdbsqlite

import (
	"context"
	"database/sql"

	"github.com/ncruces/go-sqlite3/driver"

	"github.com/xdb-dev/xdb/store"
)

var _ store.HealthChecker = (*Store)(nil)

type Store struct {
	cfg Config
	db  *sql.DB
}

func New(cfg Config, ns string) (*Store, error) {
	s := &Store{cfg: cfg}

	db, err := s.newDB()
	if err != nil {
		return nil, err
	}

	s.db = db

	return s, nil
}

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

	return db, nil
}
