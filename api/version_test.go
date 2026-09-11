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

// Every write response carries the version the store just stamped, so a
// client never has to re-read to learn what it wrote.
func TestRecordService_WriteResponsesCarryVersion(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()
	uri := "xdb://com.example/vposts/p1"

	t.Run("create responds with version 1", func(t *testing.T) {
		resp, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"Hello"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.InDelta(t, float64(1), m[schema.FieldVersion], 0.0001)
		assert.NotEmpty(t, m[schema.FieldUpdated])
	})

	t.Run("upsert responds with the bumped version", func(t *testing.T) {
		resp, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"Second"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.InDelta(t, float64(2), m[schema.FieldVersion], 0.0001)
	})

	t.Run("update responds with the bumped version", func(t *testing.T) {
		resp, err := svc.Update(ctx, &api.UpdateRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"Third"}`),
		})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.InDelta(t, float64(3), m[schema.FieldVersion], 0.0001)
	})

	t.Run("get agrees with the last write", func(t *testing.T) {
		resp, err := svc.Get(ctx, &api.GetRecordRequest{URI: uri})
		require.NoError(t, err)

		m := recordData(t, resp.Data)
		assert.InDelta(t, float64(3), m[schema.FieldVersion], 0.0001)
	})
}

// A version echoed in the payload is checked against the stored version.
// This test verifies that a stale copy conflicts after another write.
func TestRecordService_WritePrecondition(t *testing.T) {
	s := store.New(xdbmemory.NewDriver())
	svc := api.NewRecordService(s)
	ctx := context.Background()
	uri := "xdb://com.example/vposts/cas"

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  uri,
		Data: json.RawMessage(`{"title":"a"}`),
	})
	require.NoError(t, err)

	t.Run("matching version succeeds", func(t *testing.T) {
		_, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"b","_version":1}`),
		})
		require.NoError(t, err)
	})

	t.Run("stale version conflicts", func(t *testing.T) {
		_, err := svc.Upsert(ctx, &api.UpsertRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"c","_version":1}`),
		})
		require.ErrorIs(t, err, core.ErrConflict)
	})
}

// Delete is the one verb with no payload to carry a precondition, so it
// takes one as an explicit request field.
func TestRecordService_DeletePrecondition(t *testing.T) {
	ctx := context.Background()
	uri := "xdb://com.example/vposts/del"

	newSvc := func(t *testing.T) *api.RecordService {
		t.Helper()

		svc := api.NewRecordService(store.New(xdbmemory.NewDriver()))
		_, err := svc.Create(ctx, &api.CreateRecordRequest{
			URI:  uri,
			Data: json.RawMessage(`{"title":"a"}`),
		})
		require.NoError(t, err)

		return svc
	}

	t.Run("matching version deletes", func(t *testing.T) {
		svc := newSvc(t)

		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{URI: uri, Version: 1})
		require.NoError(t, err)

		_, err = svc.Get(ctx, &api.GetRecordRequest{URI: uri})
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("stale version conflicts and keeps the record", func(t *testing.T) {
		svc := newSvc(t)

		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{URI: uri, Version: 99})
		require.ErrorIs(t, err, core.ErrConflict)

		_, err = svc.Get(ctx, &api.GetRecordRequest{URI: uri})
		assert.NoError(t, err)
	})

	// The conflict carries the same tags as a stale write, so the caller
	// is told to re-read rather than to update or upsert.
	t.Run("stale version conflict says to re-read", func(t *testing.T) {
		svc := newSvc(t)

		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{URI: uri, Version: 99})
		require.ErrorIs(t, err, core.ErrConflict)

		tags := core.ErrorTags(err)
		assert.Equal(t, "99", tags["expected"])
		assert.Equal(t, "1", tags["got"])
		assert.Contains(t, tags["fix"], "re-read")
	})

	t.Run("omitted version deletes unconditionally", func(t *testing.T) {
		svc := newSvc(t)

		_, err := svc.Delete(ctx, &api.DeleteRecordRequest{URI: uri})
		require.NoError(t, err)
	})
}

// Watch events carry the record's version explicitly: delete events have
// no payload to read it from, and a consumer that sees the version jump
// knows the lossy bus dropped something.
func TestWatchEvents_CarryVersion(t *testing.T) {
	ctx := context.Background()

	bus := api.NewBus()
	defer bus.Close()

	svc := api.NewRecordService(store.New(xdbmemory.NewDriver()), api.WithEvents(bus))

	events, unsubscribe := bus.Subscribe(core.MustParseURI("xdb://com.example"))
	defer unsubscribe()

	uri := "xdb://com.example/vposts/w1"

	_, err := svc.Create(ctx, &api.CreateRecordRequest{
		URI:  uri,
		Data: json.RawMessage(`{"title":"a"}`),
	})
	require.NoError(t, err)

	created := <-events
	assert.Equal(t, "record.create", created.Type)
	assert.Equal(t, int64(1), created.Version)

	_, err = svc.Upsert(ctx, &api.UpsertRecordRequest{
		URI:  uri,
		Data: json.RawMessage(`{"title":"b"}`),
	})
	require.NoError(t, err)

	upserted := <-events
	assert.Equal(t, int64(2), upserted.Version)

	// Deletes have no payload, so the version is the only thing that
	// orders them against the writes before.
	_, err = svc.Delete(ctx, &api.DeleteRecordRequest{URI: uri})
	require.NoError(t, err)

	deleted := <-events
	assert.Equal(t, "record.delete", deleted.Type)
	assert.Equal(t, int64(2), deleted.Version)
}
