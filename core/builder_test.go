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

func TestBuilderInvalidAttrTyped(t *testing.T) {
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
