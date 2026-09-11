package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// QuerySuite checks filter behavior through the store facade on each backend.
// It also tests native [store.QueryDriver] pushdown when the driver supports it.
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

	// Check facade filtering on every backend, including drivers that use
	// in-memory evaluation and queries that native pushdown declines.
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

// runFilterHardening checks [store.Store.ListRecords] filter semantics
// through [store.New], covering native pushdown and in-memory evaluation.
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

	t.Run("presence filter selects records without a field", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/triage_issues")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title":    {Type: core.TypeString},
				"assignee": {Type: core.TypeString},
			},
		}))

		assigned := core.NewRecord("com.example", "triage_issues", "t1")
		assigned.Set("title", "crash on start")
		assigned.Set("assignee", "priya")
		require.NoError(t, st.CreateRecord(ctx, assigned))

		unassigned := core.NewRecord("com.example", "triage_issues", "t2")
		unassigned.Set("title", "slow query")
		require.NoError(t, st.CreateRecord(ctx, unassigned))

		page, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `!has(assignee)`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "t2", page.Items[0].URI().ID())

		page, err = st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `has(assignee)`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "t1", page.Items[0].URI().ID())
	})

	t.Run("regex filter works on every backend", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/regex_posts")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
			},
		}))

		for id, title := range map[string]string{"r1": "Hello", "r2": "Goodbye"} {
			r := core.NewRecord("com.example", "regex_posts", id)
			r.Set("title", title)
			require.NoError(t, st.CreateRecord(ctx, r))
		}

		// A SQL backend has no translation for matches(). It must fall
		// back to a scan, not refuse the query.
		page, err := st.ListRecords(ctx, &store.Query{
			URI:    uri,
			Filter: `title.matches("^Hel")`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "r1", page.Items[0].URI().ID())
	})

	t.Run("time range filter selects one month", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/ledger_entries")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"at": {Type: core.TypeTime},
			},
		}))

		days := map[string]time.Time{
			"e1": time.Date(2026, 7, 31, 23, 0, 0, 0, time.UTC),
			"e2": time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
			"e3": time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		}
		for id, at := range days {
			r := core.NewRecord("com.example", "ledger_entries", id)
			r.Set("at", at)
			require.NoError(t, st.CreateRecord(ctx, r))
		}

		page, err := st.ListRecords(ctx, &store.Query{
			URI: uri,
			Filter: `at >= timestamp("2026-08-01T00:00:00Z") && ` +
				`at < timestamp("2026-09-01T00:00:00Z")`,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, page.Total)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "e2", page.Items[0].URI().ID())
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
