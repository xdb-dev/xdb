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
			raw:  "xdb://com.example.posts/123",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				MustURI(),
		},
		{
			name: "Hierarchy ID URI",
			raw:  "xdb://com.example.posts/123/456",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123/456").
				MustURI(),
		},
		{
			name: "Attribute URI",
			raw:  "xdb://com.example.posts/123#name",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				Attr("name").
				MustURI(),
		},
		{
			name: "Nested Attribute URI",
			raw:  "xdb://com.example.posts/123#profile.name",
			want: core.New().NS("com.example").
				Schema("posts").
				ID("123").
				Attr("profile.name").
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
