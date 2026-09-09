package xdbmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/tests"
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
	tests.NewRecordStoreSuite(func() store.RecordStore {
		return newStore()
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	tests.NewSchemaStoreSuite(func() store.SchemaStore {
		return newStore()
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	tests.NewNamespaceStoreSuite(func() tests.NamespaceStore {
		return newStore()
	}).Run(t)
}

func TestBatch(t *testing.T) {
	tests.NewBatchSuite(func() tests.BatchStore {
		return newStore().(tests.BatchStore)
	}).Run(t)
}

func TestTuples(t *testing.T) {
	tests.NewTupleStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestModes(t *testing.T) {
	tests.NewModeStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestCascade(t *testing.T) {
	tests.NewCascadeStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestTypes(t *testing.T) {
	tests.NewTypesStoreSuite(func() store.Store {
		return newStore()
	}).Run(t)
}

func TestVersioning(t *testing.T) {
	tests.NewVersionSuite(func() store.Store {
		return newStore()
	}).Run(t)
}
