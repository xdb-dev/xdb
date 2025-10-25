package xdbsqlite

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	// Register SQLite3 driver
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	cfg Config
	db  *sql.DB
}

func NewStore(cfg Config) (*Store, error) {
	s := &Store{cfg: cfg}

	db, err := s.newDB()
	if err != nil {
		return nil, err
	}

	s.db = db

	return s, nil
}

func (s *Store) newDB() (*sql.DB, error) {
	dsn := s.cfg.DSN()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply connection pool settings
	db.SetMaxOpenConns(s.cfg.MaxOpenConns)
	db.SetMaxIdleConns(s.cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(s.cfg.ConnMaxLifetime) * time.Second)

	// Apply SQLite-specific pragmas
	if err := s.applySQLitePragmas(db); err != nil {
		return nil, fmt.Errorf("failed to apply SQLite pragmas: %w", err)
	}

	slog.Info("[SQLite] Database configured",
		"path", s.cfg.Path,
		"journal_mode", s.cfg.JournalMode,
		"synchronous", s.cfg.Synchronous)

	return db, nil
}

func (s *Store) applySQLitePragmas(db *sql.DB) error {
	pragmas := []struct {
		name  string
		value string
	}{
		{"journal_mode", s.cfg.JournalMode},
		{"synchronous", s.cfg.Synchronous},
	}

	for _, pragma := range pragmas {
		if pragma.value == "" {
			continue
		}

		query := fmt.Sprintf("PRAGMA %s = %s", pragma.name, pragma.value)
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to set PRAGMA %s: %w", pragma.name, err)
		}
	}

	return nil
}
