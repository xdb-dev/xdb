package sql_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestSQLiteTypeName(t *testing.T) {
	tests := []struct {
		tid  core.TID
		want string
	}{
		{core.TIDInteger, "INTEGER"},
		{core.TIDFloat, "REAL"},
		{core.TIDBoolean, "INTEGER"},
		{core.TIDUnsigned, "INTEGER"},
		{core.TIDTime, "INTEGER"},
		{core.TIDString, "TEXT"},
		{core.TIDBytes, "BLOB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, xsql.SQLiteTypeName(string(tt.tid)), "SQLiteTypeName(%v)", tt.tid)
	}
}

// testDB opens an in-memory SQLite DB and returns both the raw DB and Queries.
func testDB(t *testing.T) (*sql.DB, *xsql.Queries) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	q := xsql.NewQueries(db)
	require.NoError(t, q.Bootstrap(context.Background()))

	return db, q
}

func TestBootstrap(t *testing.T) {
	t.Run("creates _schemas table", func(t *testing.T) {
		db, _ := testDB(t)

		_, err := db.ExecContext(
			context.Background(),
			"INSERT INTO _schemas (_ns, _name, _data) VALUES (?, ?, ?)",
			"ns", "name", []byte("{}"),
		)
		require.NoError(t, err)
	})

	t.Run("is idempotent", func(t *testing.T) {
		_, q := testDB(t)
		require.NoError(t, q.Bootstrap(context.Background()))
	})
}
