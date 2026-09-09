package xdbredis_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/tests"
)

// newTestStore creates a store with enforcement and versioning via [store.New].
func newTestStore(t testing.TB) store.Store {
	return store.New(newTestDriver(t))
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	var _ store.Store = s

	_, ok := s.(store.HealthChecker)
	require.True(t, ok, "store over redis driver must report health")

	_, ok = s.(store.TX)
	require.False(t, ok,
		"redis driver has no native transactions; the facade must not offer TX")
}

func TestHealth(t *testing.T) {
	s := newTestStore(t)
	h, ok := s.(store.HealthChecker)
	require.True(t, ok)
	require.NoError(t, h.Health(context.Background()))
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

func TestTuples(t *testing.T) {
	tests.NewTupleStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

func TestTypes(t *testing.T) {
	tests.NewTypesStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

func TestVersioning(t *testing.T) {
	tests.NewVersionSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

// Policy suites (ModeStoreSuite, CascadeStoreSuite) are
// driver-independent and run once against the memory reference; see
// the tests package doc. This backend's storage behavior is pinned by
// DriverSuite + the store suites above.
