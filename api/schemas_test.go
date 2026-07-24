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
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func newSchemaService() *api.SchemaService {
	return api.NewSchemaService(store.New(xdbmemory.NewDriver()))
}

// wireField is the JSON wire shape of a schema field, mirroring the schema
// package's own format ({type, elem_type, ...}).
type wireField struct {
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// schemaData builds JSON data for a schema definition without the URI field.
func schemaData(t *testing.T, fields map[string]wireField, mode schema.Mode) json.RawMessage {
	t.Helper()

	payload := struct {
		Fields map[string]wireField `json:"fields,omitempty"`
		Mode   schema.Mode          `json:"mode,omitempty"`
	}{
		Fields: fields,
		Mode:   mode,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	return data
}

// schemaDataWithRevision builds JSON data for a schema definition carrying
// an explicit revision, for exercising the update CAS.
func schemaDataWithRevision(t *testing.T, fields map[string]wireField, mode schema.Mode, revision int64) json.RawMessage {
	t.Helper()

	payload := struct {
		Fields   map[string]wireField `json:"fields,omitempty"`
		Mode     schema.Mode          `json:"mode,omitempty"`
		Revision int64                `json:"revision,omitempty"`
	}{
		Fields:   fields,
		Mode:     mode,
		Revision: revision,
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	return data
}

func createTestSchema(t *testing.T, svc *api.SchemaService, uri string) *schema.Def {
	t.Helper()

	data := schemaData(t, map[string]wireField{
		"name": {Type: "string", Required: true},
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
		data := schemaData(t, map[string]wireField{
			"title": {Type: "string", Required: true},
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

	t.Run("identical payload is idempotent", func(t *testing.T) {
		data := schemaData(t, map[string]wireField{
			"name": {Type: "string", Required: true},
		}, schema.ModeFlexible)

		resp1, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/users",
			Data: data,
		})
		require.NoError(t, err)

		// Create again with the exact same payload — idempotent success.
		resp2, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/users",
			Data: data,
		})
		require.NoError(t, err)
		assert.Equal(t, resp1.Data.URI.String(), resp2.Data.URI.String())
	})

	t.Run("divergent payload conflicts", func(t *testing.T) {
		data := schemaData(t, map[string]wireField{
			"name":  {Type: "string", Required: true},
			"email": {Type: "string"},
		}, schema.ModeFlexible)

		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/users",
			Data: data,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrConflict)
		assert.Contains(t, err.Error(), "xdb://myapp/users")
		assert.Contains(t, err.Error(), "schemas.update")
	})

	t.Run("no-data create over existing no-data schema is idempotent", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/blank",
			Data: json.RawMessage(`{}`),
		})
		require.NoError(t, err)

		resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp/blank",
			Data: json.RawMessage(`{}`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)
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

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://myapp",
			Data: json.RawMessage(`{}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestSchemaService_CreateWithIndexedUnique(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	data := json.RawMessage(`{
		"mode": "strict",
		"fields": {
			"email":  {"type": "string", "unique": true},
			"status": {"type": "string", "indexed": true},
			"name":   {"type": "string"}
		}
	}`)

	resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://myapp/members",
		Data: data,
	})
	require.NoError(t, err)

	assert.True(t, resp.Data.Fields["email"].Unique, "email.Unique preserved")
	assert.True(t, resp.Data.Fields["status"].Indexed, "status.Indexed preserved")
	assert.False(t, resp.Data.Fields["name"].Indexed, "name.Indexed")

	// Survives a round-trip through the store.
	getResp, err := svc.Get(ctx, &api.GetSchemaRequest{URI: "xdb://myapp/members"})
	require.NoError(t, err)
	assert.True(t, getResp.Data.Fields["email"].Unique, "email.Unique persisted")
	assert.True(t, getResp.Data.Fields["status"].Indexed, "status.Indexed persisted")
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

	t.Run("wrong depth returns invalid uri not not-found", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetSchemaRequest{
			URI: "xdb://myapp",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
		assert.NotErrorIs(t, err, core.ErrNotFound)
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

	t.Run("schema depth rejected", func(t *testing.T) {
		_, err := svc.List(ctx, &api.ListSchemasRequest{
			URI: "xdb://ns1/alpha",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestSchemaService_Update(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("patch merges fields", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/docs")

		data := schemaData(t, map[string]wireField{
			"author": {Type: "string", Required: false},
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
		data := schemaData(t, map[string]wireField{}, "")

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

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI:  "xdb://myapp",
			Data: json.RawMessage(`{}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})

	t.Run("matching revision bumps", func(t *testing.T) {
		def := createTestSchema(t, svc, "xdb://myapp/rev-match")
		require.Equal(t, int64(1), def.Revision)

		data := schemaDataWithRevision(t, map[string]wireField{
			"author": {Type: "string"},
		}, "", def.Revision)

		resp, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI:  "xdb://myapp/rev-match",
			Data: data,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), resp.Data.Revision)
	})

	t.Run("stale revision conflicts", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/rev-stale")

		// Bump to revision 2 using the correct base.
		_, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI: "xdb://myapp/rev-stale",
			Data: schemaDataWithRevision(t, map[string]wireField{
				"a": {Type: "string"},
			}, "", 1),
		})
		require.NoError(t, err)

		// Re-using the now-stale base revision 1 must conflict.
		_, err = svc.Update(ctx, &api.UpdateSchemaRequest{
			URI: "xdb://myapp/rev-stale",
			Data: schemaDataWithRevision(t, map[string]wireField{
				"b": {Type: "string"},
			}, "", 1),
		})
		assert.ErrorIs(t, err, core.ErrConflict)
	})

	t.Run("omitted revision is unconditional", func(t *testing.T) {
		createTestSchema(t, svc, "xdb://myapp/rev-zero")

		resp, err := svc.Update(ctx, &api.UpdateSchemaRequest{
			URI: "xdb://myapp/rev-zero",
			Data: schemaData(t, map[string]wireField{
				"c": {Type: "string"},
			}, ""),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), resp.Data.Revision)
	})
}

func TestSchemaService_CreateWithItems(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	data := json.RawMessage(`{
		"fields": {
			"tags": {
				"type": "array",
				"elem_type": "json",
				"items": {
					"name": {"type": "string", "required": true}
				}
			}
		}
	}`)

	resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://myapp/tagged",
		Data: data,
	})
	require.NoError(t, err)
	require.Contains(t, resp.Data.Fields, "tags")
	require.Contains(t, resp.Data.Fields["tags"].Items, "name")
	assert.True(t, resp.Data.Fields["tags"].Items["name"].Required)

	// Round-trip through Get: items must survive persistence.
	getResp, err := svc.Get(ctx, &api.GetSchemaRequest{URI: "xdb://myapp/tagged"})
	require.NoError(t, err)
	require.Contains(t, getResp.Data.Fields, "tags")
	require.Contains(t, getResp.Data.Fields["tags"].Items, "name")
	assert.True(t, getResp.Data.Fields["tags"].Items["name"].Required)
}

// TestSchemaService_ItemsEnforceMemberConstraints proves the end-to-end
// chain: an items member constraint declared via the schemas.create payload
// is enforced when records.create writes a violating value.
func TestSchemaService_ItemsEnforceMemberConstraints(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	schemas := api.NewSchemaService(s)
	records := api.NewRecordService(s)
	ctx := context.Background()

	data := json.RawMessage(`{
		"fields": {
			"tags": {
				"type": "array",
				"elem_type": "json",
				"items": {
					"name": {"type": "string", "required": true}
				}
			}
		}
	}`)

	_, err := schemas.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://myapp/tagged-records",
		Data: data,
	})
	require.NoError(t, err)

	// "name" must be a string; supplying a number violates the item schema.
	_, err = records.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://myapp/tagged-records/rec-1",
		Data: json.RawMessage(`{"tags":[{"name":123}]}`),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrSchemaViolation)
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

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI: "xdb://myapp",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestSchemaService_DryRun(t *testing.T) {
	svc := newSchemaService()
	ctx := context.Background()

	t.Run("create validates without writing", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:    "xdb://dry.ns/posts",
			Data:   json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.True(t, resp.DryRun.Valid)
		assert.Equal(t, "create", resp.DryRun.Would)

		_, err = svc.Get(ctx, &api.GetSchemaRequest{URI: "xdb://dry.ns/posts"})
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("create dry-run rejects invalid definitions", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:    "xdb://dry.ns/bad",
			Data:   json.RawMessage(`{"mode":"bogus","fields":{"title":{"type":"string"}}}`),
			DryRun: true,
		})
		require.Error(t, err)
	})

	t.Run("create dry-run over identical existing is a noop", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://dry.ns/live",
			Data: json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
		})
		require.NoError(t, err)

		resp, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:    "xdb://dry.ns/live",
			Data:   json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "noop", resp.DryRun.Would)
	})

	t.Run("create dry-run over divergent existing conflicts", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateSchemaRequest{
			URI:    "xdb://dry.ns/live",
			Data:   json.RawMessage(`{"fields":{"title":{"type":"string"},"extra":{"type":"string"}}}`),
			DryRun: true,
		})
		assert.ErrorIs(t, err, core.ErrConflict)
	})

	t.Run("delete dry-run preserves the schema", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI:    "xdb://dry.ns/live",
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "delete", resp.DryRun.Would)

		_, err = svc.Get(ctx, &api.GetSchemaRequest{URI: "xdb://dry.ns/live"})
		assert.NoError(t, err)
	})

	t.Run("delete dry-run on missing schema is a noop", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteSchemaRequest{
			URI:    "xdb://dry.ns/ghost",
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "noop", resp.DryRun.Would)
	})
}
