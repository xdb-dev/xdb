package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTuple() *Tuple {
	return NewTuple("com.example/posts/123", "title", "Hello")
}

func TestNewTuple(t *testing.T) {
	tuple := newTestTuple()

	assert.Equal(t, "com.example", tuple.NS().String())
	assert.Equal(t, "posts", tuple.Schema().String())
	assert.Equal(t, "123", tuple.ID().String())
	assert.Equal(t, "title", tuple.Attr().String())
	assert.Equal(t, "Hello", tuple.Value().Unwrap())
}

func TestNewTuplePanicsOnInvalidPath(t *testing.T) {
	assert.Panics(t, func() {
		NewTuple("!!!", "title", "Hello")
	})
}

func TestNewTuplePanicsOnInvalidAttr(t *testing.T) {
	assert.Panics(t, func() {
		NewTuple("com.example/posts/123", "", "Hello")
	})
}

func TestTuplePath(t *testing.T) {
	tuple := newTestTuple()
	path := tuple.Path()

	require.NotNil(t, path)
	assert.Equal(t, "com.example", path.NS().String())
	assert.Equal(t, "posts", path.Schema().String())
	assert.Equal(t, "123", path.ID().String())
}

func TestTupleSchemaURI(t *testing.T) {
	tuple := newTestTuple()
	schemaURI := tuple.SchemaURI()

	assert.Equal(t, "com.example", schemaURI.NS().String())
	assert.Equal(t, "posts", schemaURI.Schema().String())
	assert.Nil(t, schemaURI.ID())
}

func TestTupleURI(t *testing.T) {
	tuple := newTestTuple()
	uri := tuple.URI()

	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Equal(t, "123", uri.ID().String())
	assert.Equal(t, "title", uri.Attr().String())
}

func TestTupleGoString(t *testing.T) {
	tuple := newTestTuple()
	gs := tuple.GoString()

	assert.Contains(t, gs, "Tuple(")
	assert.Contains(t, gs, "xdb://com.example/posts/123")
	assert.Contains(t, gs, "title")
}
