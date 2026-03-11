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

func newTestStore(t *testing.T) *xdbfs.Store {
	t.Helper()
	s, err := xdbfs.New(t.TempDir(), xdbfs.Options{})
	require.NoError(t, err)
	return s
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	var _ store.Store = s
	var _ store.HealthChecker = s
}

func TestHealth(t *testing.T) {
	t.Run("valid_root", func(t *testing.T) {
		s := newTestStore(t)
		require.NoError(t, s.Health(context.Background()))
	})

	t.Run("nonexistent_root", func(t *testing.T) {
		s, err := xdbfs.New(t.TempDir(), xdbfs.Options{})
		require.NoError(t, err)

		// Remove the root after creation.
		os.RemoveAll(filepath.Dir(s.Root()) + "/nonexistent")
		// Create a store pointing to a path that doesn't exist.
		bad, err := xdbfs.New(filepath.Join(t.TempDir(), "gone"), xdbfs.Options{})
		require.NoError(t, err)
		os.RemoveAll(bad.Root())

		err = bad.Health(context.Background())
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

// --- FS-specific tests ---

func TestFileLayout_SchemaFile(t *testing.T) {
	root := t.TempDir()
	s, err := xdbfs.New(root, xdbfs.Options{})
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.New().NS("myapp").Schema("users").MustURI()
	def := &schema.Def{URI: uri}

	err = s.CreateSchema(ctx, uri, def)
	require.NoError(t, err)

	// Verify _schema.json exists at expected path.
	schemaPath := filepath.Join(root, "myapp", "users", "_schema.json")
	_, err = os.Stat(schemaPath)
	assert.NoError(t, err)
}

func TestFileLayout_RecordFile(t *testing.T) {
	root := t.TempDir()
	s, err := xdbfs.New(root, xdbfs.Options{})
	require.NoError(t, err)

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
	s, err := xdbfs.New(root, xdbfs.Options{})
	require.NoError(t, err)

	ctx := context.Background()
	uri := core.New().NS("cleanup-ns").Schema("only-schema").MustURI()
	def := &schema.Def{URI: uri}

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
