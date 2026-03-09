package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBuilder() *Builder {
	return New().NS("com.example").Schema("posts").ID("123")
}

func TestBuilderBool(t *testing.T) {
	tuple, err := newTestBuilder().Bool("enabled", true)
	require.NoError(t, err)
	assert.Equal(t, "enabled", tuple.Attr().String())
	assert.Equal(t, TIDBoolean, tuple.Value().Type().ID())
	assert.Equal(t, true, tuple.Value().Unwrap())
}

func TestBuilderInt(t *testing.T) {
	tuple, err := newTestBuilder().Int("count", 42)
	require.NoError(t, err)
	assert.Equal(t, TIDInteger, tuple.Value().Type().ID())
	assert.Equal(t, int64(42), tuple.Value().Unwrap())
}

func TestBuilderUint(t *testing.T) {
	tuple, err := newTestBuilder().Uint("size", 100)
	require.NoError(t, err)
	assert.Equal(t, TIDUnsigned, tuple.Value().Type().ID())
	assert.Equal(t, uint64(100), tuple.Value().Unwrap())
}

func TestBuilderFloat(t *testing.T) {
	tuple, err := newTestBuilder().Float("score", 3.14)
	require.NoError(t, err)
	assert.Equal(t, TIDFloat, tuple.Value().Type().ID())
	assert.InDelta(t, 3.14, tuple.Value().Unwrap(), 0.001)
}

func TestBuilderStr(t *testing.T) {
	tuple, err := newTestBuilder().Str("title", "Hello")
	require.NoError(t, err)
	assert.Equal(t, TIDString, tuple.Value().Type().ID())
	assert.Equal(t, "Hello", tuple.Value().Unwrap())
}

func TestBuilderBytes(t *testing.T) {
	tuple, err := newTestBuilder().Bytes("data", []byte("raw"))
	require.NoError(t, err)
	assert.Equal(t, TIDBytes, tuple.Value().Type().ID())
	assert.Equal(t, []byte("raw"), tuple.Value().Unwrap())
}

func TestBuilderTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	tuple, err := newTestBuilder().Time("created_at", now)
	require.NoError(t, err)
	assert.Equal(t, TIDTime, tuple.Value().Type().ID())
	assert.Equal(t, now, tuple.Value().Unwrap())
}

func TestBuilderJSON(t *testing.T) {
	raw := json.RawMessage(`{"nested":"data"}`)
	tuple, err := newTestBuilder().JSON("metadata", raw)
	require.NoError(t, err)
	assert.Equal(t, TIDJSON, tuple.Value().Type().ID())
	assert.Equal(t, raw, tuple.Value().Unwrap())
}

func TestBuilderInvalidAttr(t *testing.T) {
	_, err := newTestBuilder().Bool("", true)
	assert.Error(t, err)
}

func TestBuilderInvalidNS(t *testing.T) {
	_, err := New().NS("!!!").Bool("enabled", true)
	assert.Error(t, err)
}

func TestBuilderMustBool(t *testing.T) {
	tuple := newTestBuilder().MustBool("enabled", true)
	assert.Equal(t, TIDBoolean, tuple.Value().Type().ID())
}

func TestBuilderMustBoolPanicsOnInvalidNS(t *testing.T) {
	assert.Panics(t, func() {
		New().NS("!!!").MustBool("enabled", true)
	})
}

func TestBuilderMustInt(t *testing.T) {
	tuple := newTestBuilder().MustInt("count", 42)
	assert.Equal(t, TIDInteger, tuple.Value().Type().ID())
}

func TestBuilderMustUint(t *testing.T) {
	tuple := newTestBuilder().MustUint("size", 100)
	assert.Equal(t, TIDUnsigned, tuple.Value().Type().ID())
}

func TestBuilderMustFloat(t *testing.T) {
	tuple := newTestBuilder().MustFloat("score", 3.14)
	assert.Equal(t, TIDFloat, tuple.Value().Type().ID())
}

func TestBuilderMustStr(t *testing.T) {
	tuple := newTestBuilder().MustStr("title", "Hello")
	assert.Equal(t, TIDString, tuple.Value().Type().ID())
}

func TestBuilderMustBytes(t *testing.T) {
	tuple := newTestBuilder().MustBytes("data", []byte("raw"))
	assert.Equal(t, TIDBytes, tuple.Value().Type().ID())
}

func TestBuilderMustTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	tuple := newTestBuilder().MustTime("created_at", now)
	assert.Equal(t, TIDTime, tuple.Value().Type().ID())
}

func TestBuilderMustJSON(t *testing.T) {
	raw := json.RawMessage(`{"nested":"data"}`)
	tuple := newTestBuilder().MustJSON("metadata", raw)
	assert.Equal(t, TIDJSON, tuple.Value().Type().ID())
}

func TestBuilderAttr(t *testing.T) {
	b := New().NS("com.example").Schema("posts").ID("123").Attr("title")
	uri, err := b.URI()
	require.NoError(t, err)
	assert.Equal(t, "title", uri.Attr().String())
}

func TestBuilderURI(t *testing.T) {
	uri, err := New().NS("com.example").Schema("posts").ID("123").URI()
	require.NoError(t, err)

	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Equal(t, "123", uri.ID().String())
}

func TestBuilderURIErrors(t *testing.T) {
	tests := []struct {
		name string
		b    *Builder
	}{
		{"invalid ns", New().NS("!!!").Schema("posts").ID("123")},
		{"invalid schema", New().NS("com.example").Schema("!!!").ID("123")},
		{"invalid id", New().NS("com.example").Schema("posts").ID("!!!")},
		{"invalid attr", New().NS("com.example").Schema("posts").ID("123").Attr("!!!")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.b.URI()
			assert.Error(t, err)
		})
	}
}

func TestBuilderMustURI(t *testing.T) {
	uri := New().NS("com.example").Schema("posts").ID("123").MustURI()
	assert.Equal(t, "xdb://com.example/posts/123", uri.String())
}

func TestBuilderMustURIPanics(t *testing.T) {
	assert.Panics(t, func() {
		New().NS("!!!").MustURI()
	})
}

func TestBuilderRecord(t *testing.T) {
	r, err := New().NS("com.example").Schema("posts").ID("123").Record()
	require.NoError(t, err)

	assert.Equal(t, "com.example", r.NS().String())
	assert.Equal(t, "posts", r.Schema().String())
	assert.Equal(t, "123", r.ID().String())
	assert.True(t, r.IsEmpty())
}

func TestBuilderRecordError(t *testing.T) {
	_, err := New().NS("!!!").Record()
	assert.Error(t, err)
}

func TestBuilderMustRecord(t *testing.T) {
	r := New().NS("com.example").Schema("posts").ID("123").MustRecord()
	assert.Equal(t, "com.example", r.NS().String())
}

func TestBuilderMustRecordPanics(t *testing.T) {
	assert.Panics(t, func() {
		New().NS("!!!").MustRecord()
	})
}

func TestBuilderTuple(t *testing.T) {
	tuple, err := New().NS("com.example").Schema("posts").ID("123").Tuple("title", "Hello")
	require.NoError(t, err)

	assert.Equal(t, "title", tuple.Attr().String())
	assert.Equal(t, "Hello", tuple.Value().Unwrap())
}

func TestBuilderTupleErrors(t *testing.T) {
	_, err := New().NS("!!!").Tuple("title", "Hello")
	assert.Error(t, err)

	_, err = New().NS("com.example").Tuple("", "Hello")
	assert.Error(t, err)
}

func TestBuilderMustTuple(t *testing.T) {
	tuple := newTestBuilder().MustTuple("title", "Hello")
	assert.Equal(t, "title", tuple.Attr().String())
}

func TestBuilderMustTuplePanics(t *testing.T) {
	assert.Panics(t, func() {
		New().NS("!!!").MustTuple("title", "Hello")
	})
}
