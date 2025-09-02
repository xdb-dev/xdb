package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

type tupleReaderWriter interface {
	driver.TupleReader
	driver.TupleWriter
}

func TestTupleReaderWriter(t *testing.T, rw tupleReaderWriter) {
	t.Helper()

	ctx := context.Background()
	tuples := FakeTuples()
	keys := x.Keys(tuples...)

	t.Run("PutTuples", func(t *testing.T) {
		err := rw.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})

	t.Run("GetTuples", func(t *testing.T) {
		got, missing, err := rw.GetTuples(ctx, keys)
		require.NoError(t, err)
		require.Len(t, missing, 0)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("GetTuplesSomeMissing", func(t *testing.T) {
		notFound := []*types.Key{
			types.NewKey("Test", "1", "not_found"),
			types.NewKey("Test", "2", "not_found"),
		}

		got, missing, err := rw.GetTuples(ctx, append(keys, notFound...))
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		AssertEqualKeys(t, missing, notFound)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		err := rw.DeleteTuples(ctx, keys)
		require.NoError(t, err)
	})

	t.Run("GetTuplesAllMissing", func(t *testing.T) {
		got, missing, err := rw.GetTuples(ctx, keys)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Len(t, got, 0)
		AssertEqualKeys(t, missing, keys)
	})
}
