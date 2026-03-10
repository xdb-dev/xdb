package core

import (
	"encoding/json"
	"testing"
	"time"

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

func TestTupleAs(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	rawJSON := json.RawMessage(`{"key":"val"}`)

	tests := []struct {
		name  string
		attr  string
		input any
		check func(t *testing.T, tuple *Tuple)
	}{
		{
			name:  "Str",
			attr:  "title",
			input: "Hello",
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsStr()
				require.NoError(t, err)
				assert.Equal(t, "Hello", v)
			},
		},
		{
			name:  "Int",
			attr:  "age",
			input: int64(42),
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsInt()
				require.NoError(t, err)
				assert.Equal(t, int64(42), v)
			},
		},
		{
			name:  "Uint",
			attr:  "count",
			input: uint64(99),
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsUint()
				require.NoError(t, err)
				assert.Equal(t, uint64(99), v)
			},
		},
		{
			name:  "Float",
			attr:  "score",
			input: 3.14,
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsFloat()
				require.NoError(t, err)
				assert.InDelta(t, 3.14, v, 0.001)
			},
		},
		{
			name:  "Bool",
			attr:  "active",
			input: true,
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsBool()
				require.NoError(t, err)
				assert.True(t, v)
			},
		},
		{
			name:  "Bytes",
			attr:  "data",
			input: []byte("raw"),
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsBytes()
				require.NoError(t, err)
				assert.Equal(t, []byte("raw"), v)
			},
		},
		{
			name:  "Time",
			attr:  "created",
			input: now,
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsTime()
				require.NoError(t, err)
				assert.Equal(t, now, v)
			},
		},
		{
			name:  "JSON",
			attr:  "meta",
			input: rawJSON,
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsJSON()
				require.NoError(t, err)
				assert.JSONEq(t, `{"key":"val"}`, string(v))
			},
		},
		{
			name:  "Array",
			attr:  "tags",
			input: []string{"go", "xdb"},
			check: func(t *testing.T, tuple *Tuple) {
				v, err := tuple.AsArray()
				require.NoError(t, err)
				require.Len(t, v, 2)

				s0, err := v[0].AsStr()
				require.NoError(t, err)
				assert.Equal(t, "go", s0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := NewTuple("com.example/posts/1", tt.attr, tt.input)
			tt.check(t, tuple)
		})
	}

	t.Run("TypeMismatch", func(t *testing.T) {
		tuple := NewTuple("com.example/posts/1", "title", "Hello")
		_, err := tuple.AsInt()
		require.ErrorIs(t, err, ErrTypeMismatch)
	})

	t.Run("NilReceiver", func(t *testing.T) {
		var tuple *Tuple
		v, err := tuple.AsStr()
		require.NoError(t, err)
		assert.Empty(t, v)
	})
}
