package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestNewID(t *testing.T) {
	t.Parallel()

	t.Run("Valid ID", func(t *testing.T) {
		assert.NotPanics(t, func() {
			id := core.NewID("123")
			assert.Equal(t, "123", id.String())
		})
	})

	t.Run("Invalid ID", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewID("invalid id")
		})
	})

	t.Run("Equals", func(t *testing.T) {
		assert.True(t, core.NewID("123").Equals(core.NewID("123")))
		assert.False(t, core.NewID("123").Equals(core.NewID("456")))
	})
}

func TestParseID(t *testing.T) {
	t.Parallel()

	t.Run("Valid ID", func(t *testing.T) {
		id, err := core.ParseID("123")
		assert.NoError(t, err)
		assert.Equal(t, "123", id.String())
	})

	t.Run("Hierarchy ID", func(t *testing.T) {
		id, err := core.ParseID("123/456")
		assert.NoError(t, err)
		assert.Equal(t, "123/456", id.String())
	})

	t.Run("Invalid ID", func(t *testing.T) {
		id, err := core.ParseID("invalid id")
		assert.Error(t, err)
		assert.Nil(t, id)
	})
}
