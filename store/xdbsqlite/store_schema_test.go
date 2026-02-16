package xdbsqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func createTestStore(t *testing.T) *Store {
	t.Helper()
	cfg := Config{
		Dir:      t.TempDir(),
		Name:     "test.db",
		InMemory: true,
	}
	store, err := New(cfg)
	require.NoError(t, err)
	return store
}

func TestStoreGetSchemaCache(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()

	ctx := context.Background()

	ns := core.NewNS("test")
	uri := core.MustParseURI("xdb://test/User")
	def := &schema.Def{
		NS:   ns,
		Name: "User",
		Mode: schema.ModeFlexible,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	err := store.PutSchema(ctx, uri, def)
	require.NoError(t, err)

	t.Run("CacheHit", func(t *testing.T) {
		got, err := store.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, "User", got.Name)
		assert.Equal(t, ns.String(), got.NS.String())
	})

	t.Run("CacheMissNotFound", func(t *testing.T) {
		missingURI := core.MustParseURI("xdb://test/Missing")
		_, err := store.GetSchema(ctx, missingURI)
		assert.Error(t, err)
	})
}

func TestStoreListNamespacesCache(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("EmptyCache", func(t *testing.T) {
		namespaces, err := store.ListNamespaces(ctx)
		require.NoError(t, err)
		assert.Empty(t, namespaces)
	})

	t.Run("MultipleNamespaces", func(t *testing.T) {
		ns1 := core.NewNS("app1")
		ns2 := core.NewNS("app2")

		uri1 := core.MustParseURI("xdb://app1/User")
		uri2 := core.MustParseURI("xdb://app2/Product")

		def1 := &schema.Def{
			NS:   ns1,
			Name: "User",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		def2 := &schema.Def{
			NS:   ns2,
			Name: "Product",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "title", Type: core.TypeString},
			},
		}

		require.NoError(t, store.PutSchema(ctx, uri1, def1))
		require.NoError(t, store.PutSchema(ctx, uri2, def2))

		namespaces, err := store.ListNamespaces(ctx)
		require.NoError(t, err)
		assert.Len(t, namespaces, 2)

		assert.Equal(t, "app1", namespaces[0].String())
		assert.Equal(t, "app2", namespaces[1].String())
	})
}

func TestStoreListSchemasCache(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("EmptyNamespace", func(t *testing.T) {
		uri := core.MustParseURI("xdb://empty")
		schemas, err := store.ListSchemas(ctx, uri)
		require.NoError(t, err)
		assert.Nil(t, schemas)
	})

	t.Run("MultipleSchemas", func(t *testing.T) {
		ns := core.NewNS("test")

		uri1 := core.MustParseURI("xdb://test/User")
		uri2 := core.MustParseURI("xdb://test/Product")

		def1 := &schema.Def{
			NS:   ns,
			Name: "User",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		def2 := &schema.Def{
			NS:   ns,
			Name: "Product",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "title", Type: core.TypeString},
			},
		}

		require.NoError(t, store.PutSchema(ctx, uri1, def1))
		require.NoError(t, store.PutSchema(ctx, uri2, def2))

		listURI := core.MustParseURI("xdb://test")
		schemas, err := store.ListSchemas(ctx, listURI)
		require.NoError(t, err)
		assert.Len(t, schemas, 2)

		assert.Equal(t, "Product", schemas[0].Name)
		assert.Equal(t, "User", schemas[1].Name)
	})
}

func TestStorePutSchemaCache(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("NewSchemaCaching", func(t *testing.T) {
		ns := core.NewNS("test")
		uri := core.MustParseURI("xdb://test/User")
		def := &schema.Def{
			NS:   ns,
			Name: "User",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}

		err := store.PutSchema(ctx, uri, def)
		require.NoError(t, err)

		got, err := store.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, "User", got.Name)
	})

	t.Run("UpdateSchemaCaching", func(t *testing.T) {
		ns := core.NewNS("test")
		uri := core.MustParseURI("xdb://test/Product")
		def := &schema.Def{
			NS:   ns,
			Name: "Product",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "title", Type: core.TypeString},
			},
		}

		require.NoError(t, store.PutSchema(ctx, uri, def))

		def.Description = "Updated"
		require.NoError(t, store.PutSchema(ctx, uri, def))

		got, err := store.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.Description)
	})
}

func TestStoreDeleteSchemaCache(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("DeleteRemovesFromCache", func(t *testing.T) {
		ns := core.NewNS("test")
		uri := core.MustParseURI("xdb://test/User")
		def := &schema.Def{
			NS:   ns,
			Name: "User",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}

		require.NoError(t, store.PutSchema(ctx, uri, def))

		err := store.DeleteSchema(ctx, uri)
		require.NoError(t, err)

		_, err = store.GetSchema(ctx, uri)
		assert.Error(t, err)
	})

	t.Run("NamespaceCleanup", func(t *testing.T) {
		ns := core.NewNS("cleanup")
		uri := core.MustParseURI("xdb://cleanup/OnlySchema")
		def := &schema.Def{
			NS:   ns,
			Name: "OnlySchema",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "data", Type: core.TypeString},
			},
		}

		require.NoError(t, store.PutSchema(ctx, uri, def))

		namespaces, _ := store.ListNamespaces(ctx)
		assert.Contains(t, namespaceStrings(namespaces), "cleanup")

		require.NoError(t, store.DeleteSchema(ctx, uri))

		namespaces, _ = store.ListNamespaces(ctx)
		assert.NotContains(t, namespaceStrings(namespaces), "cleanup")
	})
}

func namespaceStrings(namespaces []*core.NS) []string {
	result := make([]string, len(namespaces))
	for i, ns := range namespaces {
		result[i] = ns.String()
	}
	return result
}
