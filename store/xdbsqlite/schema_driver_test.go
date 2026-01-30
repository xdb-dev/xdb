package xdbsqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

type SchemaStoreTestSuite struct {
	suite.Suite
	*tests.SchemaStoreTestSuite
}

func TestSchemaStoreTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaStoreTestSuite))
}

func (s *SchemaStoreTestSuite) SetupTest() {
	cfg := xdbsqlite.Config{
		InMemory: true,
	}

	store, err := xdbsqlite.New(cfg)
	require.NoError(s.T(), err)

	s.T().Cleanup(func() {
		_ = store.Close()
	})

	s.SchemaStoreTestSuite = tests.NewSchemaStoreTestSuite(store)
}

func (s *SchemaStoreTestSuite) TestBasic() {
	s.SchemaStoreTestSuite.Basic(s.T())
}

func (s *SchemaStoreTestSuite) TestListSchemas() {
	s.SchemaStoreTestSuite.ListSchemas(s.T())
}

func (s *SchemaStoreTestSuite) TestListNamespaces() {
	s.SchemaStoreTestSuite.ListNamespaces(s.T())
}

func (s *SchemaStoreTestSuite) TestAddNewFields() {
	s.SchemaStoreTestSuite.AddNewFields(s.T())
}

func (s *SchemaStoreTestSuite) TestDropFields() {
	s.SchemaStoreTestSuite.DropFields(s.T())
}

func (s *SchemaStoreTestSuite) TestModifyFields() {
	s.SchemaStoreTestSuite.ModifyFields(s.T())
}

func (s *SchemaStoreTestSuite) TestEdgeCases() {
	s.SchemaStoreTestSuite.EdgeCases(s.T())
}

func TestSchemaCache(t *testing.T) {
	t.Run("LoadSchemasOnStartup", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		cfg := xdbsqlite.Config{
			Dir:  tmpDir,
			Name: "test.db",
		}

		ctx := context.Background()

		store1, err := xdbsqlite.New(cfg)
		require.NoError(t, err)

		testSchema := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "users",
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
				{Name: "age", Type: core.TypeInt},
			},
			Mode: schema.ModeStrict,
		}

		uri := core.MustParseURI("xdb://com.example/users")
		err = store1.PutSchema(ctx, uri, testSchema)
		require.NoError(t, err)

		retrieved1, err := store1.GetSchema(ctx, uri)
		require.NoError(t, err)
		tests.AssertDefEqual(t, testSchema, retrieved1)

		err = store1.Close()
		require.NoError(t, err)

		require.FileExists(t, dbPath)

		store2, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store2.Close()

		retrieved2, err := store2.GetSchema(ctx, uri)
		require.NoError(t, err)
		tests.AssertDefEqual(t, testSchema, retrieved2)
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		testSchema := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "products",
			Fields: []*schema.FieldDef{
				{Name: "title", Type: core.TypeString},
				{Name: "price", Type: core.TypeFloat},
			},
			Mode: schema.ModeStrict,
		}

		uri := core.MustParseURI("xdb://com.example/products")
		err = store.PutSchema(ctx, uri, testSchema)
		require.NoError(t, err)

		const numReaders = 10
		var wg sync.WaitGroup
		wg.Add(numReaders)

		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()

				retrieved, err := store.GetSchema(ctx, uri)
				require.NoError(t, err)
				require.NotNil(t, retrieved)
				require.Equal(t, "products", retrieved.Name)
				require.Equal(t, "com.example", retrieved.NS.String())
			}()
		}

		wg.Wait()
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		const numSchemas = 10
		var wg sync.WaitGroup
		wg.Add(numSchemas)

		for i := 0; i < numSchemas; i++ {
			go func(idx int) {
				defer wg.Done()

				schemaName := strconv.Itoa(idx + 1000)
				testSchema := &schema.Def{
					NS:   core.NewNS("com.concurrent"),
					Name: schemaName,
					Fields: []*schema.FieldDef{
						{Name: "field1", Type: core.TypeString},
						{Name: "field2", Type: core.TypeInt},
					},
					Mode: schema.ModeStrict,
				}

				uri := core.MustParseURI("xdb://com.concurrent/" + schemaName)
				err := store.PutSchema(ctx, uri, testSchema)
				require.NoError(t, err)
			}(i)
		}

		wg.Wait()

		nsURI := core.MustParseURI("xdb://com.concurrent")
		schemas, err := store.ListSchemas(ctx, nsURI)
		require.NoError(t, err)
		require.Len(t, schemas, numSchemas)
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		initialSchema := &schema.Def{
			NS:   core.NewNS("com.readwrite"),
			Name: "mixed",
			Fields: []*schema.FieldDef{
				{Name: "data", Type: core.TypeString},
			},
			Mode: schema.ModeStrict,
		}

		uri := core.MustParseURI("xdb://com.readwrite/mixed")
		err = store.PutSchema(ctx, uri, initialSchema)
		require.NoError(t, err)

		const numReaders = 5
		const numWriters = 5
		var wg sync.WaitGroup
		wg.Add(numReaders + numWriters)

		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()

				for j := 0; j < 10; j++ {
					retrieved, err := store.GetSchema(ctx, uri)
					require.NoError(t, err)
					require.NotNil(t, retrieved)
				}
			}()
		}

		for i := 0; i < numWriters; i++ {
			go func(idx int) {
				defer wg.Done()

				schemaName := strconv.Itoa(idx + 2000)
				testSchema := &schema.Def{
					NS:   core.NewNS("com.readwrite"),
					Name: schemaName,
					Fields: []*schema.FieldDef{
						{Name: "field", Type: core.TypeString},
					},
					Mode: schema.ModeStrict,
				}

				newURI := core.MustParseURI("xdb://com.readwrite/" + schemaName)
				err := store.PutSchema(ctx, newURI, testSchema)
				require.NoError(t, err)
			}(i)
		}

		wg.Wait()
	})

	t.Run("CacheInvalidationOnDelete", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		testSchema := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "temporary",
			Fields: []*schema.FieldDef{
				{Name: "data", Type: core.TypeString},
			},
			Mode: schema.ModeStrict,
		}

		uri := core.MustParseURI("xdb://com.example/temporary")
		err = store.PutSchema(ctx, uri, testSchema)
		require.NoError(t, err)

		retrieved, err := store.GetSchema(ctx, uri)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		err = store.DeleteSchema(ctx, uri)
		require.NoError(t, err)

		retrieved, err = store.GetSchema(ctx, uri)
		require.Error(t, err)
		require.Nil(t, retrieved)
	})

	t.Run("CacheConsistencyAfterUpdate", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()

		originalSchema := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "evolving",
			Fields: []*schema.FieldDef{
				{Name: "field1", Type: core.TypeString},
			},
			Mode: schema.ModeStrict,
		}

		uri := core.MustParseURI("xdb://com.example/evolving")
		err = store.PutSchema(ctx, uri, originalSchema)
		require.NoError(t, err)

		updatedSchema := originalSchema.Clone()
		updatedSchema.Fields = append(updatedSchema.Fields,
			&schema.FieldDef{Name: "field2", Type: core.TypeInt},
		)

		err = store.PutSchema(ctx, uri, updatedSchema)
		require.NoError(t, err)

		retrieved, err := store.GetSchema(ctx, uri)
		require.NoError(t, err)
		tests.AssertDefEqual(t, updatedSchema, retrieved)
		require.Len(t, retrieved.Fields, 2)
	})

	t.Run("EmptyCacheOnNewStore", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		defer store.Close()

		ctx := context.Background()
		uri := core.MustParseURI("xdb://com.example/nonexistent")

		retrieved, err := store.GetSchema(ctx, uri)
		require.Error(t, err)
		require.Nil(t, retrieved)

		namespaces, err := store.ListNamespaces(ctx)
		require.NoError(t, err)
		require.Empty(t, namespaces)
	})

	t.Run("MultiplePersistentStoresIsolation", func(t *testing.T) {
		tmpDir1 := t.TempDir()
		tmpDir2 := t.TempDir()

		cfg1 := xdbsqlite.Config{
			Dir:  tmpDir1,
			Name: "store1.db",
		}
		cfg2 := xdbsqlite.Config{
			Dir:  tmpDir2,
			Name: "store2.db",
		}

		ctx := context.Background()

		store1, err := xdbsqlite.New(cfg1)
		require.NoError(t, err)
		defer store1.Close()

		store2, err := xdbsqlite.New(cfg2)
		require.NoError(t, err)
		defer store2.Close()

		schema1 := &schema.Def{
			NS:   core.NewNS("com.store1"),
			Name: "data",
			Fields: []*schema.FieldDef{
				{Name: "field1", Type: core.TypeString},
			},
			Mode: schema.ModeStrict,
		}

		schema2 := &schema.Def{
			NS:   core.NewNS("com.store2"),
			Name: "data",
			Fields: []*schema.FieldDef{
				{Name: "field2", Type: core.TypeInt},
			},
			Mode: schema.ModeStrict,
		}

		uri1 := core.MustParseURI("xdb://com.store1/data")
		uri2 := core.MustParseURI("xdb://com.store2/data")

		err = store1.PutSchema(ctx, uri1, schema1)
		require.NoError(t, err)

		err = store2.PutSchema(ctx, uri2, schema2)
		require.NoError(t, err)

		retrieved1, err := store1.GetSchema(ctx, uri1)
		require.NoError(t, err)
		tests.AssertDefEqual(t, schema1, retrieved1)

		retrieved2, err := store2.GetSchema(ctx, uri2)
		require.NoError(t, err)
		tests.AssertDefEqual(t, schema2, retrieved2)

		_, err = store1.GetSchema(ctx, uri2)
		require.Error(t, err)

		_, err = store2.GetSchema(ctx, uri1)
		require.Error(t, err)
	})
}

func TestSchemaStoreBasics(t *testing.T) {
	t.Run("CreateStoreWithMemory", func(t *testing.T) {
		cfg := xdbsqlite.Config{
			InMemory: true,
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		require.NotNil(t, store)

		err = store.Close()
		require.NoError(t, err)
	})

	t.Run("CreateStoreWithFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		cfg := xdbsqlite.Config{
			Dir:  tmpDir,
			Name: "test.db",
		}

		store, err := xdbsqlite.New(cfg)
		require.NoError(t, err)
		require.NotNil(t, store)

		err = store.Close()
		require.NoError(t, err)

		_, err = os.Stat(dbPath)
		require.NoError(t, err)
	})
}
