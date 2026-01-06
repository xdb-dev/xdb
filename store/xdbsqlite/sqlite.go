package xdbsqlite

import (
	"database/sql"

	"github.com/ncruces/go-sqlite3/driver"
)

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

func (s *Store) newDB() (*sql.DB, error) {
	db, err := driver.Open(s.cfg.DSN())
	if err != nil {
		return nil, err
	}

	return db, nil
}
