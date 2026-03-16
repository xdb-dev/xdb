// Package sql contains typed SQL queries for the xdbsqlite store.
// Value encoding/decoding is handled via [driver.Valuer] and [sql.Scanner]
// interfaces backed by a package-level codec with SQLite type mappings.
package sql

import (
	"context"
	"database/sql"

	"github.com/xdb-dev/xdb/core"
)

// DBTX abstracts [*sql.DB] and [*sql.Tx] so that the same query
// methods work inside and outside transactions.
type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var defaultCodec = newCodec()

// Queries provides typed SQL operations backed by a [DBTX].
type Queries struct {
	db DBTX
}

// NewQueries creates a new [Queries] backed by the given [DBTX].
func NewQueries(db DBTX) *Queries {
	return &Queries{db: db}
}

// SQLiteTypeName returns the SQLite type name for the given [core.TID].
// Returns "TEXT" if no mapping is registered.
func SQLiteTypeName(tid string) string {
	name, err := defaultCodec.TypeName(core.NewType(core.TID(tid)))
	if err != nil {
		return "TEXT"
	}
	return name
}

// Bootstrap creates the _schemas metadata table if it does not exist.
func (q *Queries) Bootstrap(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS _schemas (
			_ns   TEXT NOT NULL,
			_name TEXT NOT NULL,
			_data BLOB NOT NULL,
			PRIMARY KEY (_ns, _name)
		)
	`)
	return err
}
