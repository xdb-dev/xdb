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

		assert.Equal(t, "post-1", resp.Data.ID().String())
		assert.NotNil(t, resp.Data.Get("title"))
		assert.Equal(t, "Hello", resp.Data.Get("title").Value().String())
	})

	t.Run("idempotent create returns existing", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Different"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello", resp.Data.Get("title").Value().String())
	})

	t.Run("new record without data", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI: "xdb://com.example/posts/post-2",
		})
		require.NoError(t, err)
		assert.Equal(t, "post-2", resp.Data.ID().String())
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
		assert.Equal(t, "Hello", resp.Data.Get("title").Value().String())
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
		assert.Equal(t, "Updated", resp.Data.Get("title").Value().String())
		assert.Equal(t, "Alice", resp.Data.Get("author").Value().String())
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
		assert.Equal(t, "Created", resp.Data.Get("title").Value().String())
	})

	t.Run("replace existing", func(t *testing.T) {
		resp, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  "xdb://com.example/posts/post-1",
			Data: json.RawMessage(`{"title":"Replaced"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, "Replaced", resp.Data.Get("title").Value().String())

		getResp, err := svc.Get(ctx, &api.GetRecordRequest{
			URI: "xdb://com.example/posts/post-1",
		})
		require.NoError(t, err)
		assert.Equal(t, "Replaced", getResp.Data.Get("title").Value().String())
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
