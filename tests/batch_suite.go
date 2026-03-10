package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// BatchStore combines [store.Store] and [store.BatchExecutor].
type BatchStore interface {
	store.Store
	store.BatchExecutor
}

// BatchSuite runs a standard set of tests against a [BatchStore].
type BatchSuite struct {
	newStore func() BatchStore
}

// NewBatchSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewBatchSuite(fn func() BatchStore) *BatchSuite {
	return &BatchSuite{newStore: fn}
}

// Run runs all batch tests as subtests of t.
func (s *BatchSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("commits on success", s.testCommit)
	t.Run("rolls back on error", s.testRollback)
}

func (s *BatchSuite) testCommit(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	err := st.ExecuteBatch(ctx, func(tx store.Store) error {
		r1 := core.NewRecord("com.example", "posts", "batch-commit-1")
		r1.Set("title", "First")
		if err := tx.CreateRecord(ctx, r1); err != nil {
			return err
		}

		r2 := core.NewRecord("com.example", "posts", "batch-commit-2")
		r2.Set("title", "Second")
		return tx.CreateRecord(ctx, r2)
	})
	require.NoError(t, err)

	_, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/batch-commit-1"))
	require.NoError(t, err)
	_, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/batch-commit-2"))
	require.NoError(t, err)
}

func (s *BatchSuite) testRollback(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	existing := core.NewRecord("com.example", "posts", "batch-existing")
	require.NoError(t, st.CreateRecord(ctx, existing))

	err := st.ExecuteBatch(ctx, func(tx store.Store) error {
		r := core.NewRecord("com.example", "posts", "batch-rollback")
		if err := tx.CreateRecord(ctx, r); err != nil {
			return err
		}
		return assert.AnError
	})
	require.Error(t, err)

	_, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/batch-rollback"))
	require.ErrorIs(t, err, store.ErrNotFound)

	_, err = st.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/batch-existing"))
	require.NoError(t, err)
}
