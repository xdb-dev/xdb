package xdbmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xdb-dev/xdb/types"
)

func TestMemoryDriver(t *testing.T) {
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
