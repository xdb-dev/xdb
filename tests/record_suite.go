package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// RecordStoreSuite runs a standard set of tests against any [store.RecordStore].
type RecordStoreSuite struct {
	newStore func() store.RecordStore
}

// NewRecordStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewRecordStoreSuite(fn func() store.RecordStore) *RecordStoreSuite {
	return &RecordStoreSuite{newStore: fn}
}

// Run runs all record store tests as subtests of t.
func (s *RecordStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Create", s.testCreate)
	t.Run("Get", s.testGet)
	t.Run("Update", s.testUpdate)
	t.Run("Upsert", s.testUpsert)
	t.Run("Delete", s.testDelete)
	t.Run("List", s.testList)
}

func (s *RecordStoreSuite) testCreate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("stores and retrieves record", func(t *testing.T) {
		r := FakePost("create-1")
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		AssertEqualRecord(t, r, got)
	})

	t.Run("rejects duplicate", func(t *testing.T) {
		r := FakePost("create-dup")
		require.NoError(t, st.CreateRecord(ctx, r))

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrAlreadyExists)
	})
}

func (s *RecordStoreSuite) testGet(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts/missing")
		_, err := st.GetRecord(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *RecordStoreSuite) testUpdate(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("replaces existing record", func(t *testing.T) {
		r := FakePost("update-1")
		require.NoError(t, st.CreateRecord(ctx, r))

		updated := core.NewRecord("com.example", "posts", "update-1")
		updated.Set("title", "Updated Title")
		require.NoError(t, st.UpdateRecord(ctx, updated))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)

		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", title)
	})

	t.Run("not found", func(t *testing.T) {
		r := core.NewRecord("com.example", "posts", "update-missing")
		err := st.UpdateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *RecordStoreSuite) testUpsert(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("creates when missing", func(t *testing.T) {
		r := FakePost("upsert-new")
		require.NoError(t, st.UpsertRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		AssertEqualRecord(t, r, got)
	})

	t.Run("updates when present", func(t *testing.T) {
		r := FakePost("upsert-existing")
		require.NoError(t, st.CreateRecord(ctx, r))

		updated := core.NewRecord("com.example", "posts", "upsert-existing")
		updated.Set("title", "Upserted")
		require.NoError(t, st.UpsertRecord(ctx, updated))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)

		title, err := got.Get("title").AsStr()
		require.NoError(t, err)
		assert.Equal(t, "Upserted", title)
	})
}

func (s *RecordStoreSuite) testDelete(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	t.Run("removes existing record", func(t *testing.T) {
		r := FakePost("delete-1")
		require.NoError(t, st.CreateRecord(ctx, r))
		require.NoError(t, st.DeleteRecord(ctx, r.URI()))

		_, err := st.GetRecord(ctx, r.URI())
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("not found", func(t *testing.T) {
		uri := core.MustParseURI("xdb://com.example/posts/delete-missing")
		err := st.DeleteRecord(ctx, uri)
		require.ErrorIs(t, err, store.ErrNotFound)
	})
}

func (s *RecordStoreSuite) testList(t *testing.T) {
	ctx := context.Background()

	t.Run("by schema", func(t *testing.T) {
		st := s.newStore()

		for _, id := range []string{"p1", "p2", "p3"} {
			r := core.NewRecord("com.example", "posts", id)
			require.NoError(t, st.CreateRecord(ctx, r))
		}
		// Different schema — should not appear.
		other := core.NewRecord("com.example", "users", "u1")
		require.NoError(t, st.CreateRecord(ctx, other))

		uri := core.MustParseURI("xdb://com.example/posts")
		page, err := st.ListRecords(ctx, uri, nil)
		require.NoError(t, err)
		assert.Equal(t, 3, page.Total)
		assert.Len(t, page.Items, 3)
	})

	t.Run("by namespace", func(t *testing.T) {
		st := s.newStore()

		require.NoError(t, st.CreateRecord(ctx, core.NewRecord("com.example", "posts", "p1")))
		require.NoError(t, st.CreateRecord(ctx, core.NewRecord("com.example", "users", "u1")))
		require.NoError(t, st.CreateRecord(ctx, core.NewRecord("com.other", "posts", "p1")))

		uri := core.MustParseURI("xdb://com.example")
		page, err := st.ListRecords(ctx, uri, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, page.Total)
	})

	t.Run("pagination", func(t *testing.T) {
		st := s.newStore()

		for i := range 5 {
			r := core.NewRecord("com.example", "posts", string(rune('a'+i)))
			require.NoError(t, st.CreateRecord(ctx, r))
		}

		uri := core.MustParseURI("xdb://com.example/posts")

		tests := []struct {
			query      *store.ListQuery
			name       string
			wantLen    int
			wantTotal  int
			wantOffset int
		}{
			{
				name:       "first page",
				query:      &store.ListQuery{Limit: 2, Offset: 0},
				wantLen:    2,
				wantTotal:  5,
				wantOffset: 2,
			},
			{
				name:       "middle page",
				query:      &store.ListQuery{Limit: 2, Offset: 2},
				wantLen:    2,
				wantTotal:  5,
				wantOffset: 4,
			},
			{
				name:       "last page",
				query:      &store.ListQuery{Limit: 2, Offset: 4},
				wantLen:    1,
				wantTotal:  5,
				wantOffset: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				page, err := st.ListRecords(ctx, uri, tt.query)
				require.NoError(t, err)
				assert.Len(t, page.Items, tt.wantLen)
				assert.Equal(t, tt.wantTotal, page.Total)
				assert.Equal(t, tt.wantOffset, page.NextOffset)
			})
		}
	})
}
