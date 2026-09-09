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

// VersionSuite checks record metadata and version preconditions through
// [store.Store] on each backend. It covers sequential version checks;
// concurrent checks and writes require a transactional backend.
type VersionSuite struct {
	newStore func() store.Store
}

// NewVersionSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewVersionSuite(fn func() store.Store) *VersionSuite {
	return &VersionSuite{newStore: fn}
}

// Run runs all versioning tests as subtests of t.
func (s *VersionSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Stamping", s.testStamping)
	t.Run("CAS", s.testCAS)
	t.Run("Derived", s.testDerived)
	t.Run("Lifecycle", s.testLifecycle)
	t.Run("Filtering", s.testFiltering)
}

// testFiltering pins that system fields are queryable, not just
// readable. Backends with filter pushdown must resolve them the same
// way the in-process filter does.
func (s *VersionSuite) testFiltering(t *testing.T) {
	ctx := context.Background()
	st := s.seed(t)
	scope := core.MustParseURI("xdb://com.example/vposts")

	require.NoError(t, st.CreateRecord(ctx,
		core.NewRecord("com.example", "vposts", "f1").Set("title", "a"),
	))
	require.NoError(t, st.CreateRecord(ctx,
		core.NewRecord("com.example", "vposts", "f2").Set("title", "b"),
	))
	// Bump f2 so the two records differ by version.
	require.NoError(t, st.UpsertRecord(ctx,
		core.NewRecord("com.example", "vposts", "f2").Set("title", "b2"),
	))

	t.Run("filter by _version", func(t *testing.T) {
		page, err := st.ListRecords(ctx, &store.Query{
			URI:    scope,
			Filter: "_version == 2",
		})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "f2", page.Items[0].URI().ID())
	})

	t.Run("filter by _id", func(t *testing.T) {
		page, err := st.ListRecords(ctx, &store.Query{
			URI:    scope,
			Filter: `_id == "f1"`,
		})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "f1", page.Items[0].URI().ID())
	})
}

// seed creates a strict schema so the system fields must be declared by
// the stamping middleware for writes to validate at all.
func (s *VersionSuite) seed(t *testing.T) store.Store {
	t.Helper()

	ctx := context.Background()
	st := s.newStore()
	uri := core.MustParseURI("xdb://com.example/vposts")

	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"body":  {Type: core.TypeString},
		},
	}))

	return st
}

func (s *VersionSuite) version(
	t *testing.T,
	st store.Store,
	uri *core.URI,
) int64 {
	t.Helper()

	record, err := st.GetRecord(context.Background(), uri)
	require.NoError(t, err)

	v, err := record.Get(schema.FieldVersion).AsInt()
	require.NoError(t, err)

	return v
}

func (s *VersionSuite) testStamping(t *testing.T) {
	ctx := context.Background()
	st := s.seed(t)
	uri := core.MustParseURI("xdb://com.example/vposts/p1")

	t.Run("create stamps version 1 and a timestamp", func(t *testing.T) {
		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "p1").Set("title", "Hello"),
		))

		got, err := st.GetRecord(ctx, uri)
		require.NoError(t, err)

		v, err := got.Get(schema.FieldVersion).AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)

		ts, err := got.Get(schema.FieldUpdated).AsTime()
		require.NoError(t, err)
		assert.False(t, ts.IsZero())
	})

	t.Run("_id is projected from the path", func(t *testing.T) {
		got, err := st.GetRecord(ctx, uri)
		require.NoError(t, err)

		id, err := got.Get(schema.FieldID).AsStr()
		require.NoError(t, err)
		assert.Equal(t, "p1", id)
	})

	t.Run("upsert increments", func(t *testing.T) {
		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "p1").Set("title", "Second"),
		))
		assert.Equal(t, int64(2), s.version(t, st, uri))
	})

	t.Run("tuple merge increments", func(t *testing.T) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/vposts/p1", "body", "Text"),
		))
		assert.Equal(t, int64(3), s.version(t, st, uri))
	})

	t.Run("attr delete increments", func(t *testing.T) {
		require.NoError(t, st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/vposts/p1#body"),
		))
		assert.Equal(t, int64(4), s.version(t, st, uri))
	})

	t.Run("listed records carry system fields", func(t *testing.T) {
		page, err := st.ListRecords(ctx, &store.Query{
			URI: core.MustParseURI("xdb://com.example/vposts"),
		})
		require.NoError(t, err)
		require.NotEmpty(t, page.Items)

		for _, record := range page.Items {
			assert.NotNil(t, record.Get(schema.FieldID))
			assert.NotNil(t, record.Get(schema.FieldVersion))
			assert.NotNil(t, record.Get(schema.FieldUpdated))
		}
	})
}

func (s *VersionSuite) testCAS(t *testing.T) {
	ctx := context.Background()

	t.Run("matching version succeeds and bumps", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/cas1")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas1").Set("title", "a"),
		))

		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas1").
				Set("title", "b").
				Set(schema.FieldVersion, int64(1)),
		))

		assert.Equal(t, int64(2), s.version(t, st, uri))
	})

	t.Run("stale version conflicts and does not write", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/cas2")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas2").Set("title", "a"),
		))
		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas2").Set("title", "b"),
		))

		err := st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas2").
				Set("title", "c").
				Set(schema.FieldVersion, int64(1)),
		)
		require.ErrorIs(t, err, core.ErrConflict)

		got, err := st.GetRecord(ctx, uri)
		require.NoError(t, err)
		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "b", title, "conflicting write must not land")
	})

	t.Run("omitted version writes unconditionally", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/cas3")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas3").Set("title", "a"),
		))
		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas3").Set("title", "b"),
		))

		assert.Equal(t, int64(2), s.version(t, st, uri))
	})

	// Writing back the version from a prior read must reject a stale copy.
	t.Run("read-modify-write is protected by default", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/cas4")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "cas4").Set("title", "a"),
		))

		got, err := st.GetRecord(ctx, uri)
		require.NoError(t, err)

		got.Set("title", "edited")
		require.NoError(t, st.UpsertRecord(ctx, got))

		got.Set("title", "stale")
		require.ErrorIs(t, st.UpsertRecord(ctx, got), core.ErrConflict)
	})
}

func (s *VersionSuite) testDerived(t *testing.T) {
	ctx := context.Background()
	st := s.seed(t)

	t.Run("client-supplied _updated is ignored", func(t *testing.T) {
		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "d1").
				Set("title", "a").
				Set(schema.FieldUpdated, "not-a-time"),
		))

		got, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/vposts/d1"))
		require.NoError(t, err)

		ts, err := got.Get(schema.FieldUpdated).AsTime()
		require.NoError(t, err)
		assert.False(t, ts.IsZero())
	})

	t.Run("echoed _id matching the path is ignored", func(t *testing.T) {
		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "d2").
				Set("title", "a").
				Set(schema.FieldID, "d2"),
		))
	})

	t.Run("_id disagreeing with the path is rejected", func(t *testing.T) {
		err := st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "d3").
				Set("title", "a").
				Set(schema.FieldID, "elsewhere"),
		)
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("deleting a system attr is rejected", func(t *testing.T) {
		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "d4").Set("title", "a"),
		))

		err := st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/vposts/d4#"+schema.FieldVersion),
		)
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})
}

func (s *VersionSuite) testLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("delete then recreate restarts at 1", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/l1")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "l1").Set("title", "a"),
		))
		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vposts", "l1").Set("title", "b"),
		))
		require.Equal(t, int64(2), s.version(t, st, uri))

		require.NoError(t, st.DeleteRecord(ctx, uri))
		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "l1").Set("title", "c"),
		))

		assert.Equal(t, int64(1), s.version(t, st, uri))
	})

	// Removing the last user tuple must also remove the system tuples.
	t.Run("removing the last user tuple removes the record", func(t *testing.T) {
		st := s.seed(t)
		uri := core.MustParseURI("xdb://com.example/vposts/l2")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vposts", "l2").Set("title", "a"),
		))
		require.NoError(t, st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/vposts/l2#title"),
		))

		_, err := st.GetRecord(ctx, uri)
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("schema-less records are versioned too", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/vfree/f1")

		require.NoError(t, st.CreateRecord(ctx,
			core.NewRecord("com.example", "vfree", "f1").Set("name", "x"),
		))
		assert.Equal(t, int64(1), s.version(t, st, uri))

		require.NoError(t, st.UpsertRecord(ctx,
			core.NewRecord("com.example", "vfree", "f1").Set("name", "y"),
		))
		assert.Equal(t, int64(2), s.version(t, st, uri))
	})
}
