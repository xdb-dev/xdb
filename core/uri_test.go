package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xdb-dev/xdb/core"
)

func TestNewURI(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name  string
		parts []any
		want  string
	}{
		{
			name:  "Repository Only",
			parts: []any{"com.example.posts"},
			want:  "xdb://com.example.posts",
		},
		{
			name:  "Repository and Record",
			parts: []any{"com.example.posts", "123"},
			want:  "xdb://com.example.posts/123",
		},
		{
			name:  "Repository, Record, and Attribute",
			parts: []any{"com.example.posts", "123", "name"},
			want:  "xdb://com.example.posts/123#name",
		},
		{
			name:  "With Hierarchy ID",
			parts: []any{"com.example.posts", []string{"123", "456"}, "name"},
			want:  "xdb://com.example.posts/123/456#name",
		},
		{
			name:  "With Nested Attribute",
			parts: []any{"com.example.posts", "123", []string{"profile", "name"}},
			want:  "xdb://com.example.posts/123#profile.name",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			uri := core.NewURI(tc.parts...)
			assert.Equal(t, tc.want, uri.String())
		})
	}
}

func TestURI_Invalid(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name  string
		parts []any
	}{
		{
			name:  "Empty Parts",
			parts: []any{},
		},
		{
			name:  "Too Many Parts",
			parts: []any{"com.example.posts", "123", "name", "age"},
		},
		{
			name:  "Nil Repo",
			parts: []any{nil},
		},
		{
			name:  "Nil ID",
			parts: []any{"com.example.posts", nil, 123},
		},
		{
			name:  "Nil Attribute",
			parts: []any{"com.example.posts", "123", nil},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() {
				core.NewURI(tc.parts...)
			})
		})
	}
}

func TestParseURI(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		uri  string
		repo string
		id   core.ID
		attr core.Attr
	}{
		{
			name: "Repo URI",
			uri:  "xdb://com.example",
			repo: "com.example",
		},
		{
			name: "Record URI",
			uri:  "xdb://com.example.posts/123",
			repo: "com.example.posts",
			id:   core.NewID("123"),
		},
		{
			name: "Hierarchy ID URI",
			uri:  "xdb://com.example.posts/123/456",
			repo: "com.example.posts",
			id:   core.NewID("123", "456"),
		},
		{
			name: "Attribute URI",
			uri:  "xdb://com.example.posts/123#name",
			repo: "com.example.posts",
			id:   core.NewID("123"),
			attr: core.NewAttr("name"),
		},
		{
			name: "Nested Attribute URI",
			uri:  "xdb://com.example.posts/123#profile.name",
			repo: "com.example.posts",
			id:   core.NewID("123"),
			attr: core.NewAttr("profile", "name"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := core.ParseURI(tc.uri)
			assert.NoError(t, err)

			assert.Equal(t, tc.repo, uri.Repo())

			if len(tc.id) > 0 {
				assert.Equal(t, tc.id.String(), uri.ID().String())
			}
			if len(tc.attr) > 0 {
				assert.Equal(t, tc.attr.String(), uri.Attr().String())
			}

			want := core.NewURI(tc.repo, tc.id, tc.attr)
			assert.Equal(t, want.String(), uri.String())
		})
	}
}

func TestParseURI_Invalid(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		uri  string
	}{
		{
			name: "Empty string",
			uri:  "",
		},
		{
			name: "Missing scheme",
			uri:  "com.example.posts",
		},
		{
			name: "Wrong scheme",
			uri:  "http://com.example.posts",
		},
		{
			name: "Missing repository",
			uri:  "xdb://",
		},
		{
			name: "Invalid repo characters",
			uri:  "xdb://invalid repo",
		},
		{
			name: "Invalid repo with special chars",
			uri:  "xdb://repo@invalid",
		},
		{
			name: "Just xdb",
			uri:  "xdb",
		},
		{
			name: "Malformed URI",
			uri:  "xdb:///no-repo",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := core.ParseURI(tc.uri)
			assert.Error(t, err)
			assert.Nil(t, uri)
		})
	}
}

func TestMustParseURI(t *testing.T) {
	t.Parallel()

	t.Run("Valid URI", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example.posts/123")
		assert.NotNil(t, uri)
		assert.Equal(t, "com.example.posts", uri.Repo())
		assert.Equal(t, "123", uri.ID().String())
	})

	t.Run("Invalid URI Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			core.MustParseURI("invalid-uri")
		})
	})

	t.Run("Empty String Panics", func(t *testing.T) {
		assert.Panics(t, func() {
			core.MustParseURI("")
		})
	})
}

func TestURI_JSON(t *testing.T) {
	t.Parallel()

	t.Run("Marshal Tuple URI", func(t *testing.T) {
		uri := core.NewURI("com.example.posts", "123", "title")
		data, err := uri.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, `"xdb://com.example.posts/123#title"`, string(data))
	})

	t.Run("Unmarshal Tuple URI", func(t *testing.T) {
		var uri core.URI
		err := uri.UnmarshalJSON([]byte(`"xdb://com.example.posts/123#title"`))
		assert.NoError(t, err)
		assert.Equal(t, "com.example.posts", uri.Repo())
		assert.Equal(t, "123", uri.ID().String())
		assert.Equal(t, "title", uri.Attr().String())
	})

	t.Run("Unmarshal Invalid JSON", func(t *testing.T) {
		var uri core.URI
		err := uri.UnmarshalJSON([]byte(`invalid`))
		assert.Error(t, err)
	})

	t.Run("Unmarshal Invalid URI", func(t *testing.T) {
		var uri core.URI
		err := uri.UnmarshalJSON([]byte(`"invalid-uri"`))
		assert.Error(t, err)
	})

	t.Run("Round-trip", func(t *testing.T) {
		original := core.NewURI("com.example.posts", core.NewID("123", "456"), core.NewAttr("profile", "name"))

		data, err := original.MarshalJSON()
		assert.NoError(t, err)

		var parsed core.URI
		err = parsed.UnmarshalJSON(data)
		assert.NoError(t, err)

		assert.Equal(t, original.String(), parsed.String())
		assert.Equal(t, original.Repo(), parsed.Repo())
		assert.Equal(t, original.ID().String(), parsed.ID().String())
		assert.Equal(t, original.Attr().String(), parsed.Attr().String())
	})
}

func TestURI_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("URI with special characters in ID", func(t *testing.T) {
		// Note: IDs can contain any string, even with special chars
		uri := core.NewURI("com.example", "user-123_abc")
		assert.Equal(t, "xdb://com.example/user-123_abc", uri.String())

		parsed, err := core.ParseURI(uri.String())
		assert.NoError(t, err)
		assert.Equal(t, "user-123_abc", parsed.ID().String())
	})

	t.Run("URI with hierarchical ID", func(t *testing.T) {
		uri := core.NewURI("repo", core.NewID("a", "b", "c"), "attr")
		assert.Equal(t, "xdb://repo/a/b/c#attr", uri.String())

		parsed, err := core.ParseURI("xdb://repo/a/b/c#attr")
		assert.NoError(t, err)
		assert.Equal(t, core.NewID("a", "b", "c").String(), parsed.ID().String())
	})

	t.Run("URI with deeply nested attribute", func(t *testing.T) {
		attr := core.NewAttr("a", "b", "c", "d", "e")
		uri := core.NewURI("repo", "123", attr)
		assert.Equal(t, "xdb://repo/123#a.b.c.d.e", uri.String())

		parsed, err := core.ParseURI("xdb://repo/123#a.b.c.d.e")
		assert.NoError(t, err)
		assert.Equal(t, "a.b.c.d.e", parsed.Attr().String())
	})
}
