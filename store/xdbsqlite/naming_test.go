package xdbsqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestKVTableName(t *testing.T) {
	uri := core.MustNewURI("myns", "posts", "abc")
	assert.Equal(t, `"kv:myns/posts"`, kvTableName(uri))
}

func TestColumnTableName(t *testing.T) {
	uri := core.MustNewURI("myns", "posts", "abc")
	assert.Equal(t, `"t:myns/posts"`, columnTableName(uri))
}

func TestParseKVTable(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantOK  bool
		wantURI string
	}{
		{"valid", "kv:app/posts", true, "xdb://app/posts"},
		{"column table", "t:app/posts", false, ""},
		{"no prefix", "app/posts", false, ""},
		{"missing separator", "kv:apponly", false, ""},
		{"schemas table", "_schemas", false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uri, ok := parseKVTable(tc.in)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.NotNil(t, uri)
				assert.Equal(t, tc.wantURI, uri.String())
			}
		})
	}
}

// TestScanTargets sets up all three storage situations under one
// namespace and asserts scanTargets enumerates them once each, sorted,
// with the right def (which picks the engine): a strict def (column
// table), a flexible def (which owns both a _schemas row and a kv
// table — must not double-count), and a schema-less kv table (no def).
func TestScanTargets(t *testing.T) {
	ctx := context.Background()
	d := newDriver(t)

	require.NoError(t, d.CreateSchema(ctx, &schema.Def{
		URI:    core.MustParseURI("xdb://app/aaa"),
		Mode:   schema.ModeStrict,
		Fields: map[string]schema.Field{"x": {Type: core.TypeString}},
	}))
	require.NoError(t, d.CreateSchema(ctx, &schema.Def{
		URI:  core.MustParseURI("xdb://app/bbb"),
		Mode: schema.ModeFlexible,
	}))
	// Schema-less write: creates a kv table with no _schemas row.
	require.NoError(t, d.Apply(ctx, store.Mutation{
		Path:   core.MustNewURI("app", "ccc", "id1"),
		Op:     store.OpPut,
		Tuples: []*core.Tuple{core.NewTuple("app/ccc/id1", "v", "y")},
	}))

	q := xsql.NewQueries(d.db)

	t.Run("namespace scope enumerates each once, sorted", func(t *testing.T) {
		targets, err := scanTargets(ctx, q, core.MustParseURI("xdb://app"))
		require.NoError(t, err)
		require.Len(t, targets, 3)

		assert.Equal(t, "aaa", targets[0].uri.Schema())
		assert.Equal(t, "bbb", targets[1].uri.Schema())
		assert.Equal(t, "ccc", targets[2].uri.Schema())

		require.NotNil(t, targets[0].def, "strict def present")
		require.NotNil(t, targets[1].def, "flexible def present")
		assert.Nil(t, targets[2].def, "schema-less has no def")

		assert.IsType(t, &tableEngine{}, engineFor(q, targets[0].def))
		assert.IsType(t, &kvEngine{}, engineFor(q, targets[1].def))
		assert.IsType(t, &kvEngine{}, engineFor(q, targets[2].def))
	})

	t.Run("schema scope narrows to one", func(t *testing.T) {
		targets, err := scanTargets(ctx, q, core.MustParseURI("xdb://app/bbb"))
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "bbb", targets[0].uri.Schema())
	})
}
