package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestNewNS(t *testing.T) {
	t.Parallel()

	t.Run("Valid NS", func(t *testing.T) {
		assert.NotPanics(t, func() {
			ns := core.NewNS("com.example")
			assert.Equal(t, "com.example", ns.String())
		})
	})

	t.Run("Invalid NS", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewNS("invalid.example")
		})
	})

	t.Run("Equals", func(t *testing.T) {
		assert.True(t, core.NewNS("com.example").Equals(core.NewNS("com.example")))
		assert.False(t, core.NewNS("com.example").Equals(core.NewNS("com.example.other")))
	})
}

func TestParseNS(t *testing.T) {
	t.Parallel()

	t.Run("Valid NS", func(t *testing.T) {
		ns, err := core.ParseNS("com.example")
		assert.NoError(t, err)
		assert.Equal(t, "com.example", ns.String())
	})

	t.Run("Invalid NS", func(t *testing.T) {
		ns, err := core.ParseNS("invalid example")
		assert.Error(t, err)
		assert.Nil(t, ns)
	})
}
