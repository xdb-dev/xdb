package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/x"
)

func TestTupleAPI(t *testing.T) {
	ctx := context.Background()
	driver := xdbmemory.New()

	store := api.NewTupleAPI(driver)

	putEndpoint := store.PutTuples()
	getEndpoint := store.GetTuples()
	deleteEndpoint := store.DeleteTuples()

	tuples := tests.FakeTuples()
	keys := x.Keys(tuples...)

	t.Run("PutTuples", func(t *testing.T) {
		req := &api.PutTuplesRequest{}
		for _, tuple := range tuples {
			*req = append(*req, &api.Tuple{
				ID:    tuple.ID(),
				Attr:  tuple.Attr(),
				Value: tuple.Value().Unwrap(),
			})
		}

		res, err := putEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("GetTuples", func(t *testing.T) {
		req := &api.GetTuplesRequest{}
		for _, key := range keys {
			*req = append(*req, &api.Key{
				ID:   key.ID(),
				Attr: key.Attr(),
			})
		}

		res, err := getEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tuples, len(tuples))
		require.Len(t, res.Missing, 0)
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		req := &api.DeleteTuplesRequest{}
		for _, key := range keys {
			*req = append(*req, &api.Key{
				ID:   key.ID(),
				Attr: key.Attr(),
			})
		}

		res, err := deleteEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("VerifyDeleted", func(t *testing.T) {
		req := &api.GetTuplesRequest{}
		for _, key := range keys {
			*req = append(*req, &api.Key{
				ID:   key.ID(),
				Attr: key.Attr(),
			})
		}

		res, err := getEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tuples, 0)
		require.Len(t, res.Missing, len(keys))
	})
}
