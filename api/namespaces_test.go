package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func TestNamespaceService_Get(t *testing.T) {
	mem := xdbmemory.New()
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
		assert.Equal(t, "testns", resp.Data.String())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "xdb://unknown",
		})
		require.Error(t, err)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := nsSvc.Get(ctx, &api.GetNamespaceRequest{
			URI: "bad",
		})
		require.Error(t, err)
	})
}

func TestNamespaceService_List(t *testing.T) {
	mem := xdbmemory.New()
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
