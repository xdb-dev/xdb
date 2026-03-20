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

// CascadeStoreSuite runs tests that verify [store.SchemaWriter.DeleteSchemaRecords]
// across all store implementations.
type CascadeStoreSuite struct {
	newStore func() store.Store
}

// NewCascadeStoreSuite creates a new suite using the given factory.
func NewCascadeStoreSuite(fn func() store.Store) *CascadeStoreSuite {
	return &CascadeStoreSuite{newStore: fn}
}

// Run runs all cascade delete tests as subtests of t.
func (s *CascadeStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("DeleteSchemaRecords", s.testDeleteSchemaRecords)
	t.Run("DeleteSchemaRecords_NoRecords", s.testDeleteSchemaRecordsEmpty)
	t.Run("DeleteSchemaRecords_PreservesOtherSchemas", s.testPreservesOtherSchemas)
}

func (s *CascadeStoreSuite) testDeleteSchemaRecords(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeFlexible,
	}))

	// Create records.
	for _, id := range []string{"a", "b", "c"} {
		r := core.NewRecord("com.example", "posts", id)
		r.Set("title", "Post "+id)
		require.NoError(t, st.CreateRecord(ctx, r))
	}

	// Verify records exist.
	page, err := st.ListRecords(ctx, &store.Query{URI: uri})
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)

	// Cascade: delete records then schema (the real use case).
	require.NoError(t, st.DeleteSchemaRecords(ctx, uri))
	require.NoError(t, st.DeleteSchema(ctx, uri))

	// Schema should be gone.
	_, err = st.GetSchema(ctx, uri)
	require.ErrorIs(t, err, store.ErrNotFound)
}

func (s *CascadeStoreSuite) testDeleteSchemaRecordsEmpty(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	uri := core.MustParseURI("xdb://com.example/empty")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeFlexible,
	}))

	// No records — should be a no-op.
	require.NoError(t, st.DeleteSchemaRecords(ctx, uri))
}

func (s *CascadeStoreSuite) testPreservesOtherSchemas(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	postsURI := core.MustParseURI("xdb://com.example/posts")
	usersURI := core.MustParseURI("xdb://com.example/users")

	require.NoError(t, st.CreateSchema(ctx, postsURI, &schema.Def{
		URI:  postsURI,
		Mode: schema.ModeFlexible,
	}))
	require.NoError(t, st.CreateSchema(ctx, usersURI, &schema.Def{
		URI:  usersURI,
		Mode: schema.ModeFlexible,
	}))

	// Create records in both schemas.
	post := core.NewRecord("com.example", "posts", "p1")
	post.Set("title", "Post")
	require.NoError(t, st.CreateRecord(ctx, post))

	user := core.NewRecord("com.example", "users", "u1")
	user.Set("name", "Alice")
	require.NoError(t, st.CreateRecord(ctx, user))

	// Cascade delete posts only.
	require.NoError(t, st.DeleteSchemaRecords(ctx, postsURI))
	require.NoError(t, st.DeleteSchema(ctx, postsURI))

	// Posts schema gone.
	_, postSchemaErr := st.GetSchema(ctx, postsURI)
	require.ErrorIs(t, postSchemaErr, store.ErrNotFound)

	// Users schema and record still exist.
	_, usersSchemaErr := st.GetSchema(ctx, usersURI)
	require.NoError(t, usersSchemaErr)

	userURI := core.MustParseURI("xdb://com.example/users/u1")
	got, getErr := st.GetRecord(ctx, userURI)
	require.NoError(t, getErr)

	name, nameErr := got.Get("name").AsStr()
	require.NoError(t, nameErr)
	assert.Equal(t, "Alice", name)
}
