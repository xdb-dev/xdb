package xdbjson

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func TestImport_Product(t *testing.T) {
	def, err := ImportSchema(readFixture(t, "product.schema.json"), WithNS("com.acme"))
	require.NoError(t, err)

	assert.Equal(t, "com.acme", def.URI.NS())
	assert.Equal(t, "Product", def.URI.Schema())
	assert.Equal(t, "A product from Acme's catalog", def.Description)
	assert.Equal(t, schema.ModeFlexible, def.Mode) // no additionalProperties
	assert.Equal(t, "jsonschema", def.Annotations["source"])

	assert.Equal(t, core.TIDInteger, def.Fields["productId"].Type.ID())
	assert.True(t, def.Fields["productId"].Required)
	assert.Equal(t, core.TIDString, def.Fields["productName"].Type.ID())
	assert.True(t, def.Fields["productName"].Required)

	price := def.Fields["price"]
	assert.Equal(t, core.TIDFloat, price.Type.ID())
	assert.False(t, price.Required)
	assert.Equal(t, "0", price.Annotations["jsonschema.exclusiveMinimum"])

	tags := def.Fields["tags"]
	assert.Equal(t, core.TIDArray, tags.Type.ID())
	assert.Equal(t, core.TIDString, tags.Type.ElemTypeID())

	// nested object flattens to dotted attrs. dimensions is optional (not in the
	// top-level required), so its required members are NOT enforced.
	length := def.Fields["dimensions.length"]
	assert.Equal(t, core.TIDFloat, length.Type.ID())
	assert.False(t, length.Required)

	// the nested object itself is not a field.
	_, ok := def.Fields["dimensions"]
	assert.False(t, ok)
}

func TestImport_RequiredNested(t *testing.T) {
	data := []byte(`{
		"title": "Order",
		"type": "object",
		"properties": {
			"ship": {
				"type": "object",
				"properties": {"city": {"type": "string"}, "zip": {"type": "string"}},
				"required": ["city"]
			}
		},
		"required": ["ship"]
	}`)

	def, err := ImportSchema(data, WithNS("com.acme"))
	require.NoError(t, err)

	// ship is required, so its required member propagates.
	assert.True(t, def.Fields["ship.city"].Required)
	assert.False(t, def.Fields["ship.zip"].Required)
}

func TestImport_Event(t *testing.T) {
	def, err := ImportSchema(readFixture(t, "event.schema.json"), WithNS("com.acme"))
	require.NoError(t, err)

	assert.Equal(t, "Event", def.URI.Schema())
	assert.Equal(t, schema.ModeStrict, def.Mode) // additionalProperties: false

	assert.True(t, def.Fields["title"].Required)

	startsAt := def.Fields["startsAt"]
	assert.Equal(t, core.TIDTime, startsAt.Type.ID())
	assert.True(t, startsAt.Required)
	assert.Equal(t, "date-time", startsAt.Annotations["jsonschema.format"])

	priority := def.Fields["priority"]
	assert.Equal(t, core.TIDString, priority.Type.ID())
	assert.Contains(t, priority.Annotations["jsonschema.enum"], "high")

	// $ref to an object flattens.
	assert.Equal(t, core.TIDString, def.Fields["organizer.name"].Type.ID())
	assert.Equal(t, core.TIDString, def.Fields["organizer.email"].Type.ID())

	// array of $ref objects becomes an object array with Items.
	attendees := def.Fields["attendees"]
	assert.Equal(t, core.TIDArray, attendees.Type.ID())
	assert.Equal(t, core.TIDJSON, attendees.Type.ElemTypeID())
	require.NotNil(t, attendees.Items)
	assert.Equal(t, core.TIDString, attendees.Items["name"].Type.ID())
	assert.True(t, attendees.Items["name"].Required)
	assert.Equal(t, core.TIDTime, attendees.Items["rsvpAt"].Type.ID())
	assert.Equal(t, core.TIDInteger, attendees.Items["guests"].Type.ID())
}

func TestImport_SchemaNameFromID(t *testing.T) {
	data := []byte(`{
		"$id": "https://example.com/schemas/widget.schema.json",
		"type": "object",
		"properties": {"name": {"type": "string"}}
	}`)

	def, err := ImportSchema(data, WithNS("com.acme"))
	require.NoError(t, err)
	assert.Equal(t, "widget", def.URI.Schema())
}

func TestImport_TypedAdditionalPropertiesIsJSON(t *testing.T) {
	data := []byte(`{
		"title": "Config",
		"type": "object",
		"properties": {
			"labels": {"type": "object", "additionalProperties": {"type": "string"}}
		}
	}`)

	def, err := ImportSchema(data, WithNS("com.acme"))
	require.NoError(t, err)

	labels := def.Fields["labels"]
	assert.Equal(t, core.TIDJSON, labels.Type.ID())
	assert.Equal(t, "schema", labels.Annotations["jsonschema.additionalProperties"])
}

func TestImport_AllOfMerge(t *testing.T) {
	data := []byte(`{
		"title": "Merged",
		"type": "object",
		"allOf": [
			{"type": "object", "properties": {"a": {"type": "string"}}, "required": ["a"]},
			{"type": "object", "properties": {"b": {"type": "integer"}}}
		],
		"properties": {"c": {"type": "boolean"}}
	}`)

	def, err := ImportSchema(data, WithNS("com.acme"))
	require.NoError(t, err)

	assert.Equal(t, core.TIDString, def.Fields["a"].Type.ID())
	assert.True(t, def.Fields["a"].Required)
	assert.Equal(t, core.TIDInteger, def.Fields["b"].Type.ID())
	assert.Equal(t, core.TIDBoolean, def.Fields["c"].Type.ID())
}

func TestImport_NullableType(t *testing.T) {
	data := []byte(`{
		"title": "Nullable",
		"type": "object",
		"properties": {"nick": {"type": ["string", "null"]}}
	}`)

	def, err := ImportSchema(data, WithNS("com.acme"))
	require.NoError(t, err)
	assert.Equal(t, core.TIDString, def.Fields["nick"].Type.ID())
}

func TestImport_CyclicRefWithJSONOptIn(t *testing.T) {
	data := readFixture(t, "cyclic.schema.json")

	def, err := ImportSchema(data, WithNS("com.acme"), WithOpaqueJSON("#/$defs/node"))
	require.NoError(t, err)

	children := def.Fields["children"]
	assert.Equal(t, core.TIDArray, children.Type.ID())
	assert.Equal(t, core.TIDJSON, children.Type.ElemTypeID())
	assert.Empty(t, children.Items) // opaque, not walked
}

func fieldTag(t *testing.T, err error, key string) string {
	t.Helper()
	var e interface{ All() map[string]string }
	require.ErrorAs(t, err, &e)
	return e.All()[key]
}
