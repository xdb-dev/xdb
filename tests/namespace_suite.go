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

// NamespaceStore combines [store.NamespaceReader] and [store.SchemaStore].
// Namespaces are derived from schemas, so a [store.SchemaStore] is required to seed data.
type NamespaceStore interface {
	store.NamespaceReader
	store.SchemaStore
}

// NamespaceStoreSuite runs a standard set of tests against a [NamespaceStore].
type NamespaceStoreSuite struct {
	newStore func() NamespaceStore
}

// NewNamespaceStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewNamespaceStoreSuite(fn func() NamespaceStore) *NamespaceStoreSuite {
	return &NamespaceStoreSuite{newStore: fn}
}

// Run runs all namespace tests as subtests of t.
func (s *NamespaceStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Get", s.testGet)
	t.Run("List", s.testList)
}

func (s *NamespaceStoreSuite) seedSchema(
	t *testing.T,
	st store.SchemaStore,
	rawURI string,
) {
	t.Helper()
	ctx := context.Background()
	uri := core.MustParseURI(rawURI)
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{URI: uri}))
}

func (s *NamespaceStoreSuite) testGet(t *testing.T) {
	ctx := context.Background()

	t.Run("returns namespace when schemas exist", func(t *testing.T) {
		st := s.newStore()
		s.seedSchema(t, st, "xdb://com.example/posts")

		nsURI := core.MustParseURI("xdb://com.example")
		ns, err := st.GetNamespace(ctx, nsURI)
		require.NoError(t, err)
		assert.Equal(t, "com.example", ns.String())
	})

	t.Run("not found", func(t *testing.T) {
		st := s.newStore()

		uri := core.MustParseURI("xdb://com.missing")
		_, err := st.GetNamespace(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("disappears when all schemas deleted", func(t *testing.T) {
		st := s.newStore()

		schemaURI := core.MustParseURI("xdb://com.example/posts")
		s.seedSchema(t, st, schemaURI.String())

		require.NoError(t, st.DeleteSchema(ctx, schemaURI))

		nsURI := core.MustParseURI("xdb://com.example")
		_, err := st.GetNamespace(ctx, nsURI)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *NamespaceStoreSuite) testList(t *testing.T) {
	ctx := context.Background()

	t.Run("returns unique namespaces", func(t *testing.T) {
		st := s.newStore()
		s.seedSchema(t, st, "xdb://com.example/posts")
		s.seedSchema(t, st, "xdb://com.example/users")
		s.seedSchema(t, st, "xdb://com.other/posts")

		page, err := st.ListNamespaces(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, page.Total)
		assert.Len(t, page.Items, 2)
	})
}
