package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// QuerySuite pins the optional [store.QueryDriver] capability: native
// filter pushdown. Like the driver Tx tests it runs against the raw
// driver and skips when the driver does not implement the capability,
// so every backend can register it uniformly — only pushdown-capable
// drivers exercise the body.
type QuerySuite struct {
	newDriver func() store.Driver
}

// NewQuerySuite creates a new suite using the given factory.
// The factory is called once to provide the driver under test.
func NewQuerySuite(fn func() store.Driver) *QuerySuite {
	return &QuerySuite{newDriver: fn}
}

// Run runs all query pushdown tests as subtests of t.
func (s *QuerySuite) Run(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	d := s.newDriver()

	qd, ok := d.(store.QueryDriver)
	if !ok {
		t.Skip("driver does not implement store.QueryDriver")
	}

	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, d.PutSchema(ctx, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"status": {Type: core.TypeString},
			"views":  {Type: core.TypeInt},
		},
	}))

	seed := []store.Mutation{
		mut("com.example/posts/p1", store.OpPut,
			putTuples("com.example/posts/p1", "status", "active", "views", int64(100))...),
		mut("com.example/posts/p2", store.OpPut,
			putTuples("com.example/posts/p2", "status", "draft", "views", int64(50))...),
		mut("com.example/posts/p3", store.OpPut,
			putTuples("com.example/posts/p3", "status", "active", "views", int64(200))...),
	}
	for _, m := range seed {
		require.NoError(t, d.Apply(ctx, m))
	}

	t.Run("pushes down a filter predicate", func(t *testing.T) {
		page, err := qd.QueryTuples(ctx, &store.Query{
			URI:    uri,
			Filter: `status == "active"`,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, page.Total)
		require.Len(t, page.Items, 2)

		// Each page item is one matching record's full tuple set.
		for _, tuples := range page.Items {
			assert.Len(t, tuples, 2)
		}
	})

	t.Run("empty filter returns all records", func(t *testing.T) {
		page, err := qd.QueryTuples(ctx, &store.Query{URI: uri})
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Len(t, page.Items, 3)
	})

	t.Run("declines namespace-scoped queries", func(t *testing.T) {
		_, err := qd.QueryTuples(ctx, &store.Query{
			URI: core.MustParseURI("xdb://com.example"),
		})
		require.ErrorIs(t, err, store.ErrUnsupportedQuery)
	})
}
