package api_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func watchEvent(uri string) api.WatchEvent {
	return api.WatchEvent{Type: "record.create", URI: uri}
}

func recvOne(t *testing.T, ch <-chan api.WatchEvent) api.WatchEvent {
	t.Helper()

	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return api.WatchEvent{}
	}
}

func TestBus_ScopeMatching(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		event string
		want  bool
	}{
		{"namespace scope matches child record", "xdb://ns", "xdb://ns/posts/p1", true},
		{"schema scope matches its records", "xdb://ns/posts", "xdb://ns/posts/p1", true},
		{"record scope matches itself", "xdb://ns/posts/p1", "xdb://ns/posts/p1", true},
		{"different namespace does not match", "xdb://ns", "xdb://other/posts/p1", false},
		{"sibling schema does not match", "xdb://ns/posts", "xdb://ns/users/u1", false},
		{"prefix name is not a match", "xdb://ns/post", "xdb://ns/posts/p1", false},
		{"other record does not match", "xdb://ns/posts/p1", "xdb://ns/posts/p2", false},
		{"schema scope matches schema-level event", "xdb://ns/posts", "xdb://ns/posts", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := api.NewBus()
			defer bus.Close()

			ch, cancel := bus.Subscribe(core.MustParseURI(tt.scope))
			defer cancel()

			bus.Publish(watchEvent(tt.event))

			if tt.want {
				assert.Equal(t, tt.event, recvOne(t, ch).URI)
			} else {
				select {
				case e := <-ch:
					t.Fatalf("unexpected event: %+v", e)
				case <-time.After(50 * time.Millisecond):
				}
			}
		})
	}
}

func TestBus_UnsubscribeStopsDelivery(t *testing.T) {
	bus := api.NewBus()
	defer bus.Close()

	ch, cancel := bus.Subscribe(core.MustParseURI("xdb://ns"))
	cancel()

	bus.Publish(watchEvent("xdb://ns/posts/p1"))

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel must be closed after cancel")
	case <-time.After(50 * time.Millisecond):
		t.Fatal("channel should be closed")
	}
}

func TestBus_FullBufferNeverBlocksPublisher(t *testing.T) {
	bus := api.NewBus()
	defer bus.Close()

	_, cancel := bus.Subscribe(core.MustParseURI("xdb://ns"))
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 1000 {
			bus.Publish(watchEvent("xdb://ns/posts/p1"))
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publisher blocked on a slow subscriber")
	}
}

func TestBus_CloseClosesSubscribers(t *testing.T) {
	bus := api.NewBus()

	ch, cancel := bus.Subscribe(core.MustParseURI("xdb://ns"))
	defer cancel()

	bus.Close()

	_, ok := <-ch
	require.False(t, ok)

	// Publish after close must not panic.
	bus.Publish(watchEvent("xdb://ns/posts/p1"))
}

func TestServicesPublishEvents(t *testing.T) {
	ctx := context.Background()
	bus := api.NewBus()
	defer bus.Close()

	s := store.New(xdbmemory.NewDriver())
	records := api.NewRecordService(s, api.WithEvents(bus))
	schemas := api.NewSchemaService(s, api.WithEvents(bus))

	ch, cancel := bus.Subscribe(core.MustParseURI("xdb://ev.ns"))
	defer cancel()

	_, err := schemas.Create(ctx, &api.CreateSchemaRequest{
		URI:  "xdb://ev.ns/posts",
		Data: json.RawMessage(`{"fields":{"title":{"type":"string"}}}`),
	})
	require.NoError(t, err)
	e := recvOne(t, ch)
	assert.Equal(t, "schema.create", e.Type)
	assert.Equal(t, "xdb://ev.ns/posts", e.URI)
	assert.False(t, e.TS.IsZero())

	_, err = records.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://ev.ns/posts/p1",
		Data: json.RawMessage(`{"title":"Hello"}`),
	})
	require.NoError(t, err)
	e = recvOne(t, ch)
	assert.Equal(t, "record.create", e.Type)
	assert.Contains(t, string(e.Data), "Hello")

	_, err = records.Update(ctx, &api.UpdateRecordRequest{
		URI:  "xdb://ev.ns/posts/p1",
		Data: json.RawMessage(`{"title":"Changed"}`),
	})
	require.NoError(t, err)
	assert.Equal(t, "record.update", recvOne(t, ch).Type)

	_, err = records.Delete(ctx, &api.DeleteRecordRequest{URI: "xdb://ev.ns/posts/p1"})
	require.NoError(t, err)
	assert.Equal(t, "record.delete", recvOne(t, ch).Type)

	// Idempotent delete of a missing record must not publish.
	_, err = records.Delete(ctx, &api.DeleteRecordRequest{URI: "xdb://ev.ns/posts/p1"})
	require.NoError(t, err)
	select {
	case e := <-ch:
		t.Fatalf("noop delete published event: %+v", e)
	case <-time.After(50 * time.Millisecond):
	}

	// Dry-run must not publish.
	_, err = records.Create(ctx, &api.CreateRecordRequest{
		URI:    "xdb://ev.ns/posts/p2",
		Data:   json.RawMessage(`{"title":"Dry"}`),
		DryRun: true,
	})
	require.NoError(t, err)
	select {
	case e := <-ch:
		t.Fatalf("dry-run published event: %+v", e)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestWatchService_ReadyFirstAndFiltered(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	bus := api.NewBus()
	defer bus.Close()

	svc := api.NewWatchService(bus)

	type frame struct {
		event string
		data  string
	}
	frames := make(chan frame, 16)
	send := func(event string, data json.RawMessage) {
		frames <- frame{event: event, data: string(data)}
	}

	done := make(chan error, 1)
	go func() {
		done <- svc.Watch(ctx, &api.WatchRequest{URI: "xdb://w.ns/posts"}, send)
	}()

	// First frame must be ready, before any event.
	select {
	case f := <-frames:
		require.Equal(t, "ready", f.event)
		assert.Contains(t, f.data, "xdb://w.ns/posts")
	case <-time.After(time.Second):
		t.Fatal("no ready frame")
	}

	bus.Publish(api.WatchEvent{Type: "record.create", URI: "xdb://w.ns/posts/p1"})
	bus.Publish(api.WatchEvent{Type: "record.create", URI: "xdb://w.ns/other/o1"})

	select {
	case f := <-frames:
		assert.Equal(t, "event", f.event)
		assert.Contains(t, f.data, "xdb://w.ns/posts/p1")
	case <-time.After(time.Second):
		t.Fatal("no event frame")
	}

	select {
	case f := <-frames:
		t.Fatalf("unmatched event delivered: %+v", f)
	case <-time.After(50 * time.Millisecond):
	}

	cancelCtx()
	select {
	case err := <-done:
		require.NoError(t, err, "context cancel must end the stream cleanly")
	case <-time.After(time.Second):
		t.Fatal("watch did not end on cancel")
	}
}

func TestWatchService_BusCloseEndsStream(t *testing.T) {
	bus := api.NewBus()
	svc := api.NewWatchService(bus)

	done := make(chan error, 1)
	go func() {
		done <- svc.Watch(context.Background(), &api.WatchRequest{URI: "xdb://w.ns"}, func(string, json.RawMessage) {})
	}()

	time.Sleep(50 * time.Millisecond)
	bus.Close()

	select {
	case err := <-done:
		require.NoError(t, err, "daemon stop (bus close) must end the stream cleanly")
	case <-time.After(time.Second):
		t.Fatal("watch did not end on bus close")
	}
}
