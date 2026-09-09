package xdbsqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/storetest"
)

// newTestStore creates a store with enforcement and versioning via [store.New].
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	return store.New(newTestDriver(t))
}

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	_ = s

	_, ok := s.(store.HealthChecker)
	require.True(t, ok, "store over sqlite driver must report health")

	_, ok = s.(store.TX)
	require.True(t, ok, "store over sqlite driver must support TX")
}

func TestHealth(t *testing.T) {
	s := newTestStore(t)
	h, ok := s.(store.HealthChecker)
	require.True(t, ok)
	require.NoError(t, h.Health(context.Background()))
}

func TestRecords(t *testing.T) {
	storetest.NewRecordStoreSuite(func() store.RecordStore {
		return newTestStore(t)
	}).Run(t)
}

func TestSchemas(t *testing.T) {
	storetest.NewSchemaStoreSuite(func() store.SchemaStore {
		return newTestStore(t)
	}).Run(t)
}

func TestNamespaces(t *testing.T) {
	storetest.NewNamespaceStoreSuite(func() storetest.NamespaceStore {
		return newTestStore(t)
	}).Run(t)
}

func TestBatch(t *testing.T) {
	storetest.NewBatchSuite(func() storetest.BatchStore {
		return newTestStore(t).(storetest.BatchStore)
	}).Run(t)
}

func TestTuples(t *testing.T) {
	storetest.NewTupleStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

func TestTypes(t *testing.T) {
	storetest.NewTypesStoreSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

func TestVersioning(t *testing.T) {
	storetest.NewVersionSuite(func() store.Store {
		return newTestStore(t)
	}).Run(t)
}

// The other policy suites (ModeStoreSuite, CascadeStoreSuite) are
// driver-independent and run once against the memory reference; see
// the tests package doc. This backend's storage behavior is pinned by
// DriverSuite + the store suites above.

func TestCreateSchema_ThenCreateRecord(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	uri := core.MustParseURI("xdb://test/books")
	def := &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title":  {Type: core.TypeString},
			"author": {Type: core.TypeString},
		},
	}
	require.NoError(t, st.CreateSchema(ctx, uri, def))

	r := core.NewRecord("test", "books", "1984")
	r.Set("title", "1984")
	r.Set("author", "George Orwell")
	require.NoError(t, st.CreateRecord(ctx, r))

	got, err := st.GetRecord(ctx, r.URI())
	require.NoError(t, err)

	title, err := got.Get("title").AsStr()
	require.NoError(t, err)
	assert.Equal(t, "1984", title)

	author, err := got.Get("author").AsStr()
	require.NoError(t, err)
	assert.Equal(t, "George Orwell", author)
}

func TestArrayField_Roundtrip(t *testing.T) {
	ctx := context.Background()

	t.Run("strict schema with typed array", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://test/posts")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
				"tags":  {Type: core.NewArrayType(core.TIDString)},
			},
		}
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		r := core.NewRecord("test", "posts", "p1")
		r.Set("title", "hello")
		r.Set("tags", core.ArrayVal(core.TIDString,
			core.StringVal("go"),
			core.StringVal("db"),
		))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		tags, err := got.Get("tags").Value().AsArray()
		require.NoError(t, err)
		require.Len(t, tags, 2)

		page, err := st.ListRecords(ctx, &store.Query{URI: uri})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		listedTags, err := page.Items[0].Get("tags").Value().AsArray()
		require.NoError(t, err)
		require.Len(t, listedTags, 2)
	})

	t.Run("dynamic schema infers array elem type", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://test/events")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString, Required: true},
			},
		}))

		r := core.NewRecord("test", "events", "e1")
		r.Set("name", "click")
		r.Set("tags", core.ArrayVal(core.TIDString,
			core.StringVal("a"),
			core.StringVal("b"),
		))
		require.NoError(t, st.CreateRecord(ctx, r))

		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		tagsField, ok := def.Fields["tags"]
		require.True(t, ok)
		assert.Equal(t, core.TIDArray, tagsField.Type.ID())
		assert.Equal(t, core.TIDString, tagsField.Type.ElemTypeID())

		page, err := st.ListRecords(ctx, &store.Query{URI: uri})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		listedTags, err := page.Items[0].Get("tags").Value().AsArray()
		require.NoError(t, err)
		require.Len(t, listedTags, 2)
	})
}

func TestUpdateSchema_DDLEvolution(t *testing.T) {
	ctx := context.Background()

	strictSchema := func(fields map[string]schema.Field) *schema.Def {
		return &schema.Def{
			URI:    core.MustParseURI("xdb://com.test/evolve"),
			Mode:   schema.ModeStrict,
			Fields: fields,
		}
	}

	t.Run("adds column for new field", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
			"count": {Type: core.TypeInt},
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

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
			"extra": {Type: core.TypeString},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
		})))

		// Verify schema only has title. System fields are stamped onto
		// every definition, so compare what the user declared.
		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		declared := schema.StripSystemFields(def).Fields
		assert.Len(t, declared, 1)
		assert.Contains(t, declared, "title")
	})

	t.Run("adds and drops in same update", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"b": {Type: core.TypeInt},
		})))

		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"a": {Type: core.TypeString},
			"c": {Type: core.TypeFloat},
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

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
		})))

		err := st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeInt},
		}))
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("rejects mode change", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
		})))

		err := st.UpdateSchema(ctx, uri, &schema.Def{
			URI:    uri,
			Mode:   schema.ModeFlexible,
			Fields: map[string]schema.Field{"title": {Type: core.TypeString}},
		})
		require.ErrorIs(t, err, core.ErrSchemaViolation)
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
			Fields: map[string]schema.Field{"x": {Type: core.TypeString}},
		}))

		def, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Contains(t, def.Fields, "x")
	})

	t.Run("records work after evolution", func(t *testing.T) {
		st := newTestStore(t)
		uri := core.MustParseURI("xdb://com.test/evolve")

		// Create with one field, write a record.
		require.NoError(t, st.CreateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
		})))
		r1 := core.NewRecord("com.test", "evolve", "r1")
		r1.Set("title", "hello")
		require.NoError(t, st.CreateRecord(ctx, r1))

		// Evolve: add a field.
		require.NoError(t, st.UpdateSchema(ctx, uri, strictSchema(map[string]schema.Field{
			"title": {Type: core.TypeString},
			"count": {Type: core.TypeInt},
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
