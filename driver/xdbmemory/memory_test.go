package xdbmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

func TestMemoryDriver_Tuples(t *testing.T) {
	driver := New()
	ctx := context.Background()

	tuples := []*types.Tuple{
		types.NewTuple("User", "1", "name", "Alice"),
		types.NewTuple("User", "2", "name", "Bob"),
	}

	t.Run("PutTuples", func(t *testing.T) {
		err := driver.PutTuples(ctx, tuples)
		assert.NoError(t, err)
	})

	t.Run("GetTuples", func(t *testing.T) {
		tuples, missed, err := driver.GetTuples(ctx, []*types.Key{
			types.NewKey("User", "1", "name"),
			types.NewKey("User", "2", "name"),
		})
		assert.NoError(t, err)
		assert.Empty(t, missed)
		assert.Equal(t, tuples[0].Value().Unwrap(), "Alice")
		assert.Equal(t, tuples[1].Value().Unwrap(), "Bob")
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		err := driver.DeleteTuples(ctx, []*types.Key{
			types.NewKey("User", "1", "name"),
		})
		assert.NoError(t, err)
	})

	t.Run("GetTuples after deletion", func(t *testing.T) {
		tuples, missed, err := driver.GetTuples(ctx, []*types.Key{
			types.NewKey("User", "1", "name"),
			types.NewKey("User", "2", "name"),
		})
		assert.NoError(t, err)
		assert.Equal(t, len(tuples), 1)
		assert.Equal(t, len(missed), 1)
		assert.Equal(t, tuples[0].Value().Unwrap(), "Bob")
		assert.Equal(t, missed[0].String(), "Key(User/1/name)")
	})
}

func TestMemoryDriver_Records(t *testing.T) {
	driver := New()
	ctx := context.Background()

	records := []*types.Record{
		types.NewRecord("User", "1").
			Set("name", "Alice"),
		types.NewRecord("User", "2").
			Set("name", "Bob"),
	}

	t.Run("PutRecords", func(t *testing.T) {
		err := driver.PutRecords(ctx, records)
		assert.NoError(t, err)
	})

	t.Run("GetRecords", func(t *testing.T) {
		records, missed, err := driver.GetRecords(ctx, []*types.Key{
			types.NewKey("User", "1"),
			types.NewKey("User", "2"),
		})
		assert.NoError(t, err)
		assert.Empty(t, missed)
		assert.Equal(t, records[0].Key().String(), "Key(User/1)")
		assert.Equal(t, records[1].Key().String(), "Key(User/2)")
	})

	t.Run("DeleteRecords", func(t *testing.T) {
		err := driver.DeleteRecords(ctx, []*types.Key{
			types.NewKey("User", "1"),
		})
		assert.NoError(t, err)
	})

	t.Run("GetRecords after deletion", func(t *testing.T) {
		records, missed, err := driver.GetRecords(ctx, []*types.Key{
			types.NewKey("User", "1"),
			types.NewKey("User", "2"),
		})
		assert.NoError(t, err)
		assert.Equal(t, len(records), 1)
		assert.Equal(t, len(missed), 1)
		assert.Equal(t, records[0].Key().String(), "Key(User/2)")
		assert.Equal(t, missed[0].String(), "Key(User/1)")
	})
}
