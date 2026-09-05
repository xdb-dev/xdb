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

// UniqueStoreSuite verifies that a field marked unique rejects a second
// record with the same value, on every backend and in every mode. It
// requires [store.Store] because the tests create a schema and then write
// records against it.
type UniqueStoreSuite struct {
	newStore func() store.Store
}

// NewUniqueStoreSuite creates a new suite using the given factory. The
// factory is called before each test group to provide a fresh store.
func NewUniqueStoreSuite(fn func() store.Store) *UniqueStoreSuite {
	return &UniqueStoreSuite{newStore: fn}
}

// Run runs all unique enforcement tests as subtests of t.
func (s *UniqueStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Strict", func(t *testing.T) { s.testMode(t, schema.ModeStrict) })
	t.Run("Flexible", func(t *testing.T) { s.testMode(t, schema.ModeFlexible) })
	t.Run("Dynamic", func(t *testing.T) { s.testMode(t, schema.ModeDynamic) })
	t.Run("Patch", s.testPatch)
	t.Run("NullsAreDistinct", s.testNullsAreDistinct)
	t.Run("DryRun", s.testDryRun)
}

// uniqueSchema declares a members schema whose email field is unique.
func uniqueSchema(uri *core.URI, mode schema.Mode) *schema.Def {
	return &schema.Def{
		URI:  uri,
		Mode: mode,
		Fields: map[string]schema.Field{
			"email": {Type: core.TypeString, Unique: true},
			"name":  {Type: core.TypeString},
		},
	}
}

func (s *UniqueStoreSuite) testMode(t *testing.T, mode schema.Mode) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/members")
	require.NoError(t, st.CreateSchema(ctx, schemaURI, uniqueSchema(schemaURI, mode)))

	first := core.NewRecord("com.example", "members", "m1")
	first.Set("email", "ada@example.com")
	first.Set("name", "Ada")
	require.NoError(t, st.CreateRecord(ctx, first))

	t.Run("rejects a duplicate value", func(t *testing.T) {
		dup := core.NewRecord("com.example", "members", "m2")
		dup.Set("email", "ada@example.com")
		dup.Set("name", "Ada Two")

		err := st.CreateRecord(ctx, dup)
		require.ErrorIs(t, err, core.ErrUniqueViolation)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("accepts a different value", func(t *testing.T) {
		other := core.NewRecord("com.example", "members", "m3")
		other.Set("email", "grace@example.com")
		other.Set("name", "Grace")
		require.NoError(t, st.CreateRecord(ctx, other))
	})

	t.Run("a record does not collide with itself", func(t *testing.T) {
		same := core.NewRecord("com.example", "members", "m1")
		same.Set("email", "ada@example.com")
		same.Set("name", "Ada Lovelace")
		require.NoError(t, st.UpsertRecord(ctx, same))
	})
}

func (s *UniqueStoreSuite) testPatch(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/members")
	require.NoError(t, st.CreateSchema(ctx, schemaURI, uniqueSchema(schemaURI, schema.ModeStrict)))

	first := core.NewRecord("com.example", "members", "p1")
	first.Set("email", "ada@example.com")
	require.NoError(t, st.CreateRecord(ctx, first))

	second := core.NewRecord("com.example", "members", "p2")
	second.Set("email", "grace@example.com")
	require.NoError(t, st.CreateRecord(ctx, second))

	// Moving p2 onto p1's address is a duplicate.
	clash := core.NewRecord("com.example", "members", "p2")
	clash.Set("email", "ada@example.com")

	err := st.UpsertRecord(ctx, clash)
	require.ErrorIs(t, err, core.ErrUniqueViolation)
}

func (s *UniqueStoreSuite) testNullsAreDistinct(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/members")
	require.NoError(t, st.CreateSchema(ctx, schemaURI, uniqueSchema(schemaURI, schema.ModeStrict)))

	// Many records may omit a unique field: absent values are distinct.
	for _, id := range []string{"n1", "n2"} {
		r := core.NewRecord("com.example", "members", id)
		r.Set("name", "No Email")
		require.NoError(t, st.CreateRecord(ctx, r))
	}
}

// testDryRun pins that a validate-only check sees the same violation a
// write would, so `records create --dry-run` cannot report a duplicate
// as valid.
func (s *UniqueStoreSuite) testDryRun(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	validator, ok := st.(store.Validator)
	if !ok {
		t.Skip("store does not implement store.Validator")
	}

	schemaURI := core.MustParseURI("xdb://com.example/members")
	require.NoError(t, st.CreateSchema(ctx, schemaURI, uniqueSchema(schemaURI, schema.ModeStrict)))

	first := core.NewRecord("com.example", "members", "d1")
	first.Set("email", "ada@example.com")
	require.NoError(t, st.CreateRecord(ctx, first))

	dup := core.NewRecord("com.example", "members", "d2")
	dup.Set("email", "ada@example.com")

	err := validator.ValidateRecord(ctx, dup, store.OpCreate)
	require.ErrorIs(t, err, core.ErrUniqueViolation)
}
