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

	assert.Equal(t, "com.example", r.NS().String())
	assert.Equal(t, "posts", r.Schema().String())
	assert.Equal(t, "123", r.ID().String())
	assert.True(t, r.IsEmpty())
}

func TestRecordSetAndGet(t *testing.T) {
	r := newTestRecord()

	r.Set("title", "Hello World")
	r.Set("count", 42)

	title := r.Get("title")
	require.NotNil(t, title)
	assert.Equal(t, "Hello World", title.Value().Unwrap())

	count := r.Get("count")
	require.NotNil(t, count)
	assert.Equal(t, int64(42), count.Value().Unwrap())
}

func TestRecordGetMissing(t *testing.T) {
	r := newTestRecord()
	assert.Nil(t, r.Get("nonexistent"))
}

func TestRecordSetOverwrite(t *testing.T) {
	r := newTestRecord()

	r.Set("title", "First")
	r.Set("title", "Second")

	title := r.Get("title")
	require.NotNil(t, title)
	assert.Equal(t, "Second", title.Value().Unwrap())
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
	uri := r.SchemaURI()

	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Nil(t, uri.ID())
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

	assert.Equal(t, "com.example", tuple.NS().String())
	assert.Equal(t, "posts", tuple.Schema().String())
	assert.Equal(t, "123", tuple.ID().String())
	assert.Equal(t, "title", tuple.Attr().String())
}
