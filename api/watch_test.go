package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
)

func TestWatchService_Watch_NoBusNotImplemented(t *testing.T) {
	t.Parallel()

	svc := api.NewWatchService(nil)

	err := svc.Watch(context.Background(), &api.WatchRequest{URI: "xdb://com.example"}, nil)

	assert.ErrorIs(t, err, core.ErrNotImplemented)
	assert.Contains(t, err.Error(), "watch")
}

func TestWatchService_Watch_InvalidURI(t *testing.T) {
	t.Parallel()

	svc := api.NewWatchService(api.NewBus())

	err := svc.Watch(context.Background(), &api.WatchRequest{URI: "bad"}, nil)

	assert.ErrorIs(t, err, core.ErrInvalidURI)
	assert.NotErrorIs(t, err, core.ErrNotImplemented)
}

func TestWatchService_Watch_AllDepthsSubscribe(t *testing.T) {
	t.Parallel()

	bus := api.NewBus()
	svc := api.NewWatchService(bus)

	for _, uri := range []string{
		"xdb://com.example",
		"xdb://com.example/posts",
		"xdb://com.example/posts/123",
		"xdb://com.example/posts/123#title",
	} {
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- svc.Watch(ctx, &api.WatchRequest{URI: uri}, func(string, json.RawMessage) {})
		}()

		cancel()
		assert.NoError(t, <-done, "uri %s", uri)
	}
}
