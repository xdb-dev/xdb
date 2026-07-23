package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// versionedStore returns a store with a strict schema, so the system
// fields must be declared by the stamping middleware for writes to pass
// validation at all.
func versionedStore(t *testing.T) (store.Store, *core.URI) {
	t.Helper()

	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	schemaURI := core.MustParseURI("xdb://app/posts")

	require.NoError(t, s.CreateSchema(ctx, schemaURI, &schema.Def{
		URI:  schemaURI,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"body":  {Type: core.TypeString},
		},
	}))

	return s, schemaURI
}

func versionOf(t *testing.T, s store.Store, uri *core.URI) int64 {
	t.Helper()

	record, err := s.GetRecord(context.Background(), uri)
	require.NoError(t, err)

	v, err := record.Get(schema.FieldVersion).AsInt()
	require.NoError(t, err)

	return v
}

func TestVersioning_StampsAndIncrements(t *testing.T) {
	ctx := context.Background()
	s, _ := versionedStore(t)
	uri := core.MustParseURI("xdb://app/posts/1")

	t.Run("create stamps version 1 and a timestamp", func(t *testing.T) {
		record := core.NewRecord("app", "posts", "1").Set("title", "hello")
		require.NoError(t, s.CreateRecord(ctx, record))

		got, err := s.GetRecord(ctx, uri)
		require.NoError(t, err)

		v, err := got.Get(schema.FieldVersion).AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)

		ts, err := got.Get(schema.FieldUpdated).AsTime()
		require.NoError(t, err)
		assert.False(t, ts.IsZero())
	})

	t.Run("upsert increments", func(t *testing.T) {
		record := core.NewRecord("app", "posts", "1").Set("title", "second")
		require.NoError(t, s.UpsertRecord(ctx, record))

		assert.Equal(t, int64(2), versionOf(t, s, uri))
	})

	t.Run("tuple merge increments", func(t *testing.T) {
		tuple := core.NewTuple("app/posts/1", "body", "text")
		require.NoError(t, s.PutTuples(ctx, tuple))

		assert.Equal(t, int64(3), versionOf(t, s, uri))
	})

	t.Run("attr delete increments", func(t *testing.T) {
		attr := core.MustParseURI("xdb://app/posts/1#body")
		require.NoError(t, s.DeleteTuples(ctx, attr))

		assert.Equal(t, int64(4), versionOf(t, s, uri))
	})

	t.Run("delete then recreate restarts at 1", func(t *testing.T) {
		require.NoError(t, s.DeleteRecord(ctx, uri))

		record := core.NewRecord("app", "posts", "1").Set("title", "again")
		require.NoError(t, s.CreateRecord(ctx, record))

		assert.Equal(t, int64(1), versionOf(t, s, uri))
	})
}

func TestVersioning_CAS(t *testing.T) {
	ctx := context.Background()

	t.Run("matching version succeeds and bumps", func(t *testing.T) {
		s, _ := versionedStore(t)
		uri := core.MustParseURI("xdb://app/posts/cas1")

		require.NoError(t, s.CreateRecord(ctx,
			core.NewRecord("app", "posts", "cas1").Set("title", "a"),
		))

		update := core.NewRecord("app", "posts", "cas1").
			Set("title", "b").
			Set(schema.FieldVersion, int64(1))
		require.NoError(t, s.UpsertRecord(ctx, update))

		assert.Equal(t, int64(2), versionOf(t, s, uri))
	})

	t.Run("stale version conflicts and does not write", func(t *testing.T) {
		s, _ := versionedStore(t)
		uri := core.MustParseURI("xdb://app/posts/cas2")

		require.NoError(t, s.CreateRecord(ctx,
			core.NewRecord("app", "posts", "cas2").Set("title", "a"),
		))
		require.NoError(t, s.UpsertRecord(ctx,
			core.NewRecord("app", "posts", "cas2").Set("title", "b"),
		))

		stale := core.NewRecord("app", "posts", "cas2").
			Set("title", "c").
			Set(schema.FieldVersion, int64(1))
		err := s.UpsertRecord(ctx, stale)
		require.ErrorIs(t, err, core.ErrConflict)

		got, err := s.GetRecord(ctx, uri)
		require.NoError(t, err)
		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "b", title, "conflicting write must not land")
	})

	t.Run("omitted version writes unconditionally", func(t *testing.T) {
		s, _ := versionedStore(t)
		uri := core.MustParseURI("xdb://app/posts/cas3")

		require.NoError(t, s.CreateRecord(ctx,
			core.NewRecord("app", "posts", "cas3").Set("title", "a"),
		))
		require.NoError(t, s.UpsertRecord(ctx,
			core.NewRecord("app", "posts", "cas3").Set("title", "b"),
		))

		assert.Equal(t, int64(2), versionOf(t, s, uri))
	})

	t.Run("read-modify-write round-trips safely", func(t *testing.T) {
		s, _ := versionedStore(t)
		uri := core.MustParseURI("xdb://app/posts/cas4")

		require.NoError(t, s.CreateRecord(ctx,
			core.NewRecord("app", "posts", "cas4").Set("title", "a"),
		))

		// Read, edit, write back: the version rides along as the
		// precondition without the caller doing anything.
		got, err := s.GetRecord(ctx, uri)
		require.NoError(t, err)
		got.Set("title", "edited")
		require.NoError(t, s.UpsertRecord(ctx, got))

		// A second write from the same stale copy must lose.
		got.Set("title", "stale")
		require.ErrorIs(t, s.UpsertRecord(ctx, got), core.ErrConflict)
	})
}

// Derived fields are server-owned: a client that echoes back what it
// read must not be punished for it (that is the read-modify-write path),
// but it must not be able to forge them either. So they are ignored —
// except an _id that disagrees with the URI, which is a misaddressed
// write worth catching.
func TestVersioning_DerivedFieldsAreServerOwned(t *testing.T) {
	ctx := context.Background()
	s, _ := versionedStore(t)

	t.Run("client-supplied _updated is ignored", func(t *testing.T) {
		stale := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		record := core.NewRecord("app", "posts", "d1").
			Set("title", "a").
			Set(schema.FieldUpdated, stale)

		require.NoError(t, s.CreateRecord(ctx, record))

		got, err := s.GetRecord(ctx, core.MustParseURI("xdb://app/posts/d1"))
		require.NoError(t, err)

		ts, err := got.Get(schema.FieldUpdated).AsTime()
		require.NoError(t, err)
		assert.True(t, ts.After(stale), "stamp must win over the client's value")
	})

	t.Run("echoed _id matching the path is ignored", func(t *testing.T) {
		record := core.NewRecord("app", "posts", "d2").
			Set("title", "a").
			Set(schema.FieldID, "d2")

		require.NoError(t, s.CreateRecord(ctx, record))
	})

	t.Run("_id disagreeing with the path is rejected", func(t *testing.T) {
		record := core.NewRecord("app", "posts", "d3").
			Set("title", "a").
			Set(schema.FieldID, "somewhere-else")

		require.ErrorIs(t, s.CreateRecord(ctx, record), core.ErrSchemaViolation)
	})

	t.Run("deleting a system attr is rejected", func(t *testing.T) {
		require.NoError(t, s.CreateRecord(ctx,
			core.NewRecord("app", "posts", "d4").Set("title", "a"),
		))

		err := s.DeleteTuples(ctx,
			core.MustParseURI("xdb://app/posts/d4#"+schema.FieldVersion),
		)
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})
}

func TestVersioning_VirtualID(t *testing.T) {
	ctx := context.Background()
	s, _ := versionedStore(t)
	uri := core.MustParseURI("xdb://app/posts/v1")

	require.NoError(t, s.CreateRecord(ctx,
		core.NewRecord("app", "posts", "v1").Set("title", "a"),
	))

	t.Run("GetRecord projects _id from the path", func(t *testing.T) {
		got, err := s.GetRecord(ctx, uri)
		require.NoError(t, err)

		id, err := got.Get(schema.FieldID).AsStr()
		require.NoError(t, err)
		assert.Equal(t, "v1", id)
	})

	t.Run("ListRecords projects _id", func(t *testing.T) {
		page, err := s.ListRecords(ctx, &store.Query{
			URI: core.MustParseURI("xdb://app/posts"),
		})
		require.NoError(t, err)
		require.NotEmpty(t, page.Items)

		for _, record := range page.Items {
			require.NotNil(t, record.Get(schema.FieldID),
				"every listed record carries _id")
		}
	})
}

func TestVersioning_RecordDiesWithItsLastUserTuple(t *testing.T) {
	ctx := context.Background()
	s, _ := versionedStore(t)
	uri := core.MustParseURI("xdb://app/posts/last")

	require.NoError(t, s.CreateRecord(ctx,
		core.NewRecord("app", "posts", "last").Set("title", "a"),
	))

	// Deleting the only user attr must remove the record, not leave a
	// husk of system tuples behind.
	require.NoError(t, s.DeleteTuples(ctx,
		core.MustParseURI("xdb://app/posts/last#title"),
	))

	_, err := s.GetRecord(ctx, uri)
	assert.ErrorIs(t, err, core.ErrNotFound)
}

func TestVersioning_SchemaLessRecords(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	uri := core.MustParseURI("xdb://noschema/things/1")

	require.NoError(t, s.CreateRecord(ctx,
		core.NewRecord("noschema", "things", "1").Set("name", "x"),
	))

	assert.Equal(t, int64(1), versionOf(t, s, uri))

	require.NoError(t, s.UpsertRecord(ctx,
		core.NewRecord("noschema", "things", "1").Set("name", "y"),
	))

	assert.Equal(t, int64(2), versionOf(t, s, uri))
}
