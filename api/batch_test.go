package api_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/rpc"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// noTxStore hides the TX capability of the wrapped store, simulating a
// non-transactional backend like fs or redis.
type noTxStore struct {
	store.Store
}

func batchFixture(t *testing.T) (store.Store, *api.BatchService) {
	t.Helper()

	s := store.New(xdbmemory.NewDriver())
	schemas := api.NewSchemaService(s)

	_, err := schemas.Create(context.Background(), &api.CreateSchemaRequest{
		URI:  "xdb://batch.t/items",
		Data: json.RawMessage(`{"fields":{"name":{"type":"string","required":true},"qty":{"type":"integer"}}}`),
	})
	require.NoError(t, err)

	return s, api.NewBatchService(s)
}

func op(kind, uri, data string) api.BatchOperation {
	o := api.BatchOperation{Op: kind, URI: uri}
	if data != "" {
		o.Data = json.RawMessage(data)
	}

	return o
}

func TestBatchService_Execute_AllSucceed(t *testing.T) {
	s, svc := batchFixture(t)
	ctx := context.Background()

	resp, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
		Operations: []api.BatchOperation{
			op("records.create", "xdb://batch.t/items/i1", `{"name":"a","qty":1}`),
			op("records.create", "xdb://batch.t/items/i2", `{"name":"b","qty":2}`),
			op("records.update", "xdb://batch.t/items/i1", `{"qty":10}`),
		},
	})
	require.NoError(t, err)

	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 3, resp.Succeeded)
	assert.Equal(t, 0, resp.Failed)
	assert.False(t, resp.RolledBack)
	require.Len(t, resp.Results, 3)

	for i, r := range resp.Results {
		assert.Equal(t, i, r.Index)
		assert.Equal(t, "ok", r.Status)
		assert.Nil(t, r.Error)
	}

	records := api.NewRecordService(s)
	got, err := records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/i1"})
	require.NoError(t, err)
	assert.Equal(t, float64(10), recordData(t, got.Data)["qty"])
}

func TestBatchService_Execute_MidBatchFailureRollsBack(t *testing.T) {
	s, svc := batchFixture(t)
	ctx := context.Background()

	records := api.NewRecordService(s)
	_, err := records.Create(ctx, &api.CreateRecordRequest{
		URI:  "xdb://batch.t/items/existing",
		Data: json.RawMessage(`{"name":"seed","qty":1}`),
	})
	require.NoError(t, err)

	resp, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
		Operations: []api.BatchOperation{
			op("records.create", "xdb://batch.t/items/new1", `{"name":"a","qty":1}`),
			op("records.create", "xdb://batch.t/items/existing", `{"name":"DIVERGENT","qty":9}`),
			op("records.create", "xdb://batch.t/items/new2", `{"name":"c","qty":3}`),
		},
	})
	require.NoError(t, err, "per-op failures are reported in results, not as a call error")

	assert.True(t, resp.RolledBack)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 0, resp.Succeeded)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Results, 3)

	assert.Equal(t, "ok", resp.Results[0].Status)
	assert.Equal(t, "error", resp.Results[1].Status)
	require.NotNil(t, resp.Results[1].Error)
	assert.Equal(t, rpc.CodeConflict, resp.Results[1].Error.Code)
	assert.Equal(t, "skipped", resp.Results[2].Status)

	_, err = records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/new1"})
	assert.ErrorIs(t, err, core.ErrNotFound, "op 1 must be rolled back")
}

func TestBatchService_Execute_UnknownOpRejectsUpfront(t *testing.T) {
	s, svc := batchFixture(t)
	ctx := context.Background()

	_, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
		Operations: []api.BatchOperation{
			op("records.create", "xdb://batch.t/items/i1", `{"name":"a","qty":1}`),
			op("records.frobnicate", "xdb://batch.t/items/i1", ""),
		},
	})
	require.Error(t, err)

	var rpcErr *rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	assert.Equal(t, rpc.CodeInvalidParams, rpcErr.Code)
	assert.Contains(t, rpcErr.Message, "records.frobnicate")

	records := api.NewRecordService(s)
	_, err = records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/i1"})
	assert.ErrorIs(t, err, core.ErrNotFound, "nothing may execute when validation fails upfront")
}

func TestBatchService_Execute_DryRun(t *testing.T) {
	s, svc := batchFixture(t)
	ctx := context.Background()

	resp, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
		Operations: []api.BatchOperation{
			op("records.create", "xdb://batch.t/items/i1", `{"name":"a","qty":1}`),
			op("records.create", "xdb://batch.t/items/bad", `{"qty":1}`),
		},
		DryRun: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Succeeded)
	assert.Equal(t, 1, resp.Failed)
	require.Len(t, resp.Results, 2)
	require.NotNil(t, resp.Results[0].DryRun)
	assert.Equal(t, "create", resp.Results[0].DryRun.Would)
	require.NotNil(t, resp.Results[1].Error)
	assert.Equal(t, rpc.CodeSchemaViolation, resp.Results[1].Error.Code)

	records := api.NewRecordService(s)
	_, err = records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/i1"})
	assert.ErrorIs(t, err, core.ErrNotFound, "dry-run batch must not write")
}

func TestBatchService_Execute_NonTxBackend(t *testing.T) {
	s, _ := batchFixture(t)
	ctx := context.Background()
	svc := api.NewBatchService(noTxStore{Store: s})

	t.Run("refuses without non_atomic", func(t *testing.T) {
		_, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
			Operations: []api.BatchOperation{
				op("records.create", "xdb://batch.t/items/i1", `{"name":"a","qty":1}`),
			},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrNotImplemented)
		assert.Contains(t, err.Error(), "non_atomic")
	})

	t.Run("non_atomic executes sequentially with per-op attribution", func(t *testing.T) {
		resp, err := svc.Execute(ctx, &api.ExecuteBatchRequest{
			Operations: []api.BatchOperation{
				op("records.create", "xdb://batch.t/items/s1", `{"name":"a","qty":1}`),
				op("records.create", "xdb://batch.t/items/s2", `{"qty":1}`),
				op("records.create", "xdb://batch.t/items/s3", `{"name":"c","qty":3}`),
			},
			NonAtomic: true,
		})
		require.NoError(t, err)

		assert.False(t, resp.RolledBack)
		assert.Equal(t, 2, resp.Succeeded)
		assert.Equal(t, 1, resp.Failed)
		assert.Equal(t, "ok", resp.Results[0].Status)
		assert.Equal(t, "error", resp.Results[1].Status)
		assert.Equal(t, "ok", resp.Results[2].Status)

		records := api.NewRecordService(s)
		_, err = records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/s1"})
		assert.NoError(t, err)
		_, err = records.Get(ctx, &api.GetRecordRequest{URI: "xdb://batch.t/items/s3"})
		assert.NoError(t, err)
	})
}
