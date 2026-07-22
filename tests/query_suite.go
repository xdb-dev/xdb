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

	// Filter hardening is pinned at the facade level, unconditionally —
	// unlike the QueryDriver-only tests below, every backend takes this
	// path: native SQL pushdown (sqlite) or a scan + in-memory CEL
	// evaluation (memory/fs/redis, and sqlite itself whenever pushdown
	// declines a query). Running it uniformly is the sqlite≡memory proof.
	t.Run("FilterHardening", func(t *testing.T) {
		s.runFilterHardening(t, ctx, d)
	})

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

// runFilterHardening exercises [store.Store.ListRecords] filter semantics
// through the facade, over the given raw driver, wrapped the way every
// consumer wraps a driver: [store.New]. This is the layer at which sqlite
// (native pushdown) and memory/fs/redis (scan + in-memory CEL) are proven
// equivalent.
func (s *QuerySuite) runFilterHardening(t *testing.T, ctx context.Context, d store.Driver) {
	t.Helper()
	st := store.New(d)

	t.Run("strict schema rejects unknown filter field", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/strict_articles")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
			},
		}))

		r := core.NewRecord("com.example", "strict_articles", "a1")
		r.Set("title", "hello")
		require.NoError(t, st.CreateRecord(ctx, r))

		_, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `bogus == "x"`,
		})
		require.ErrorIs(t, err, core.ErrInvalidFilter)
		assert.Contains(t, err.Error(), `"bogus"`)
	})

	t.Run("flexible schema unknown filter field matches nothing", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/flexible_notes")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
			},
		}))

		r := core.NewRecord("com.example", "flexible_notes", "n1")
		r.Set("title", "hello")
		require.NoError(t, st.CreateRecord(ctx, r))

		page, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `bogus == "x"`,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, page.Total)
		assert.Empty(t, page.Items)
	})

	t.Run("string functions are case-sensitive", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/case_posts")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
			},
		}))

		r := core.NewRecord("com.example", "case_posts", "c1")
		r.Set("title", "Hello World")
		require.NoError(t, st.CreateRecord(ctx, r))

		page, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `title.contains("hello")`,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, page.Total, "lowercase needle must not match mixed-case data")

		page, err = st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `title.contains("Hello")`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
	})

	t.Run("dynamic schema field never written falls back without error", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/dynamic_metrics")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString, Required: true},
			},
		}))

		r := core.NewRecord("com.example", "dynamic_metrics", "m1")
		r.Set("name", "click")
		require.NoError(t, st.CreateRecord(ctx, r))

		// "score" has never been written, so it is absent from the stored
		// def's column set. filter.Compile allows it (dynamic mode is not
		// strict), but a column-strategy engine has no such column — this
		// must fall back to a scan rather than leak a raw SQL error.
		page, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `score > 10`,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, page.Total)
	})
}
