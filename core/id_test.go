package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	id := NewID("abc123")
	assert.Equal(t, "abc123", id.String())
}

func TestNewIDPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewID("")
	})
}

func TestParseID(t *testing.T) {
	id, err := ParseID("abc123")
	require.NoError(t, err)
	assert.Equal(t, "abc123", id.String())
}

func TestParseIDError(t *testing.T) {
	_, err := ParseID("")
	assert.ErrorIs(t, err, ErrInvalidID)
}

func TestIDStringNil(t *testing.T) {
	var nilID *ID
	assert.Equal(t, "", nilID.String())
}

func TestIDEquals(t *testing.T) {
	a := NewID("abc")
	b := NewID("abc")
	c := NewID("def")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestIDEqualsNil(t *testing.T) {
	var a *ID
	var b *ID
	c := NewID("abc")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
	assert.False(t, c.Equals(a))
}
