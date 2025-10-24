package server

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

// initDatabase initializes the SQLite database with the provided configuration.
func initDatabase(cfg SQLiteConfig) (*xdbsqlite.KVStore, error) {
	dsn := buildSQLiteDSN(cfg)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// Apply SQLite-specific pragmas
	if err := applySQLitePragmas(db, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply SQLite pragmas: %w", err)
	}

	slog.Info("[SQLite] Database configured",
		"path", cfg.Path,
		"journal_mode", cfg.JournalMode,
		"synchronous", cfg.Synchronous)

	return xdbsqlite.NewKVStore(db), nil
}

// buildSQLiteDSN constructs a SQLite DSN from configuration.
func buildSQLiteDSN(cfg SQLiteConfig) string {
	var params []string

	// Add mode parameter
	if cfg.Mode != "" {
		params = append(params, "mode="+cfg.Mode)
	}

	// Add cache parameter
	if cfg.Cache != "" {
		params = append(params, "cache="+cfg.Cache)
	}

	// Add busy timeout
	if cfg.BusyTimeout > 0 {
		params = append(params, fmt.Sprintf("_busy_timeout=%d", cfg.BusyTimeout))
	}

	dsn := cfg.Path
	if len(params) > 0 {
		dsn = fmt.Sprintf("file:%s?%s", cfg.Path, strings.Join(params, "&"))
	}

	return dsn
}

// applySQLitePragmas applies SQLite-specific PRAGMA statements.
func applySQLitePragmas(db *sql.DB, cfg SQLiteConfig) error {
	pragmas := []struct {
		name  string
		value string
	}{
		{"journal_mode", cfg.JournalMode},
		{"synchronous", cfg.Synchronous},
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
