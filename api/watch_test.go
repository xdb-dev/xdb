package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func TestWatchService_Watch_NotImplemented(t *testing.T) {
	t.Parallel()

	s := store.New(xdbmemory.NewDriver())
	svc := api.NewWatchService(s)

	err := svc.Watch(context.Background(), &api.WatchRequest{URI: "xdb://com.example"}, nil)

	assert.ErrorIs(t, err, core.ErrNotImplemented)
	assert.Contains(t, err.Error(), "watch")
}

func TestWatchService_Watch_InvalidURI(t *testing.T) {
	t.Parallel()

	s := store.New(xdbmemory.NewDriver())
	svc := api.NewWatchService(s)

	err := svc.Watch(context.Background(), &api.WatchRequest{URI: "bad"}, nil)

	assert.ErrorIs(t, err, core.ErrInvalidURI)
	assert.NotErrorIs(t, err, core.ErrNotImplemented)
}

func TestWatchService_Watch_AllDepthsParse(t *testing.T) {
	t.Parallel()

	s := store.New(xdbmemory.NewDriver())
	svc := api.NewWatchService(s)

	for _, uri := range []string{
		"xdb://com.example",
		"xdb://com.example/posts",
		"xdb://com.example/posts/123",
		"xdb://com.example/posts/123#title",
	} {
		err := svc.Watch(context.Background(), &api.WatchRequest{URI: uri}, nil)
		assert.ErrorIs(t, err, core.ErrNotImplemented, "uri %s", uri)
	}
}
