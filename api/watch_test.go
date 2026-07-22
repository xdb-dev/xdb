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

	err := svc.Watch(context.Background(), &api.WatchRequest{}, nil)

	assert.ErrorIs(t, err, core.ErrNotImplemented)
	assert.Contains(t, err.Error(), "watch")
}
