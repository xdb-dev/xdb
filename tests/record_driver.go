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
	keys := x.Keys(records...)

	t.Run("PutRecords", func(t *testing.T) {
		err := rw.PutRecords(ctx, records)
		require.NoError(t, err)
	})

	t.Run("GetRecords", func(t *testing.T) {
		got, missing, err := rw.GetRecords(ctx, keys)
		require.NoError(t, err)
		require.Len(t, missing, 0)
		AssertEqualRecords(t, records, got)
	})

	t.Run("GetRecordsSomeMissing", func(t *testing.T) {
		notFound := []*core.Key{
			core.NewKey("Post", "1", "not_found"),
			core.NewKey("Post", "2", "not_found"),
		}

		got, missing, err := rw.GetRecords(ctx, append(keys, notFound...))
		require.NoError(t, err)
		AssertEqualKeys(t, notFound, missing)
		AssertEqualRecords(t, records, got)
	})

	t.Run("DeleteRecords", func(t *testing.T) {
		err := rw.DeleteRecords(ctx, keys)
		require.NoError(t, err)
	})

	t.Run("GetRecordsAllMissing", func(t *testing.T) {
		got, missing, err := rw.GetRecords(ctx, keys)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Len(t, got, 0)
		AssertEqualKeys(t, missing, keys)
	})
}
