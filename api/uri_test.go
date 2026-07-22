package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		method    string
		minDepth  int
		maxDepth  int
		allowAttr bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "records.get happy path with attr",
			raw:       "xdb://com.example/posts/123#title",
			method:    "records.get",
			minDepth:  3,
			maxDepth:  3,
			allowAttr: true,
		},
		{
			name:      "records.get wrong depth",
			raw:       "xdb://com.example",
			method:    "records.get",
			minDepth:  3,
			maxDepth:  3,
			allowAttr: true,
			wantErr:   true,
			errMsg:    `records.get expects a record URI xdb://ns/schema/id, got "xdb://com.example"`,
		},
		{
			name:      "records.create rejects attr",
			raw:       "xdb://com.example/posts/123#title",
			method:    "records.create",
			minDepth:  3,
			maxDepth:  3,
			allowAttr: false,
			wantErr:   true,
			errMsg:    "records.create does not accept an attribute",
		},
		{
			name:      "records.list accepts namespace depth",
			raw:       "xdb://com.example",
			method:    "records.list",
			minDepth:  1,
			maxDepth:  2,
			allowAttr: false,
		},
		{
			name:      "records.list accepts schema depth",
			raw:       "xdb://com.example/posts",
			method:    "records.list",
			minDepth:  1,
			maxDepth:  2,
			allowAttr: false,
		},
		{
			name:      "records.list rejects record depth",
			raw:       "xdb://com.example/posts/123",
			method:    "records.list",
			minDepth:  1,
			maxDepth:  2,
			allowAttr: false,
			wantErr:   true,
			errMsg:    "a namespace or schema URI xdb://ns[/schema]",
		},
		{
			name:      "schemas.get exact depth",
			raw:       "xdb://com.example/posts",
			method:    "schemas.get",
			minDepth:  2,
			maxDepth:  2,
			allowAttr: false,
		},
		{
			name:      "schemas.get wrong depth",
			raw:       "xdb://com.example",
			method:    "schemas.get",
			minDepth:  2,
			maxDepth:  2,
			allowAttr: false,
			wantErr:   true,
			errMsg:    "a schema URI xdb://ns/schema",
		},
		{
			name:      "namespaces.get exact depth",
			raw:       "xdb://com.example",
			method:    "namespaces.get",
			minDepth:  1,
			maxDepth:  1,
			allowAttr: false,
		},
		{
			name:      "namespaces.get wrong depth",
			raw:       "xdb://com.example/posts",
			method:    "namespaces.get",
			minDepth:  1,
			maxDepth:  1,
			allowAttr: false,
			wantErr:   true,
			errMsg:    "a namespace URI xdb://ns",
		},
		{
			name:      "watch accepts full range",
			raw:       "xdb://com.example/posts/123",
			method:    "watch",
			minDepth:  1,
			maxDepth:  3,
			allowAttr: true,
		},
		{
			name:      "unsupported range falls back to a generic depth hint",
			raw:       "xdb://com.example",
			method:    "watch",
			minDepth:  2,
			maxDepth:  3,
			allowAttr: true,
			wantErr:   true,
			errMsg:    "a URI of depth 2-3",
		},
		{
			name:      "malformed URI passes through as ErrInvalidURI",
			raw:       "not-a-uri",
			method:    "records.get",
			minDepth:  3,
			maxDepth:  3,
			allowAttr: true,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := parseURI(tt.raw, tt.method, tt.minDepth, tt.maxDepth, tt.allowAttr)
			if tt.wantErr {
				assert.ErrorIs(t, err, core.ErrInvalidURI)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, uri)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, uri)
		})
	}
}
