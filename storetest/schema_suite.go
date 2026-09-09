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
	t.Run("ArrayElemTypeEnforcement", s.testArrayElemTypeEnforcement)
	t.Run("Revision", s.testRevision)
}

// testRevision verifies schema optimistic-concurrency: CreateSchema stamps
// Revision 1, UpdateSchema bumps it, a stale base revision returns
// [core.ErrConflict], and a zero base revision updates unconditionally.
func (s *SchemaStoreSuite) testRevision(t *testing.T) {
	ctx := context.Background()

	flexDef := func(uri *core.URI, fields ...string) *schema.Def {
		fs := map[string]schema.Field{}
		for _, name := range fields {
			fs[name] = schema.Field{Type: core.TypeString}
		}
		return &schema.Def{URI: uri, Mode: schema.ModeFlexible, Fields: fs}
	}

	t.Run("create stamps revision 1", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/rev_create")
		require.NoError(t, st.CreateSchema(ctx, uri, flexDef(uri, "title")))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, int64(1), got.Revision)
	})

	t.Run("create normalizes client revision to 1", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/rev_norm")
		def := flexDef(uri, "title")
		def.Revision = 99
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, int64(1), got.Revision)
	})

	t.Run("update with correct revision bumps", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/rev_bump")
		require.NoError(t, st.CreateSchema(ctx, uri, flexDef(uri, "title")))

		cur, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		require.Equal(t, int64(1), cur.Revision)

		update := flexDef(uri, "title", "body")
		update.Revision = cur.Revision
		require.NoError(t, st.UpdateSchema(ctx, uri, update))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, int64(2), got.Revision)
	})

	t.Run("update with stale revision conflicts", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/rev_conflict")
		require.NoError(t, st.CreateSchema(ctx, uri, flexDef(uri, "title")))

		// Bump to revision 2 using the correct base.
		bump := flexDef(uri, "title", "body")
		bump.Revision = 1
		require.NoError(t, st.UpdateSchema(ctx, uri, bump))

		// Re-using the now-stale base revision 1 must conflict.
		stale := flexDef(uri, "title", "body")
		stale.Revision = 1
		err := st.UpdateSchema(ctx, uri, stale)
		require.ErrorIs(t, err, core.ErrConflict)
	})

	t.Run("update with zero revision is unconditional", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/rev_uncond")
		require.NoError(t, st.CreateSchema(ctx, uri, flexDef(uri, "title")))

		// Bump to revision 2.
		bump := flexDef(uri, "title", "body")
		bump.Revision = 1
		require.NoError(t, st.UpdateSchema(ctx, uri, bump))

		// Zero base revision updates regardless of the stored revision.
		uncond := flexDef(uri, "title", "body", "extra")
		uncond.Revision = 0
		require.NoError(t, st.UpdateSchema(ctx, uri, uncond))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Equal(t, int64(3), got.Revision)
	})
}

func (s *SchemaStoreSuite) testCreate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("stores and retrieves schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString, Required: true},
			},
		}

		require.NoError(t, st.CreateSchema(ctx, uri, def))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)

		// The store stamps system fields on the way in, and CreateSchema
		// does not write them back through the caller's def, so compare
		// against what the caller actually declared.
		AssertEqualDef(t, def, schema.StripSystemFields(got))
	})

	t.Run("rejects duplicate", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/dup")
		def := &schema.Def{URI: uri, Mode: schema.ModeFlexible}

		require.NoError(t, st.CreateSchema(ctx, uri, def))

		err := st.CreateSchema(ctx, uri, def)
		require.ErrorIs(t, err, core.ErrAlreadyExists)
	})
}

func (s *SchemaStoreSuite) testGet(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		_, err := st.GetSchema(ctx, uri)
		require.ErrorIs(t, err, core.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testUpdate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("replaces existing schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
			},
		}
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		updated := &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
				"body":  {Type: core.TypeString},
			},
		}
		require.NoError(t, st.UpdateSchema(ctx, uri, updated))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		assert.Contains(t, got.Fields, "body")
	})

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		err := st.UpdateSchema(ctx, uri, &schema.Def{URI: uri, Mode: schema.ModeFlexible})
		require.ErrorIs(t, err, core.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testDelete(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("removes existing schema", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri, Mode: schema.ModeFlexible}))
		require.NoError(t, st.DeleteSchema(ctx, uri))

		_, err := st.GetSchema(ctx, uri)
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/missing")
		err := st.DeleteSchema(ctx, uri)
		require.ErrorIs(t, err, core.ErrNotFound)
	})
}

func (s *SchemaStoreSuite) testArrayElemTypeEnforcement(t *testing.T) {
	ctx := context.Background()

	t.Run("create rejects array field without elem_type in every mode", func(t *testing.T) {
		for _, mode := range []schema.Mode{
			schema.ModeStrict,
			schema.ModeDynamic,
			schema.ModeFlexible,
		} {
			t.Run(string(mode), func(t *testing.T) {
				st := s.newStore()
				uri := core.MustParseURI(
					"xdb://com.example/missing_elem_" + string(mode),
				)
				def := &schema.Def{
					URI:  uri,
					Mode: mode,
					Fields: map[string]schema.Field{
						"tags": {Type: core.NewArrayType("")},
					},
				}

				err := st.CreateSchema(ctx, uri, def)
				require.ErrorIs(t, err, core.ErrSchemaViolation)
			})
		}
	})

	t.Run("create accepts array field with elem_type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/typed_arrays")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
			},
		}))
	})

	t.Run("update rejects changing elem_type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/immutable_tags")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
			},
		}))

		err := st.UpdateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDInteger)},
			},
		})
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("update rejects array field without elem_type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/update_missing_elem")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString},
			},
		}))

		err := st.UpdateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString},
				"tags": {Type: core.NewArrayType("")}, // missing elem_type
			},
		})
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("update rejects changing field type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/immutable_type")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString},
			},
		}))

		err := st.UpdateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeFlexible,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeInt},
			},
		})
		require.ErrorIs(t, err, core.ErrSchemaViolation)
	})
}

func (s *SchemaStoreSuite) testList(t *testing.T) {
	ctx := context.Background()

	t.Run("by namespace", func(t *testing.T) {
		st := s.newStore()

		for _, name := range []string{"posts", "users", "comments"} {
			uri := core.MustParseURI("xdb://com.example/" + name)
			require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri, Mode: schema.ModeFlexible}))
		}

		otherURI := core.MustParseURI("xdb://com.other/posts")
		require.NoError(t, st.CreateSchema(ctx, otherURI, &schema.Def{URI: otherURI, Mode: schema.ModeFlexible}))

		nsURI := core.MustParseURI("xdb://com.example")
		page, err := st.ListSchemas(ctx, &store.Query{URI: nsURI})
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Len(t, page.Items, 3)
	})

	t.Run("all namespaces", func(t *testing.T) {
		st := s.newStore()

		for _, name := range []string{"posts", "users"} {
			uri := core.MustParseURI("xdb://com.example/" + name)
			require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri, Mode: schema.ModeFlexible}))
		}
		otherURI := core.MustParseURI("xdb://com.other/posts")
		require.NoError(t, st.CreateSchema(ctx, otherURI, &schema.Def{URI: otherURI, Mode: schema.ModeFlexible}))

		page, err := st.ListSchemas(ctx, &store.Query{})
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
	})
}
