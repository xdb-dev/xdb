package xdbsqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func newTestStore(t *testing.T) *xdbsqlite.Store {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	s, err := xdbsqlite.New(db)
	require.NoError(t, err)

	return s
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	var _ store.Store = s
	var _ store.HealthChecker = s
	var _ store.BatchExecutor = s
}

func TestHealth(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Health(context.Background()))
}

func TestRecords(t *testing.T) {
	tests.NewRecordStoreSuite(func() store.RecordStore {
		return newTestStore(t)
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	tests.NewSchemaStoreSuite(func() store.SchemaStore {
		return newTestStore(t)
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	tests.NewNamespaceStoreSuite(func() tests.NamespaceStore {
		return newTestStore(t)
	}).Run(t)
}

func TestBatch(t *testing.T) {
	tests.NewBatchSuite(func() tests.BatchStore {
		return newTestStore(t)
	}).Run(t)
}

func TestModes(t *testing.T) {
	tests.NewModeStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

func TestUpdateSchema_DDLEvolution(t *testing.T) {
	ctx := context.Background()

	strictSchema := func(fields map[string]schema.FieldDef) *schema.Def {
		return &schema.Def{
			URI:    core.MustParseURI("xdb://com.test/evolve"),
			Mode:   schema.ModeStrict,
			Fields: fields,
		}
	}

	t.Run("adds column for new field", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
			"count": {Type: core.TIDInteger},
		})))

		// Verify: write a record using the new column.
		r := core.NewRecord("com.test", "evolve", "r1")
		r.Set("title", "hello")
		r.Set("count", int64(42))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "hello", title)
		count, err := got.Get("count").AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
	})

	t.Run("drops column for removed field", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
			"extra": {Type: core.TIDString},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
		})))

		// Verify schema only has title.
		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Len(t, def.Fields, 1)
		assert.Contains(t, def.Fields, "title")
	})

	t.Run("adds and drops in same update", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"a": {Type: core.TIDString},
			"b": {Type: core.TIDInteger},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"a": {Type: core.TIDString},
			"c": {Type: core.TIDFloat},
		})))

		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Contains(t, def.Fields, "a")
		assert.Contains(t, def.Fields, "c")
		assert.NotContains(t, def.Fields, "b")
	})

	t.Run("rejects type change", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
		})))

		err := st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDInteger},
		}))
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("rejects mode change", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
		})))

		err := st.UpdateSchema(ctx, uri, &schema.Def{
			URI:    uri,
			Mode:   schema.ModeFlexible,
			Fields: map[string]schema.FieldDef{"title": {Type: core.TIDString}},
		})
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("no-op for flexible mode", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
		}))

		// Adding fields to a flexible schema updates metadata only.
		require.NoError(t, st.UpdateSchema(ctx, uri, &schema.Def{
			URI:    uri,
			Mode:   schema.ModeFlexible,
			Fields: map[string]schema.FieldDef{"x": {Type: core.TIDString}},
		}))

		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Contains(t, def.Fields, "x")
	})

	t.Run("records work after evolution", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		// Create with one field, write a record.
		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
		})))
		r1 := core.NewRecord("com.test", "evolve", "r1")
		r1.Set("title", "hello")
		require.NoError(t, st.CreateRecord(ctx, r1))

		// Evolve: add a field.
		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.FieldDef{
			"title": {Type: core.TIDString},
			"count": {Type: core.TIDInteger},
		})))

		// Old record should still be readable (count is null/zero).
		got, err := st.GetRecord(ctx, r1.URI())
		require.NoError(t, err)
		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "hello", title)

		// New record with both fields.
		r2 := core.NewRecord("com.test", "evolve", "r2")
		r2.Set("title", "world")
		r2.Set("count", int64(7))
		require.NoError(t, st.CreateRecord(ctx, r2))

		got2, err := st.GetRecord(ctx, r2.URI())
		require.NoError(t, err)
		title2, err := got2.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "world", title2)
		count, err := got2.Get("count").AsInt()
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
	})
}
