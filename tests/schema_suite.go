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

// SchemaStoreSuite runs a standard set of tests against any [store.SchemaStore].
type SchemaStoreSuite struct {
	newStore func() store.SchemaStore
}

// NewSchemaStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewSchemaStoreSuite(fn func() store.SchemaStore) *SchemaStoreSuite {
	return &SchemaStoreSuite{newStore: fn}
}

// Run runs all schema store tests as subtests of t.
func (s *SchemaStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Create", s.testCreate)
	t.Run("Get", s.testGet)
	t.Run("Update", s.testUpdate)
	t.Run("Delete", s.testDelete)
	t.Run("List", s.testList)
}

func (s *SchemaStoreSuite) testCreate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("stores and retrieves schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.FieldDef{
				"title": {Type: core.TIDString, Required: true},
			},
		}

		require.NoError(t, st.CreateSchema(ctx, uri, def))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		AssertDefEqual(t, def, got)
	})

	t.Run("rejects duplicate", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/dup")
		def := &schema.Def{URI: uri}

		require.NoError(t, st.CreateSchema(ctx, uri, def))

		err := st.CreateSchema(ctx, uri, def)
		require.ErrorIs(t, err, store.ErrAlreadyExists)
	})
}

func (s *SchemaStoreSuite) testGet(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		_, err := st.GetSchema(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testUpdate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("replaces existing schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		def := &schema.Def{URI: uri, Mode: schema.ModeFlexible}
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		updated := &schema.Def{URI: uri, Mode: schema.ModeStrict}
		require.NoError(t, st.UpdateSchema(ctx, uri, updated))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, schema.ModeStrict, got.Mode)
	})

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		err := st.UpdateSchema(ctx, uri, &schema.Def{URI: uri})
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testDelete(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("removes existing schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri}))
		require.NoError(t, st.DeleteSchema(ctx, uri))

		_, err := st.GetSchema(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		err := st.DeleteSchema(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testList(t *testing.T) {
	ctx := context.Background()

	t.Run("by namespace", func(t *testing.T) {
		st := s.newStore()

		for _, name := range []string{"posts", "users", "comments"} {
			uri := core.MustParseURI("xdb://com.example/" + name)
			require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri}))
		}

		otherURI := core.MustParseURI("xdb://com.other/posts")
		require.NoError(t, st.CreateSchema(ctx, otherURI, &schema.Def{URI: otherURI}))

		nsURI := core.MustParseURI("xdb://com.example")
		page, err := st.ListSchemas(ctx, nsURI, nil)
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Len(t, page.Items, 3)
	})

	t.Run("all namespaces", func(t *testing.T) {
		st := s.newStore()

		for _, name := range []string{"posts", "users"} {
			uri := core.MustParseURI("xdb://com.example/" + name)
			require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri}))
		}
		otherURI := core.MustParseURI("xdb://com.other/posts")
		require.NoError(t, st.CreateSchema(ctx, otherURI, &schema.Def{URI: otherURI}))

		page, err := st.ListSchemas(ctx, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
	})
}
