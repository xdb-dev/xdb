package xdbmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/tests"
)

func TestStoreImplementsInterfaces(t *testing.T) {
	s := xdbmemory.New()

	var _ store.Store = s
	var _ store.HealthChecker = s
	var _ store.BatchExecutor = s
}

func TestHealth(t *testing.T) {
	s := xdbmemory.New()
	require.NoError(t, s.Health(context.Background()))
}

func TestRecords(t *testing.T) {
	tests.NewRecordStoreSuite(func() store.RecordStore {
		return xdbmemory.New()
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	tests.NewSchemaStoreSuite(func() store.SchemaStore {
		return xdbmemory.New()
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	tests.NewNamespaceStoreSuite(func() tests.NamespaceStore {
		return xdbmemory.New()
	}).Run(t)
}

func TestBatch(t *testing.T) {
	tests.NewBatchSuite(func() tests.BatchStore {
		return xdbmemory.New()
	}).Run(t)
}
