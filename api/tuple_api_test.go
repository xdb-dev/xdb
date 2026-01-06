package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/x"
)

func TestTupleAPI(t *testing.T) {
	ctx := context.Background()
	driver := xdbmemory.New()

	schemaURI := core.New().NS("com.example").Schema("all_types").MustURI()
	err := driver.PutSchema(ctx, schemaURI, tests.FakeAllTypesSchema())
	require.NoError(t, err)

	store := api.NewTupleAPI(driver)

	putEndpoint := store.PutTuples()
	getEndpoint := store.GetTuples()
	deleteEndpoint := store.DeleteTuples()

	tuples := tests.FakeTuples()
	uris := x.URIs(tuples...)

	t.Run("PutTuples", func(t *testing.T) {
		req := &api.PutTuplesRequest{}
		for _, tuple := range tuples {
			*req = append(*req, &api.Tuple{
				ID:    tuple.URI().Path(),
				Attr:  tuple.Attr().String(),
				Value: tuple.Value().Unwrap(),
			})
		}

		res, err := putEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("GetTuples", func(t *testing.T) {
		req := &api.GetTuplesRequest{}
		for _, uri := range uris {
			*req = append(*req, uri.String())
		}

		res, err := getEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tuples, len(tuples))
		require.Len(t, res.Missing, 0)
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		req := &api.DeleteTuplesRequest{}
		for _, uri := range uris {
			*req = append(*req, uri.String())
		}

		res, err := deleteEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("VerifyDeleted", func(t *testing.T) {
		req := &api.GetTuplesRequest{}
		for _, uri := range uris {
			*req = append(*req, uri.String())
		}

		res, err := getEndpoint(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Tuples, 0)
		require.Len(t, res.Missing, len(uris))
	})
}
