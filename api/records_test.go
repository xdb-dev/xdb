package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
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
	s := xdbmemory.New()
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

	t.Run("idempotent create returns existing", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Different"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.Equal(t, "Hello", m["title"])
	})

	t.Run("new record without data", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-2",
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Data)
	})
}

func TestRecordService_Get(t *testing.T) {
	s := xdbmemory.New()
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
}

func TestRecordService_List(t *testing.T) {
	s := xdbmemory.New()
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
}

func TestRecordService_GetFields(t *testing.T) {
	s := xdbmemory.New()
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
	s := xdbmemory.New()
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
	s := xdbmemory.New()
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
}

func TestRecordService_Upsert(t *testing.T) {
	s := xdbmemory.New()
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
}

func TestRecordService_Delete(t *testing.T) {
	s := xdbmemory.New()
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
}
