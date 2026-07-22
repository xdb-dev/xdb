package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// recordData unmarshals a json.RawMessage response into a map for assertions.
func recordData(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))

	return m
}

func TestRecordService_Create(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	t.Run("new record with JSON data", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Hello","count":42}`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
		assert.Equal(t, float64(42), m["count"])
	})

	t.Run("identical payload is idempotent", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Hello","count":42}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
	})

	t.Run("divergent payload conflicts", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Different"}`),
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrConflict)
		assert.Contains(t, err.Error(), "xdb://com.example/posts/post-1")
		assert.Contains(t, err.Error(), "records.update")
	})

	t.Run("no-data create over existing no-data record is idempotent", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-nodata",
		})
		require.NoError(t, err)

		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-nodata",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)
	})

	t.Run("new record without data", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-2",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)
	})

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})

	t.Run("attr not accepted", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-3#title",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestRecordService_Get(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello"}`),
	})
	require.NoError(t, err)

	t.Run("existing record", func(t *testing.T) {
		resp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/missing",
		})
		require.Error(t, err)
	})

	t.Run("wrong depth returns invalid uri not not-found", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
		assert.NotErrorIs(t, err, core.ErrNotFound)
	})
}

func TestRecordService_List(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	for i, title := range []string{"Alpha", "Beta", "Gamma"} {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-" + string(rune('1'+i)),
			Data: json.RawMessage(`{"title":"` + title + `"}`),
		})
		require.NoError(t, err)
	}

	t.Run("all records", func(t *testing.T) {
		resp, err := svc.List(ctx, &api.ListRecordsRequest{
			URI: "xdb://com.example/posts",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Len(t, resp.Items, 3)
	})

	t.Run("pagination", func(t *testing.T) {
		resp, err := svc.List(ctx, &api.ListRecordsRequest{
			URI:   "xdb://com.example/posts",
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, 3, resp.Total)
		assert.Equal(t, 2, resp.NextOffset)
	})

	t.Run("record depth rejected", func(t *testing.T) {
		_, err := svc.List(ctx, &api.ListRecordsRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})

	t.Run("namespace depth gate opens (Phase 10 scope, gate only)", func(t *testing.T) {
		_, err := svc.List(ctx, &api.ListRecordsRequest{
			URI: "xdb://com.example",
		})
		assert.NotErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestRecordService_GetFields(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello","author":"Alice","count":5}`),
	})
	require.NoError(t, err)

	resp, err := svc.Get(ctx, &api.GetRecordRequest{
		URI:    "xdb://com.example/posts/post-1",
		Fields: []string{"title"},
	})
	require.NoError(t, err)

	m := recordData(t, resp.Data)
	assert.Equal(t, "post-1", m["_id"])
	assert.Equal(t, "Hello", m["title"])
	assert.NotContains(t, m, "author")
	assert.NotContains(t, m, "count")
}

func TestRecordService_ListFields(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	for _, title := range []string{"Alpha", "Beta"} {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/" + title,
			Data: json.RawMessage(`{"title":"` + title + `","author":"Bob"}`),
		})
		require.NoError(t, err)
	}

	resp, err := svc.List(ctx, &api.ListRecordsRequest{
		URI:    "xdb://com.example/posts",
		Fields: []string{"title"},
	})
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2)

	for _, item := range resp.Items {
		m := recordData(t, item)
		assert.Contains(t, m, "_id")
		assert.Contains(t, m, "title")
		assert.NotContains(t, m, "author")
	}
}

func TestRecordService_Update(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Original","author":"Alice"}`),
	})
	require.NoError(t, err)

	t.Run("patch merge preserves old fields", func(t *testing.T) {
		resp, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Updated"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Updated", m["title"])
		assert.Equal(t, "Alice", m["author"])
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:  "xdb://com.example/posts/missing",
			Data: json.RawMessage(`{"title":"Nope"}`),
		})
		require.Error(t, err)
	})

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:  "xdb://com.example/posts",
			Data: json.RawMessage(`{"title":"Nope"}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})

	t.Run("attr not accepted", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:  "xdb://com.example/posts/post-1#title",
			Data: json.RawMessage(`{"title":"Nope"}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestRecordService_Upsert(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	t.Run("create new via upsert", func(t *testing.T) {
		resp, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Created"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Created", m["title"])
	})

	t.Run("replace existing", func(t *testing.T) {
		resp, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Replaced"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Replaced", m["title"])

		getResp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)

		m = recordData(t, getResp.Data)
		assert.Equal(t, "Replaced", m["title"])
	})

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  "xdb://com.example/posts",
			Data: json.RawMessage(`{"title":"Nope"}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})

	t.Run("attr not accepted", func(t *testing.T) {
		_, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  "xdb://com.example/posts/post-1#title",
			Data: json.RawMessage(`{"title":"Nope"}`),
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestRecordService_GetTuple(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello","author":"Alice"}`),
	})
	require.NoError(t, err)

	t.Run("attr-level URI returns just that attr", func(t *testing.T) {
		resp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1#title",
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
		assert.Equal(t, "post-1", m["_id"])
		assert.NotContains(t, m, "author")
	})

	t.Run("absent attr is not found", func(t *testing.T) {
		_, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1#missing",
		})
		require.Error(t, err)
	})
}

func TestRecordService_DeleteTuple(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello","author":"Alice"}`),
	})
	require.NoError(t, err)

	t.Run("attr-level URI deletes just that attr", func(t *testing.T) {
		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI: "xdb://com.example/posts/post-1#author",
		})
		require.NoError(t, err)

		resp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
		assert.NotContains(t, m, "author")
	})

	t.Run("idempotent on absent attr", func(t *testing.T) {
		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI: "xdb://com.example/posts/post-1#missing",
		})
		require.NoError(t, err)
	})
}

func TestRecordService_Delete(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://com.example/posts/post-1",
		Data: json.RawMessage(`{"title":"Hello"}`),
	})
	require.NoError(t, err)

	t.Run("existing record", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("idempotent delete on non-existent", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("wrong depth returns invalid uri", func(t *testing.T) {
		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI: "xdb://com.example/posts",
		})
		assert.ErrorIs(t, err, core.ErrInvalidURI)
	})
}

func TestRecordService_DryRun(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()

	t.Run("create validates without writing", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:    "xdb://dry.ns/posts/p1",
			Data:   json.RawMessage(`{"title":"Hello"}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.True(t, resp.DryRun.Valid)
		assert.Equal(t, "create", resp.DryRun.Would)

		_, err = svc.Get(ctx, &api.GetRecordRequest{URI: "xdb://dry.ns/posts/p1"})
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("create over identical existing would be a noop", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://dry.ns/posts/p2",
			Data: json.RawMessage(`{"title":"Same"}`),
		})
		require.NoError(t, err)

		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:    "xdb://dry.ns/posts/p2",
			Data:   json.RawMessage(`{"title":"Same"}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "noop", resp.DryRun.Would)
	})

	t.Run("create over divergent existing conflicts", func(t *testing.T) {
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:    "xdb://dry.ns/posts/p2",
			Data:   json.RawMessage(`{"title":"Different"}`),
			DryRun: true,
		})
		assert.ErrorIs(t, err, core.ErrConflict)
	})

	t.Run("create dry-run surfaces schema violations", func(t *testing.T) {
		schemas := api.NewSchemaService(s)
		_, err := schemas.Create(ctx, &api.CreateSchemaRequest{
			URI:  "xdb://dry.ns/typed",
			Data: json.RawMessage(`{"fields":{"qty":{"type":"integer","required":true}}}`),
		})
		require.NoError(t, err)

		_, err = svc.Create(ctx, &api.CreateRecordRequest{
			URI:    "xdb://dry.ns/typed/t1",
			Data:   json.RawMessage(`{}`),
			DryRun: true,
		})
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("update dry-run on missing record is not found", func(t *testing.T) {
		_, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:    "xdb://dry.ns/posts/missing",
			Data:   json.RawMessage(`{"title":"X"}`),
			DryRun: true,
		})
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("update dry-run does not write", func(t *testing.T) {
		resp, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:    "xdb://dry.ns/posts/p2",
			Data:   json.RawMessage(`{"title":"Patched"}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "update", resp.DryRun.Would)

		got, err := svc.Get(ctx, &api.GetRecordRequest{URI: "xdb://dry.ns/posts/p2"})
		require.NoError(t, err)
		assert.Equal(t, "Same", recordData(t, got.Data)["title"])
	})

	t.Run("upsert dry-run reports replace vs create", func(t *testing.T) {
		resp, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:    "xdb://dry.ns/posts/p2",
			Data:   json.RawMessage(`{"title":"Replaced"}`),
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "replace", resp.DryRun.Would)

		resp, err = svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:    "xdb://dry.ns/posts/p9",
			Data:   json.RawMessage(`{"title":"New"}`),
			DryRun: true,
		})
		require.NoError(t, err)
		assert.Equal(t, "create", resp.DryRun.Would)
	})

	t.Run("delete dry-run preserves the record", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI:    "xdb://dry.ns/posts/p2",
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "delete", resp.DryRun.Would)

		_, err = svc.Get(ctx, &api.GetRecordRequest{URI: "xdb://dry.ns/posts/p2"})
		assert.NoError(t, err)
	})

	t.Run("delete dry-run on missing record is a noop", func(t *testing.T) {
		resp, err := svc.Delete(ctx, &api.DeleteRecordRequest{
			URI:    "xdb://dry.ns/posts/ghost",
			DryRun: true,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.DryRun)
		assert.Equal(t, "noop", resp.DryRun.Would)
	})

	t.Run("real ops carry no dry_run marker", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://dry.ns/posts/real1",
			Data: json.RawMessage(`{"title":"Real"}`),
		})
		require.NoError(t, err)
		assert.Nil(t, resp.DryRun)
	})
}

func TestRecordService_CreateCoercesTypedFields(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	schemas := api.NewSchemaService(s)
	svc := api.NewRecordService(s)
	ctx := context.Background()

	_, err := schemas.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://typed.ns/items",
		Data: json.RawMessage(`{"fields":{"name":{"type":"string"},"qty":{"type":"integer"}}}`),
	})
	require.NoError(t, err)

	// The decoder must find the schema def (keyed by the schema-level URI)
	// so a JSON number coerces to INTEGER instead of failing validation.
	resp, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://typed.ns/items/i1",
		Data: json.RawMessage(`{"name":"widget","qty":42}`),
	})
	require.NoError(t, err)

	m := recordData(t, resp.Data)
	assert.Equal(t, float64(42), m["qty"])
}
