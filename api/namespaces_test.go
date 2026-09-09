package api_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func TestNamespaceService_Get(t *testing.T) {
	mem := store.New(xdbmemory.NewDriver())
	schemaSvc := api.NewSchemaService(mem)
	nsSvc := api.NewNamespaceService(mem)
	ctx := context.Background()

	// Create a schema so the namespace exists.
	createTestSchema(t, schemaSvc, "xdb://testns/things")

	t.Run("success", func(t *testing.T) {
		resp, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "xdb://testns",
		})
		require.NoError(t, err)
		assert.Equal(t, "testns", resp.Data)
	})

	// The store reports absence as (false, nil), so namespaces.get is the
	// layer that has to turn it into ErrNotFound. Assert on the sentinel:
	// omitting that translation returns 200 with an empty payload, which
	// compiles and still passes a bare require.Error.
	t.Run("not found", func(t *testing.T) {
		_, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "xdb://unknown",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "bad",
		})
		require.Error(t, err)
	})

	t.Run("wrong depth returns invalid uri not not-found", func(t *testing.T) {
		_, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "xdb://testns/things",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
		assert.NotErrorIs(t, err, core.ErrNotFound)
	})
}

func TestNamespaceService_List(t *testing.T) {
	mem := store.New(xdbmemory.NewDriver())
	schemaSvc := api.NewSchemaService(mem)
	nsSvc := api.NewNamespaceService(mem)
	ctx := context.Background()

	// Create schemas in multiple namespaces.
	createTestSchema(t, schemaSvc, "xdb://alpha/s1")
	createTestSchema(t, schemaSvc, "xdb://alpha/s2")
	createTestSchema(t, schemaSvc, "xdb://beta/s1")
	createTestSchema(t, schemaSvc, "xdb://gamma/s1")

	t.Run("list all", func(t *testing.T) {
		resp, err := nsSvc.List(ctx, &api.ListNamespacesRequest{})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 3)
	})

	t.Run("pagination", func(t *testing.T) {
		resp, err := nsSvc.List(ctx, &api.ListNamespacesRequest{
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, 3, resp.Total)
		assert.NotZero(t, resp.NextOffset)

		resp2, err := nsSvc.List(ctx, &api.ListNamespacesRequest{
			Limit:  2,
			Offset: resp.NextOffset,
		})
		require.NoError(t, err)
		assert.Len(t, resp2.Items, 1)
		assert.Zero(t, resp2.NextOffset)
	})
}

func TestNamespaceService_GetIncludesSchemas(t *testing.T) {
	mem := store.New(xdbmemory.NewDriver())
	schemaSvc := api.NewSchemaService(mem)
	nsSvc := api.NewNamespaceService(mem)
	ctx := context.Background()

	createTestSchema(t, schemaSvc, "xdb://tree.ns/posts")
	createTestSchema(t, schemaSvc, "xdb://tree.ns/authors")

	resp, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{URI: "xdb://tree.ns"})
	require.NoError(t, err)

	assert.Equal(t, "tree.ns", resp.Data)
	assert.Equal(t, 2, resp.TotalSchemas)
	assert.Equal(t, []string{"xdb://tree.ns/authors", "xdb://tree.ns/posts"}, resp.Schemas)
}

func TestNamespaceService_GetListsAllSchemas(t *testing.T) {
	mem := store.New(xdbmemory.NewDriver())
	schemaSvc := api.NewSchemaService(mem)
	nsSvc := api.NewNamespaceService(mem)
	ctx := context.Background()

	n := store.DefaultLimit + 5
	for i := range n {
		createTestSchema(t, schemaSvc, fmt.Sprintf("xdb://many.ns/s%03d", i))
	}

	resp, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{URI: "xdb://many.ns"})
	require.NoError(t, err)

	assert.Equal(t, n, resp.TotalSchemas)
	assert.Len(t, resp.Schemas, n, "Schemas must not stop at the default page size")
	assert.True(t, slices.IsSorted(resp.Schemas))
}
