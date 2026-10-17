package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/storetest"
)

func TestListRecords_RejectsABadOrder(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, s.CreateSchema(ctx, uri, strictDef(uri)))

	tests := []struct {
		name    string
		orderBy []string
		want    string
	}{
		{name: "unknown field", orderBy: []string{"sort"}, want: `"sort"`},
		{name: "unknown descending field", orderBy: []string{"-sort"}, want: `"sort"`},
		{name: "lists every reserved field", orderBy: []string{"sort"}, want: "_id, _updated, _version, note, title"},
		{name: "empty name", orderBy: []string{"-"}, want: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.ListRecords(ctx, &store.Query{
				URI:     uri,
				OrderBy: tt.orderBy,
			})
			require.ErrorIs(t, err, core.ErrInvalidOrder)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestListRecords_OrdersByAReservedField(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	for _, id := range []string{"a", "b"} {
		r := core.NewRecord("com.example", "posts", id)
		r.Set("title", id)
		require.NoError(t, s.CreateRecord(ctx, r))
	}

	page, err := s.ListRecords(ctx, &store.Query{
		URI:     uri,
		OrderBy: []string{"-_id", "_updated"},
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "b", page.Items[0].URI().ID())
}

func TestListRecords_OrdersByAnUndeclaredField(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://com.example/posts")

	tests := []struct {
		name string
		mode schema.Mode
	}{
		{name: "flexible", mode: schema.ModeFlexible},
		{name: "dynamic", mode: schema.ModeDynamic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New(xdbmemory.NewDriver())
			def := testDef(uri)
			def.Mode = tt.mode
			require.NoError(t, s.CreateSchema(ctx, uri, def))

			for id, priority := range map[string]int64{"a": 2, "b": 1} {
				r := core.NewRecord("com.example", "posts", id)
				r.Set("title", id)
				r.Set("priority", priority)
				require.NoError(t, s.CreateRecord(ctx, r))
			}

			page, err := s.ListRecords(ctx, &store.Query{
				URI:     uri,
				OrderBy: []string{"priority"},
			})
			require.NoError(t, err)
			assert.Equal(t, []string{"b", "a"}, storetest.RecordIDs(page.Items))
		})
	}
}

func TestListRecords_OrdersTimesByMillisecond(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	storetest.SeedTimed(t, s, "com.example", "events")

	for _, tt := range storetest.TimedOrderCases() {
		t.Run(tt.Name, func(t *testing.T) {
			page, err := s.ListRecords(ctx, &store.Query{
				URI:     core.MustParseURI("xdb://com.example/events"),
				OrderBy: tt.OrderBy,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.Want, storetest.RecordIDs(page.Items))
		})
	}
}
