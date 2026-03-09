package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidComponent(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"abc", true},
		{"ABC", true},
		{"123", true},
		{"a.b.c", true},
		{"a_b", true},
		{"a-b", true},
		{"a/b", true},
		{"", false},
		{"a b", false},
		{"a!b", false},
		{"a@b", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidComponent(tt.input))
		})
	}
}

func TestParseURI(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		ns     string
		schema string
		id     string
		attr   string
	}{
		{
			name: "namespace only",
			uri:  "xdb://com.example",
			ns:   "com.example",
		},
		{
			name:   "namespace and schema",
			uri:    "xdb://com.example/posts",
			ns:     "com.example",
			schema: "posts",
		},
		{
			name:   "namespace, schema, and ID",
			uri:    "xdb://com.example/posts/123",
			ns:     "com.example",
			schema: "posts",
			id:     "123",
		},
		{
			name:   "full URI with attribute",
			uri:    "xdb://com.example/posts/123#title",
			ns:     "com.example",
			schema: "posts",
			id:     "123",
			attr:   "title",
		},
		{
			name:   "ID with slashes",
			uri:    "xdb://com.example/posts/a/b/c",
			ns:     "com.example",
			schema: "posts",
			id:     "a/b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := ParseURI(tt.uri)
			require.NoError(t, err)

			assert.Equal(t, tt.ns, uri.NS().String())

			if tt.schema != "" {
				require.NotNil(t, uri.Schema())
				assert.Equal(t, tt.schema, uri.Schema().String())
			} else {
				assert.Nil(t, uri.Schema())
			}

			if tt.id != "" {
				require.NotNil(t, uri.ID())
				assert.Equal(t, tt.id, uri.ID().String())
			} else {
				assert.Nil(t, uri.ID())
			}

			if tt.attr != "" {
				require.NotNil(t, uri.Attr())
				assert.Equal(t, tt.attr, uri.Attr().String())
			} else {
				assert.Nil(t, uri.Attr())
			}
		})
	}
}

func TestParseURIErrors(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{name: "wrong scheme", uri: "http://com.example/posts"},
		{name: "empty host", uri: "xdb:///posts"},
		{name: "invalid namespace", uri: "xdb://!!!"},
		{name: "invalid schema", uri: "xdb://com.example/!!!"},
		{name: "empty string", uri: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseURI(tt.uri)
			assert.Error(t, err)
		})
	}
}

func TestParsePath(t *testing.T) {
	uri, err := ParsePath("com.example/posts/123#title")
	require.NoError(t, err)

	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Equal(t, "123", uri.ID().String())
	assert.Equal(t, "title", uri.Attr().String())
}

func TestMustParseURI(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123")
	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Equal(t, "123", uri.ID().String())
}

func TestMustParseURIPanics(t *testing.T) {
	assert.Panics(t, func() {
		MustParseURI("http://invalid")
	})
}

func TestURIString(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "namespace only",
			uri:      "xdb://com.example",
			expected: "xdb://com.example",
		},
		{
			name:     "with schema",
			uri:      "xdb://com.example/posts",
			expected: "xdb://com.example/posts",
		},
		{
			name:     "with ID",
			uri:      "xdb://com.example/posts/123",
			expected: "xdb://com.example/posts/123",
		},
		{
			name:     "with attribute",
			uri:      "xdb://com.example/posts/123#title",
			expected: "xdb://com.example/posts/123#title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := MustParseURI(tt.uri)
			assert.Equal(t, tt.expected, uri.String())
		})
	}
}

func TestURIPath(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123#title")
	assert.Equal(t, "com.example/posts/123#title", uri.Path())
}

func TestURIEquals(t *testing.T) {
	a := MustParseURI("xdb://com.example/posts/123#title")
	b := MustParseURI("xdb://com.example/posts/123#title")
	c := MustParseURI("xdb://com.example/posts/456")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestURIEqualsNilComponents(t *testing.T) {
	a := MustParseURI("xdb://com.example")
	b := MustParseURI("xdb://com.example")
	c := MustParseURI("xdb://com.example/posts")

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
}

func TestURISchemaURI(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123#title")
	schemaURI := uri.SchemaURI()

	assert.Equal(t, "com.example", schemaURI.NS().String())
	assert.Equal(t, "posts", schemaURI.Schema().String())
	assert.Nil(t, schemaURI.ID())
	assert.Nil(t, schemaURI.Attr())
}

func TestURIMarshalJSON(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123")

	data, err := json.Marshal(uri)
	require.NoError(t, err)
	assert.Equal(t, `"xdb://com.example/posts/123"`, string(data))
}

func TestURIUnmarshalJSON(t *testing.T) {
	var uri URI
	err := json.Unmarshal([]byte(`"xdb://com.example/posts/123"`), &uri)
	require.NoError(t, err)

	assert.Equal(t, "com.example", uri.NS().String())
	assert.Equal(t, "posts", uri.Schema().String())
	assert.Equal(t, "123", uri.ID().String())
}

func TestURIUnmarshalJSONErrors(t *testing.T) {
	var uri URI

	err := json.Unmarshal([]byte(`123`), &uri)
	assert.Error(t, err)

	err = json.Unmarshal([]byte(`"http://invalid"`), &uri)
	assert.Error(t, err)
}

func TestURIRoundtrip(t *testing.T) {
	original := MustParseURI("xdb://com.example/posts/123#title")

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded URI
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, original.Equals(&decoded))
}
