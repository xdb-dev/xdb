package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

type recordReaderWriter interface {
	driver.RecordReader
	driver.RecordWriter
}

func TestRecordReaderWriter(t *testing.T, rw recordReaderWriter) {
	t.Helper()

	ctx := context.Background()
	records := FakePosts(10)
	uris := x.URIs(records...)

	t.Run("PutRecords", func(t *testing.T) {
		err := rw.PutRecords(ctx, records)
		require.NoError(t, err)
	})

	t.Run("GetRecords", func(t *testing.T) {
		got, missing, err := rw.GetRecords(ctx, uris)
		require.NoError(t, err)
		require.Len(t, missing, 0)
		AssertEqualRecords(t, records, got)
	})

	t.Run("GetRecordsSomeMissing", func(t *testing.T) {
		notFound := []*core.URI{
			core.New().NS("com.example").Schema("posts").ID("not_found_1").MustURI(),
			core.New().NS("com.example").Schema("posts").ID("not_found_2").MustURI(),
		}

		got, missing, err := rw.GetRecords(ctx, append(uris, notFound...))
		require.NoError(t, err)
		AssertEqualURIs(t, notFound, missing)
		AssertEqualRecords(t, records, got)
	})

	t.Run("DeleteRecords", func(t *testing.T) {
		err := rw.DeleteRecords(ctx, uris)
		require.NoError(t, err)
	})

	t.Run("GetRecordsAllMissing", func(t *testing.T) {
		got, missing, err := rw.GetRecords(ctx, uris)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Len(t, got, 0)
		AssertEqualURIs(t, missing, uris)
	})
}
