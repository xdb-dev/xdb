package xdbfs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/tests"
)

// newTestDriver builds a raw driver over a fresh temp directory.
func newTestDriver(t *testing.T) *xdbfs.Driver {
	t.Helper()
	d, err := xdbfs.NewDriver(t.TempDir(), xdbfs.Options{})
	require.NoError(t, err)
	return d
}

// newTestStore creates a store with enforcement and versioning via [store.New].
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	return store.New(newTestDriver(t))
}

func TestDriverImplementsInterfaces(t *testing.T) {
	d := newTestDriver(t)

	var _ store.Driver = d
	var _ store.HealthChecker = d
	var _ store.Closer = d
}

func TestHealth(t *testing.T) {
	t.Run("valid_root", func(t *testing.T) {
		d := newTestDriver(t)
		require.NoError(t, d.Health(context.Background()))
	})

	t.Run("nonexistent_root", func(t *testing.T) {
		d, err := xdbfs.NewDriver(filepath.Join(t.TempDir(), "gone"), xdbfs.Options{})
		require.NoError(t, err)
		require.NoError(t, os.RemoveAll(d.Root()))

		err = d.Health(context.Background())
		assert.Error(t, err)
	})
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

// --- FS-specific tests ---

func TestFileLayout_SchemaFile(t *testing.T) {
	root := t.TempDir()
	d, err := xdbfs.NewDriver(root, xdbfs.Options{})
	require.NoError(t, err)
	s := store.New(d)

	ctx := context.Background()
	uri := core.MustNewURI("myapp", "users")
	def := &schema.Def{URI: uri, Mode: schema.ModeFlexible}

	err = s.CreateSchema(ctx, uri, def)
	require.NoError(t, err)

	// Verify _schema.json exists at expected path.
	schemaPath := filepath.Join(root, "myapp", "users", "_schema.json")
	_, err = os.Stat(schemaPath)
	assert.NoError(t, err)
}

func TestFileLayout_RecordFile(t *testing.T) {
	root := t.TempDir()
	d, err := xdbfs.NewDriver(root, xdbfs.Options{})
	require.NoError(t, err)
	s := store.New(d)

	ctx := context.Background()
	record := core.NewRecord("myapp", "users", "user-1").
		Set("name", "Alice")

	err = s.CreateRecord(ctx, record)
	require.NoError(t, err)

	// Verify <id>.json exists at expected path.
	recordPath := filepath.Join(root, "myapp", "users", "user-1.json")
	_, err = os.Stat(recordPath)
	assert.NoError(t, err)
}

func TestDeleteSchema_CleansEmptyDirs(t *testing.T) {
	root := t.TempDir()
	d, err := xdbfs.NewDriver(root, xdbfs.Options{})
	require.NoError(t, err)
	s := store.New(d)

	ctx := context.Background()
	uri := core.MustNewURI("cleanup-ns", "only-schema")
	def := &schema.Def{URI: uri, Mode: schema.ModeFlexible}

	err = s.CreateSchema(ctx, uri, def)
	require.NoError(t, err)

	err = s.DeleteSchema(ctx, uri)
	require.NoError(t, err)

	// Schema dir should be gone.
	_, err = os.Stat(filepath.Join(root, "cleanup-ns", "only-schema"))
	assert.True(t, os.IsNotExist(err))

	// Namespace dir should also be gone.
	_, err = os.Stat(filepath.Join(root, "cleanup-ns"))
	assert.True(t, os.IsNotExist(err))
}
