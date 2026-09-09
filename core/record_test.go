package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRecord() *Record {
	return NewRecord("com.example", "posts", "123")
}

func TestNewRecord(t *testing.T) {
	r := newTestRecord()

	assert.Equal(t, "com.example", r.URI().NS())
	assert.Equal(t, "posts", r.URI().Schema())
	assert.Equal(t, "123", r.URI().ID())
	assert.True(t, r.IsEmpty())
}

func TestRecordSetAndGet(t *testing.T) {
	r := newTestRecord()

	r.Set("title", "Hello World")
	r.Set("count", 42)

	title := r.Get("title")
	require.NotNil(t, title)
	assert.Equal(t, "Hello World", readValue(t, title.Value()))

	count := r.Get("count")
	require.NotNil(t, count)
	assert.Equal(t, int64(42), readValue(t, count.Value()))
}

func TestRecordGetMissing(t *testing.T) {
	r := newTestRecord()
	assert.Nil(t, r.Get("nonexistent"))
}

func TestRecordGetMissingAsReturnsAttrNotFound(t *testing.T) {
	r := newTestRecord()

	// Reading a missing attribute is distinguishable from an empty string:
	// it returns the zero value and ErrAttrNotFound.
	v, err := r.Get("tpyo").AsStr()
	require.ErrorIs(t, err, ErrAttrNotFound)
	assert.Empty(t, v)

	// ErrAttrNotFound must not be mistaken for a resource-not-found error.
	assert.NotErrorIs(t, err, ErrNotFound)
}

func TestRecordSetOverwrite(t *testing.T) {
	r := newTestRecord()

	r.Set("title", "First")
	r.Set("title", "Second")

	title := r.Get("title")
	require.NotNil(t, title)
	assert.Equal(t, "Second", readValue(t, title.Value()))
}

func TestRecordSetChaining(t *testing.T) {
	r := newTestRecord().
		Set("title", "Hello").
		Set("count", 42)

	assert.False(t, r.IsEmpty())
	assert.NotNil(t, r.Get("title"))
	assert.NotNil(t, r.Get("count"))
}

func TestRecordIsEmpty(t *testing.T) {
	r := newTestRecord()
	assert.True(t, r.IsEmpty())

	r.Set("title", "Hello")
	assert.False(t, r.IsEmpty())
}

func TestRecordTuples(t *testing.T) {
	r := newTestRecord().
		Set("title", "Hello").
		Set("count", 42)

	tuples := r.Tuples()
	assert.Len(t, tuples, 2)
}

func TestRecordURI(t *testing.T) {
	r := newTestRecord()
	uri := r.URI()

	assert.Equal(t, "xdb://com.example/posts/123", uri.String())
}

func TestRecordSchemaURI(t *testing.T) {
	r := newTestRecord()
	uri := r.URI().SchemaURI()

	assert.Equal(t, "com.example", uri.NS())
	assert.Equal(t, "posts", uri.Schema())
	assert.Empty(t, uri.ID())
}

func TestRecordGoString(t *testing.T) {
	r := newTestRecord()
	assert.Equal(t, "Record(xdb://com.example/posts/123)", r.GoString())
}

func TestRecordSetPanicsOnInvalidAttr(t *testing.T) {
	r := newTestRecord()
	assert.Panics(t, func() {
		r.Set("", "value")
	})
}

func TestRecordTupleAccessors(t *testing.T) {
	r := newTestRecord()
	r.Set("title", "Hello")

	tuple := r.Get("title")
	require.NotNil(t, tuple)

	assert.Equal(t, "com.example", tuple.Path().NS())
	assert.Equal(t, "posts", tuple.Path().Schema())
	assert.Equal(t, "123", tuple.Path().ID())
	assert.Equal(t, "title", tuple.Attr())
}
