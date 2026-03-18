package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func newSchemaService() *api.SchemaService {
	return api.NewSchemaService(xdbmemory.New())
}

// schemaData builds JSON data for a schema definition without the URI field.
func schemaData(t *testing.T, fields map[string]schema.FieldDef, mode schema.Mode) json.RawMessage {
	t.Helper()

	payload := struct {
		Fields map[string]schema.FieldDef `json:"Fields,omitempty"`
		Mode   schema.Mode               `json:"Mode,omitempty"`
	}{
		Fields: fields,
		Mode:   mode,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	return data
}

func createTestSchema(t *testing.T, svc *api.SchemaService, uri string) *schema.Def {
	t.Helper()

	data := schemaData(t, map[string]schema.FieldDef{
		"name": {Type: core.TIDString, Required: true},
	}, schema.ModeStrict)

	resp, err := svc.Create(context.Background(), &api.CreateSchemaRequest{
		URI:  uri,
		Data: data,
	})
	require.NoError(t, err)

	return resp.Data
}

func TestSchemaService_Create(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		data := schemaData(t, map[string]schema.FieldDef{
			"title": {Type: core.TIDString, Required: true},
		}, schema.ModeStrict)

		resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/articles",
			Data: data,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)

		assert.Equal(t, "xdb://myapp/articles", resp.Data.URI.String())
		assert.Equal(t, schema.ModeStrict, resp.Data.Mode)
		assert.Contains(t, resp.Data.Fields, "title")
	})

	t.Run("idempotent create returns existing", func(t *testing.T) {
		data := schemaData(t, map[string]schema.FieldDef{
			"name": {Type: core.TIDString, Required: true},
		}, schema.ModeFlexible)

		resp1, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/users",
			Data: data,
		})
		require.NoError(t, err)

		// Create again — should return existing.
		resp2, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/users",
			Data: data,
		})
		require.NoError(t, err)
		assert.Equal(t, resp1.Data.URI.String(), resp2.Data.URI.String())
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "not-a-uri",
			Data: json.RawMessage(`{}`),
		})
		require.Error(t, err)
	})

	t.Run("invalid JSON data", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/broken",
			Data: json.RawMessage(`{invalid`),
		})
		require.Error(t, err)
	})
}

func TestSchemaService_Get(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/items")

		resp, err := svc.Get(ctx, &api.GetSchemaRequest{
			URI: "xdb://myapp/items",
		})
		require.NoError(t, err)
		assert.Equal(t, "xdb://myapp/items", resp.Data.URI.String())
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetSchemaRequest{
			URI: "xdb://myapp/nonexistent",
		})
		require.Error(t, err)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetSchemaRequest{
			URI: "bad",
		})
		require.Error(t, err)
	})
}

func TestSchemaService_List(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	// Create several schemas across two namespaces.
	createTestSchema(t, svc, "xdb://ns1/alpha")
	createTestSchema(t, svc, "xdb://ns1/beta")
	createTestSchema(t, svc, "xdb://ns1/gamma")
	createTestSchema(t, svc, "xdb://ns2/delta")

	t.Run("list all in namespace", func(t *testing.T) {
		resp, err := svc.List(ctx, &api.ListSchemasRequest{
			URI: "xdb://ns1",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 3)
	})

	t.Run("pagination", func(t *testing.T) {
		resp, err := svc.List(ctx, &api.ListSchemasRequest{
			URI:   "xdb://ns1",
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, 3, resp.Total)
		assert.NotZero(t, resp.NextOffset)

		// Fetch next page.
		resp2, err := svc.List(ctx, &api.ListSchemasRequest{
			URI:    "xdb://ns1",
			Limit:  2,
			Offset: resp.NextOffset,
		})
		require.NoError(t, err)
		assert.Len(t, resp2.Items, 1)
		assert.Zero(t, resp2.NextOffset)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := svc.List(ctx, &api.ListSchemasRequest{
			URI: "bad",
		})
		require.Error(t, err)
	})
}

func TestSchemaService_Update(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("patch merges fields", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/docs")

		data := schemaData(t, map[string]schema.FieldDef{
			"author": {Type: core.TIDString, Required: false},
		}, "")

		resp, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI:  "xdb://myapp/docs",
			Data: data,
		})
		require.NoError(t, err)

		// Original field still present.
		assert.Contains(t, resp.Data.Fields, "name")
		// New field added.
		assert.Contains(t, resp.Data.Fields, "author")
	})

	t.Run("not found", func(t *testing.T) {
		data := schemaData(t, map[string]schema.FieldDef{}, "")

		_, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI:  "xdb://myapp/missing",
			Data: data,
		})
		require.Error(t, err)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI:  "bad",
			Data: json.RawMessage(`{}`),
		})
		require.Error(t, err)
	})
}

func TestSchemaService_Delete(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/temp")

		resp, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI: "xdb://myapp/temp",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify it's gone.
		_, err = svc.Get(ctx, &api.GetSchemaRequest{
			URI: "xdb://myapp/temp",
		})
		require.Error(t, err)
	})

	t.Run("idempotent delete", func(t *testing.T) {
		// Delete something that doesn't exist — should succeed.
		resp, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI: "xdb://myapp/ghost",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI: "bad",
		})
		require.Error(t, err)
	})
}
