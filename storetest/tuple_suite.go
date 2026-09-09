package storetest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// TupleStoreSuite runs a standard set of tests for the tuple-level
// verbs on a [store.Store]: attr-level gets, merges, and deletes,
// including the schema policy the enforcement middleware applies to
// them.
type TupleStoreSuite struct {
	newStore func() store.Store
}

// NewTupleStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewTupleStoreSuite(fn func() store.Store) *TupleStoreSuite {
	return &TupleStoreSuite{newStore: fn}
}

// Run runs all tuple store tests as subtests of t.
func (s *TupleStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("PutGet", s.testPutGet)
	t.Run("GetTuples", s.testGetTuples)
	t.Run("Delete", s.testDelete)
	t.Run("Policy", s.testPolicy)
}

func (s *TupleStoreSuite) testPutGet(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("put creates the record", func(t *testing.T) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/posts/p1", "title", "Hello"),
			core.NewTuple("com.example/posts/p1", "author", "Alice"),
		))

		rec, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1"))
		require.NoError(t, err)

		title, err := rec.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Hello", title)
	})

	t.Run("get returns a single tuple", func(t *testing.T) {
		tuple, err := st.GetTuple(ctx,
			core.MustParseURI("xdb://com.example/posts/p1#author"))
		require.NoError(t, err)

		author, err := tuple.AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Alice", author)
	})

	t.Run("put merges without touching other attrs", func(t *testing.T) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/posts/p1", "title", "Updated"),
		))

		rec, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1"))
		require.NoError(t, err)

		title, err := rec.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Updated", title)

		author, err := rec.Get("author").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Alice", author)
	})

	t.Run("put spanning records", func(t *testing.T) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/posts/p2", "title", "Second"),
			core.NewTuple("com.example/users/u1", "name", "Bob"),
		))

		_, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p2"))
		require.NoError(t, err)
		_, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/users/u1"))
		require.NoError(t, err)
	})

	t.Run("get absent attr returns not found", func(t *testing.T) {
		_, err := st.GetTuple(ctx,
			core.MustParseURI("xdb://com.example/posts/p1#missing"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("get absent record returns not found", func(t *testing.T) {
		_, err := st.GetTuple(ctx,
			core.MustParseURI("xdb://com.example/posts/missing#title"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("get without attr is an error", func(t *testing.T) {
		_, err := st.GetTuple(ctx,
			core.MustParseURI("xdb://com.example/posts/p1"))
		require.Error(t, err)
	})
}

func (s *TupleStoreSuite) testGetTuples(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	require.NoError(t, st.PutTuples(ctx,
		core.NewTuple("com.example/posts/p1", "title", "Hello"),
		core.NewTuple("com.example/posts/p1", "author", "Alice"),
	))

	t.Run("returns present tuples omitting absences", func(t *testing.T) {
		got, err := st.GetTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/p1#title"),
			core.MustParseURI("xdb://com.example/posts/p1#missing"),
			core.MustParseURI("xdb://com.example/posts/p1#author"),
		)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "title", got[0].Attr())
		assert.Equal(t, "author", got[1].Attr())
	})

	t.Run("all absent yields empty and no error", func(t *testing.T) {
		got, err := st.GetTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/missing#title"),
		)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func (s *TupleStoreSuite) testDelete(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	seed := func(id string) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/posts/"+id, "title", "Hello"),
			core.NewTuple("com.example/posts/"+id, "author", "Alice"),
		))
	}

	t.Run("removes the tuple and keeps the record", func(t *testing.T) {
		seed("d1")

		require.NoError(t, st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/d1#author")))

		rec, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/d1"))
		require.NoError(t, err)
		assert.Nil(t, rec.Get("author"))
		assert.NotNil(t, rec.Get("title"))
	})

	t.Run("removing the last tuple removes the record", func(t *testing.T) {
		seed("d2")

		require.NoError(t, st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/d2#title"),
			core.MustParseURI("xdb://com.example/posts/d2#author"),
		))

		_, err := st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/d2"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("idempotent on absent tuples", func(t *testing.T) {
		require.NoError(t, st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/missing#title")))
	})

	t.Run("delete without attr is an error", func(t *testing.T) {
		seed("d3")

		err := st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/d3"))
		require.Error(t, err)
	})
}

func (s *TupleStoreSuite) testPolicy(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	uri := core.MustParseURI("xdb://com.example/articles")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title":  {Type: core.TypeString, Required: true},
			"author": {Type: core.TypeString},
		},
	}))

	t.Run("merge-that-creates must include required fields", func(t *testing.T) {
		err := st.PutTuples(ctx,
			core.NewTuple("com.example/articles/a1", "author", "Alice"),
		)
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("merge onto existing record needs no required fields", func(t *testing.T) {
		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/articles/a2", "title", "Hello"),
		))

		require.NoError(t, st.PutTuples(ctx,
			core.NewTuple("com.example/articles/a2", "author", "Alice"),
		))
	})

	t.Run("strict mode rejects undeclared attrs", func(t *testing.T) {
		err := st.PutTuples(ctx,
			core.NewTuple("com.example/articles/a2", "rogue", "nope"),
		)
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("type mismatches are rejected", func(t *testing.T) {
		err := st.PutTuples(ctx,
			core.NewTuple("com.example/articles/a2", "title", int64(42)),
		)
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("cannot delete a required attr", func(t *testing.T) {
		err := st.DeleteTuples(ctx,
			core.MustParseURI("xdb://com.example/articles/a2#title"))
		require.ErrorIs(t, err, core.ErrSchemaViolation)

		// The whole record is still deletable.
		require.NoError(t, st.DeleteRecord(ctx,
			core.MustParseURI("xdb://com.example/articles/a2")))
	})
}
