package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/tests"
)

func TestParseURI(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name string
		raw  string
		want *core.URI
	}{
		{
			name: "Repo URI",
			raw:  "xdb://com.example",
			want: core.New().NS("com.example").MustURI(),
		},
		{
			name: "Collection URI",
			raw:  "xdb://com.example/posts",
			want: core.New().NS("com.example").
				Schema("posts").
				MustURI(),
		},
		{
			name: "Record URI",
			raw:  "xdb://com.example/posts/123",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				MustURI(),
		},
		{
			name: "Hierarchy ID URI",
			raw:  "xdb://com.example/posts/123/456",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123/456").
				MustURI(),
		},
		{
			name: "3-level Hierarchy ID URI",
			raw:  "xdb://com.example/posts/123/456/789",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123/456/789").
				MustURI(),
		},
		{
			name: "Nested Attribute URI",
			raw:  "xdb://com.example/posts/123#name",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				Attr("name").
				MustURI(),
		},
		{
			name: "Nested Attribute URI",
			raw:  "xdb://com.example/posts/123#profile.name",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				Attr("profile.name").
				MustURI(),
		},
		{
			name: "Multiple Nested Attribute URI",
			raw:  "xdb://com.example/posts/123/456/789#profile.bio.interests",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123/456/789").
				Attr("profile.bio.interests").
				MustURI(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := core.ParseURI(tc.raw)
			assert.NoError(t, err)
			tests.AssertEqualURI(t, tc.want, uri)
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
			uri:  "com.example/posts",
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
		{
			name: "Wrong scheme http",
			uri:  "http://com.example",
		},
		{
			name: "Wrong scheme https",
			uri:  "https://com.example",
		},
		{
			name: "Wrong scheme ftp",
			uri:  "ftp://com.example",
		},
		{
			name: "URI with user info",
			uri:  "xdb://user:pass@com.example",
		},
		{
			name: "URI with user",
			uri:  "xdb://user@com.example",
		},
		{
			name: "URI with port",
			uri:  "xdb://com.example:8080",
		},
		{
			name: "Invalid schema with spaces",
			uri:  "xdb://com.example/invalid schema",
		},
		{
			name: "Invalid schema with special chars",
			uri:  "xdb://com.example/schema!@#",
		},
		{
			name: "Invalid schema with parentheses",
			uri:  "xdb://com.example/(schema)",
		},
		{
			name: "Invalid ID with spaces",
			uri:  "xdb://com.example/posts/invalid id",
		},
		{
			name: "Invalid ID with special chars",
			uri:  "xdb://com.example/posts/id@123",
		},
		{
			name: "Invalid ID with brackets",
			uri:  "xdb://com.example/posts/[123]",
		},
		{
			name: "Invalid attribute with spaces",
			uri:  "xdb://com.example/posts/123#my attr",
		},
		{
			name: "Invalid attribute with special chars",
			uri:  "xdb://com.example/posts/123#attr!@#",
		},
		{
			name: "Invalid attribute with brackets",
			uri:  "xdb://com.example/posts/123#attr[0]",
		},
		{
			name: "Multiple fragments",
			uri:  "xdb://com.example/posts/123#attr#other",
		},
		{
			name: "Empty namespace",
			uri:  "xdb:///posts",
		},
		{
			name: "Namespace with percent encoding",
			uri:  "xdb://com%2Eexample",
		},
		{
			name: "Schema with percent encoding",
			uri:  "xdb://com.example/post%20s",
		},
		{
			name: "ID with percent encoding",
			uri:  "xdb://com.example/posts/12%203",
		},
		{
			name: "Attribute with percent encoding",
			uri:  "xdb://com.example/posts/123#name%20field",
		},
		{
			name: "Invalid namespace with ampersand",
			uri:  "xdb://com&example",
		},
		{
			name: "Invalid namespace with equals",
			uri:  "xdb://com=example",
		},
		{
			name: "Invalid schema with asterisk",
			uri:  "xdb://com.example/posts*",
		},
		{
			name: "Invalid ID with plus",
			uri:  "xdb://com.example/posts/123+456",
		},
		{
			name: "Invalid attribute with dollar",
			uri:  "xdb://com.example/posts/123#$attr",
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
