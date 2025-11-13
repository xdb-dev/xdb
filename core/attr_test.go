package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestNewAttr(t *testing.T) {
	t.Parallel()

	t.Run("Valid Attr", func(t *testing.T) {
		assert.NotPanics(t, func() {
			attr := core.NewAttr("name")
			assert.Equal(t, "name", attr.String())
		})
	})

	t.Run("Invalid Attr", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewAttr("invalid attr")
		})
	})

	t.Run("Equals", func(t *testing.T) {
		assert.True(t, core.NewAttr("name").Equals(core.NewAttr("name")))
		assert.False(t, core.NewAttr("name").Equals(core.NewAttr("other")))
	})
}

func TestParseAttr(t *testing.T) {
	t.Parallel()

	t.Run("Valid Attr", func(t *testing.T) {
		attr, err := core.ParseAttr("name")
		assert.NoError(t, err)
		assert.Equal(t, "name", attr.String())
	})

	t.Run("Nested Attr", func(t *testing.T) {
		attr, err := core.ParseAttr("profile.name")
		assert.NoError(t, err)
		assert.Equal(t, "profile.name", attr.String())
	})

	t.Run("Invalid Attr", func(t *testing.T) {
		attr, err := core.ParseAttr("invalid attr")
		assert.Error(t, err)
		assert.Nil(t, attr)
	})
}
