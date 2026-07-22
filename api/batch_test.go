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

func TestBatchService_Execute_NotImplemented(t *testing.T) {
	t.Parallel()

	s := store.New(xdbmemory.NewDriver())
	svc := api.NewBatchService(s)

	_, err := svc.Execute(context.Background(), &api.ExecuteBatchRequest{})

	assert.ErrorIs(t, err, core.ErrNotImplemented)
	assert.Contains(t, err.Error(), "batch.execute")
}
