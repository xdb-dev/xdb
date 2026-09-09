package xdbmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/storetest"
)

// newStore creates a store with enforcement and versioning via [store.New].
func newStore() store.Store {
	return store.New(xdbmemory.NewDriver())
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newStore()

	var _ store.Store = s

	_, ok := s.(store.HealthChecker)
	require.True(t, ok, "store over memory driver must report health")

	_, ok = s.(store.TX)
	require.True(t, ok, "store over memory driver must support TX")
}

func TestHealth(t *testing.T) {
	s := newStore()
	h, ok := s.(store.HealthChecker)
	require.True(t, ok)
	require.NoError(t, h.Health(context.Background()))
}

func TestRecords(t *testing.T) {
	storetest.NewRecordStoreSuite(func() store.RecordStore {
		return newStore()
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	storetest.NewSchemaStoreSuite(func() store.SchemaStore {
		return newStore()
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	storetest.NewNamespaceStoreSuite(func() storetest.NamespaceStore {
		return newStore()
	}).Run(t)
}

func TestBatch(t *testing.T) {
	storetest.NewBatchSuite(func() storetest.BatchStore {
		return newStore().(storetest.BatchStore)
	}).Run(t)
}

func TestTuples(t *testing.T) {
	storetest.NewTupleStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestModes(t *testing.T) {
	storetest.NewModeStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestCascade(t *testing.T) {
	storetest.NewCascadeStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestTypes(t *testing.T) {
	storetest.NewTypesStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestVersioning(t *testing.T) {
	storetest.NewVersionSuite(func() store.Store {
		return newStore()
	}).Run(t)
}
