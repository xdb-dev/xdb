package xdbsqlite_test

import (
	"context"
	"testing"

	sqlite3driver "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/driver/xdbsqlite/internal"
	"github.com/xdb-dev/xdb/tests"
)

func TestSQLiteDriver_Schema(t *testing.T) {
	t.Parallel()

	db, err := sqlite3driver.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	queries := internal.NewQueries(tx)
	err = queries.CreateMetadataTable(context.Background())
	require.NoError(t, err)

	driver := xdbsqlite.NewSchemaDriverTx(tx)
	tests.TestSchemaReaderWriter(t, driver)
}
