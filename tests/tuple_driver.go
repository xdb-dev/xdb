package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
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
	uris := x.URIs(tuples...)

	t.Run("PutTuples", func(t *testing.T) {
		err := rw.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})

	t.Run("GetTuples", func(t *testing.T) {
		got, missing, err := rw.GetTuples(ctx, uris)
		require.NoError(t, err)
		require.Empty(t, missing)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("GetTuplesSomeMissing", func(t *testing.T) {
		notFound := []*core.URI{
			core.New().NS("com.example").Schema("all_types").ID("not_found_1").MustURI(),
			core.New().NS("com.example").Schema("all_types").ID("not_found_2").MustURI(),
		}

		got, missing, err := rw.GetTuples(ctx, append(uris, notFound...))
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		AssertEqualURIs(t, missing, notFound)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		err := rw.DeleteTuples(ctx, uris)
		require.NoError(t, err)
	})

	t.Run("GetTuplesAllMissing", func(t *testing.T) {
		got, missing, err := rw.GetTuples(ctx, uris)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Empty(t, got)
		AssertEqualURIs(t, missing, uris)
	})
}
