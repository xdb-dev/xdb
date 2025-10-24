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
		want *core.URI
	}{
		{
			name: "Repo URI",
			uri:  "xdb://com.example",
			want: core.NewURI("com.example"),
		},
		{
			name: "Record URI",
			uri:  "xdb://com.example.posts/123",
			want: core.NewURI("com.example.posts", "123"),
		},
		{
			name: "Hierarchy ID URI",
			uri:  "xdb://com.example.posts/123/456",
			want: core.NewURI("com.example.posts", []string{"123", "456"}),
		},
		{
			name: "Attribute URI",
			uri:  "xdb://com.example.posts/123#name",
			want: core.NewURI("com.example.posts", "123", "name"),
		},
		{
			name: "Nested Attribute URI",
			uri:  "xdb://com.example.posts/123#profile.name",
			want: core.NewURI("com.example.posts", "123", []string{"profile", "name"}),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := core.ParseURI(tc.uri)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.String(), uri.String())
		})
	}
}
