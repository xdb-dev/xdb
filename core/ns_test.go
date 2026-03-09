package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNS(t *testing.T) {
	ns := NewNS("com.example")
	assert.Equal(t, "com.example", ns.String())
}

func TestNewNSPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewNS("")
	})
}

func TestParseNS(t *testing.T) {
	ns, err := ParseNS("com.example")
	require.NoError(t, err)
	assert.Equal(t, "com.example", ns.String())
}

func TestParseNSError(t *testing.T) {
	_, err := ParseNS("")
	assert.ErrorIs(t, err, ErrInvalidNS)

	_, err = ParseNS("!!!")
	assert.ErrorIs(t, err, ErrInvalidNS)
}

func TestNSEquals(t *testing.T) {
	a := NewNS("com.example")
	b := NewNS("com.example")
	c := NewNS("com.other")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestNSMarshalJSON(t *testing.T) {
	ns := NewNS("com.example")

	data, err := json.Marshal(ns)
	require.NoError(t, err)
	assert.Equal(t, `"com.example"`, string(data))
}

func TestNSUnmarshalJSON(t *testing.T) {
	var ns NS
	err := json.Unmarshal([]byte(`"com.example"`), &ns)
	require.NoError(t, err)
	assert.Equal(t, "com.example", ns.String())
}

func TestNSUnmarshalJSONErrors(t *testing.T) {
	var ns NS

	err := json.Unmarshal([]byte(`123`), &ns)
	assert.Error(t, err)

	err = json.Unmarshal([]byte(`"!!!"`), &ns)
	assert.Error(t, err)
}
