package xdbsqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func newTestStore(t *testing.T) *xdbsqlite.Store {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	s, err := xdbsqlite.New(db)
	require.NoError(t, err)

	return s
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	var _ store.Store = s
	var _ store.HealthChecker = s
	var _ store.BatchExecutor = s
}

func TestHealth(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Health(context.Background()))
}

func TestRecords(t *testing.T) {
	tests.NewRecordStoreSuite(func() store.RecordStore {
		return newTestStore(t)
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	tests.NewSchemaStoreSuite(func() store.SchemaStore {
		return newTestStore(t)
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	tests.NewNamespaceStoreSuite(func() tests.NamespaceStore {
		return newTestStore(t)
	}).Run(t)
}

func TestBatch(t *testing.T) {
	tests.NewBatchSuite(func() tests.BatchStore {
		return newTestStore(t)
	}).Run(t)
}

func TestModes(t *testing.T) {
	tests.NewModeStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}
