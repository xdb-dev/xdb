package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchema(t *testing.T) {
	s := NewSchema("posts")
	assert.Equal(t, "posts", s.String())
}

func TestNewSchemaPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewSchema("")
	})
}

func TestParseSchema(t *testing.T) {
	s, err := ParseSchema("posts")
	require.NoError(t, err)
	assert.Equal(t, "posts", s.String())
}

func TestParseSchemaError(t *testing.T) {
	_, err := ParseSchema("")
	assert.ErrorIs(t, err, ErrInvalidSchema)
}

func TestSchemaEquals(t *testing.T) {
	a := NewSchema("posts")
	b := NewSchema("posts")
	c := NewSchema("users")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}
