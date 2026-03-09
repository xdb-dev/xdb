package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAttr(t *testing.T) {
	a := NewAttr("title")
	assert.Equal(t, "title", a.String())
}

func TestNewAttrPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewAttr("")
	})
}

func TestParseAttr(t *testing.T) {
	a, err := ParseAttr("title")
	require.NoError(t, err)
	assert.Equal(t, "title", a.String())
}

func TestParseAttrError(t *testing.T) {
	_, err := ParseAttr("")
	assert.ErrorIs(t, err, ErrInvalidAttr)
}

func TestAttrStringNil(t *testing.T) {
	var nilAttr *Attr
	assert.Equal(t, "", nilAttr.String())
}

func TestAttrEquals(t *testing.T) {
	a := NewAttr("title")
	b := NewAttr("title")
	c := NewAttr("body")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestAttrEqualsNil(t *testing.T) {
	var a *Attr
	var b *Attr
	c := NewAttr("title")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
	assert.False(t, c.Equals(a))
}

