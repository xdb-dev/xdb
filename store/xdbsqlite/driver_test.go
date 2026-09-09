package xdbsqlite_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/storetest"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// newTestDriver creates a fresh driver over an isolated on-disk
// database (shared in-memory DBs leak state between factory calls).
func newTestDriver(t *testing.T) *xdbsqlite.Driver {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	d, err := xdbsqlite.NewDriver(db)
	require.NoError(t, err)
	return d
}

func TestDriverSuite(t *testing.T) {
	suite := storetest.NewDriverSuite(func() store.Driver {
		return newTestDriver(t)
	})
	suite.Run(t)
}

func TestQuerySuite(t *testing.T) {
	storetest.NewQuerySuite(func() store.Driver {
		return newTestDriver(t)
	}).Run(t)
}
