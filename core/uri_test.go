package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateComponent(t *testing.T) {
	tests := []struct {
		input      string
		allowSlash bool
		valid      bool
	}{
		{"abc", false, true},
		{"ABC", false, true},
		{"123", false, true},
		{"a.b.c", false, true},
		{"a_b", false, true},
		{"a-b", false, true},
		{"a/b", false, false}, // slash forbidden by default (NS/schema/attr)
		{"a/b", true, true},   // slash allowed for IDs
		{"", false, false},
		{"", true, false},
		{"a b", false, false},
		{"a!b", false, false},
		{"a@b", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateComponent("test", tt.input, tt.allowSlash)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrInvalidURI)
			}
		})
	}
}

func TestNewURI(t *testing.T) {
	tests := []struct {
		name     string
		ns       string
		parts    []string
		expected string
		wantErr  bool
	}{
		{
			name:     "namespace only",
			ns:       "com.example",
			expected: "xdb://com.example",
		},
		{
			name:     "namespace and schema",
			ns:       "com.example",
			parts:    []string{"posts"},
			expected: "xdb://com.example/posts",
		},
		{
			name:     "namespace, schema, and ID",
			ns:       "com.example",
			parts:    []string{"posts", "123"},
			expected: "xdb://com.example/posts/123",
		},
		{
			name:    "invalid namespace",
			ns:      "bad ns",
			wantErr: true,
		},
		{
			name:    "invalid schema",
			ns:      "com.example",
			parts:   []string{"bad schema"},
			wantErr: true,
		},
		{
			name:    "empty namespace",
			ns:      "",
			wantErr: true,
		},
		{
			name:    "too many parts",
			ns:      "com.example",
			parts:   []string{"posts", "123", "extra"},
			wantErr: true,
		},
		{
			name:    "empty schema part",
			ns:      "com.example",
			parts:   []string{""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := NewURI(tt.ns, tt.parts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, uri.String())
		})
	}
}

func TestURI_Depth(t *testing.T) {
	tests := []struct {
		name  string
		uri   URI
		depth int
	}{
		{name: "namespace only", uri: URI{ns: "com.example"}, depth: 1},
		{name: "namespace and schema", uri: URI{ns: "com.example", schema: "posts"}, depth: 2},
		{name: "namespace, schema, and id", uri: URI{ns: "com.example", schema: "posts", id: "123"}, depth: 3},
		{name: "namespace with attr stays depth 1", uri: URI{ns: "com.example", attr: "title"}, depth: 1},
		{name: "schema with attr stays depth 2", uri: URI{ns: "com.example", schema: "posts", attr: "title"}, depth: 2},
		{name: "record with attr stays depth 3", uri: URI{ns: "com.example", schema: "posts", id: "123", attr: "title"}, depth: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.depth, tt.uri.Depth())
		})
	}
}

func TestMustNewURI(t *testing.T) {
	uri := MustNewURI("com.example", "posts", "123")
	assert.Equal(t, "xdb://com.example/posts/123", uri.String())

	assert.Panics(t, func() {
		MustNewURI("bad ns")
	})
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

			assert.Equal(t, tt.ns, uri.NS())
			assert.Equal(t, tt.schema, uri.Schema())
			assert.Equal(t, tt.id, uri.ID())
			assert.Equal(t, tt.attr, uri.Attr())
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

func TestURIRoundtripProperty(t *testing.T) {
	uris := []string{
		"xdb://com.example",
		"xdb://com.example/posts",
		"xdb://com.example/posts/123",
		"xdb://com.example/posts/123#title",
		"xdb://com.example/posts/a/b/c",
		"xdb://com.example/posts/a/b/c#author.id",
	}

	for _, s := range uris {
		t.Run(s, func(t *testing.T) {
			u := MustParseURI(s)
			require.Equal(t, s, u.String())

			reparsed := MustParseURI(u.String())
			assert.Equal(t, *u, *reparsed)
		})
	}
}

func TestParseURIErrorsExtra(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{name: "slash in attr", uri: "xdb://com.example/posts/123#a/b"},
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

	assert.Equal(t, "com.example", uri.NS())
	assert.Equal(t, "posts", uri.Schema())
	assert.Equal(t, "123", uri.ID())
	assert.Equal(t, "title", uri.Attr())
}

func TestMustParseURI(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123")
	assert.Equal(t, "com.example", uri.NS())
	assert.Equal(t, "posts", uri.Schema())
	assert.Equal(t, "123", uri.ID())
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

	assert.Equal(t, *a, *b)
	assert.NotEqual(t, *a, *c)
}

func TestURIEqualsNilComponents(t *testing.T) {
	a := MustParseURI("xdb://com.example")
	b := MustParseURI("xdb://com.example")
	c := MustParseURI("xdb://com.example/posts")

	assert.Equal(t, *a, *b)
	assert.NotEqual(t, *a, *c)
}

func TestURISchemaURI(t *testing.T) {
	uri := MustParseURI("xdb://com.example/posts/123#title")
	schemaURI := uri.SchemaURI()

	assert.Equal(t, "com.example", schemaURI.NS())
	assert.Equal(t, "posts", schemaURI.Schema())
	assert.Empty(t, schemaURI.ID())
	assert.Empty(t, schemaURI.Attr())
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

	assert.Equal(t, "com.example", uri.NS())
	assert.Equal(t, "posts", uri.Schema())
	assert.Equal(t, "123", uri.ID())
}

func TestURIUnmarshalJSONErrors(t *testing.T) {
	var uri URI

	err := json.Unmarshal([]byte(`123`), &uri)
	require.Error(t, err)

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

	assert.Equal(t, *original, decoded)
}
