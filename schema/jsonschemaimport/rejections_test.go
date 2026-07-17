package jsonschemaimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReject_DottedAndInvalidKeys(t *testing.T) {
	data := []byte(`{
		"title": "Bad",
		"type": "object",
		"properties": {
			"first.name": {"type": "string"},
			"has space": {"type": "string"},
			"ok": {"type": "string"}
		}
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKey)

	keys := fieldTag(t, err, "keys")
	assert.Contains(t, keys, "first.name")
	assert.Contains(t, keys, "has space")
	assert.NotContains(t, keys, "ok")
}

func TestReject_AnyOf(t *testing.T) {
	data := []byte(`{
		"title": "U",
		"type": "object",
		"properties": {
			"x": {"anyOf": [{"type": "string"}, {"type": "integer"}]}
		}
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnion)
	assert.Equal(t, "#/properties/x", fieldTag(t, err, "pointer"))
	assert.Equal(t, "anyOf", fieldTag(t, err, "keyword"))
}

func TestReject_OneOf(t *testing.T) {
	data := []byte(`{
		"title": "U",
		"type": "object",
		"properties": {
			"x": {"oneOf": [{"type": "string"}, {"type": "integer"}]}
		}
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnion)
	assert.Equal(t, "oneOf", fieldTag(t, err, "keyword"))
}

func TestReject_CyclicRefWithoutOptIn(t *testing.T) {
	_, err := Import(readFixture(t, "cyclic.schema.json"), WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCyclicRef)
	assert.Equal(t, "#/$defs/node", fieldTag(t, err, "ref"))
}

func TestReject_CrossDocumentRef(t *testing.T) {
	data := []byte(`{
		"title": "X",
		"type": "object",
		"properties": {
			"other": {"$ref": "other.json#/foo"}
		}
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCrossDocument)
	assert.Equal(t, "other.json#/foo", fieldTag(t, err, "ref"))
	assert.Equal(t, "#/properties/other", fieldTag(t, err, "pointer"))
}

func TestReject_AllOfConflict(t *testing.T) {
	data := []byte(`{
		"title": "X",
		"type": "object",
		"allOf": [
			{"type": "object", "properties": {"a": {"type": "string"}}},
			{"type": "object", "properties": {"a": {"type": "integer"}}}
		]
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConflict)
	assert.Equal(t, "a", fieldTag(t, err, "field"))
}

func TestReject_UnresolvedRef(t *testing.T) {
	data := []byte(`{
		"title": "X",
		"type": "object",
		"properties": {
			"x": {"$ref": "#/$defs/missing"}
		}
	}`)

	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnresolvedRef)
}

func TestReject_NoNamespace(t *testing.T) {
	data := []byte(`{"title": "X", "type": "object", "properties": {"a": {"type": "string"}}}`)
	_, err := Import(data)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoNamespace)
}

func TestReject_RootNotObject(t *testing.T) {
	data := []byte(`{"title": "X", "type": "string"}`)
	_, err := Import(data, WithNamespace("com.acme"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupported)
}
