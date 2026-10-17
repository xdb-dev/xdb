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

func TestListRecords_OrderBy(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/ranked")

	tests := []struct {
		mode     schema.Mode
		pushdown bool
	}{
		{mode: schema.ModeStrict, pushdown: true},
		{mode: schema.ModeDynamic, pushdown: true},
		{mode: schema.ModeFlexible},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			d := newTestDriver(t)
			st := store.New(d)

			require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
				URI:  uri,
				Mode: tt.mode,
				Fields: map[string]schema.Field{
					"title": {Type: core.TypeString},
					"rank":  {Type: core.TypeInt},
				},
			}))
			storetest.SeedRanked(t, st, "test", "ranked")

			for _, c := range storetest.RankedOrderCases() {
				q := &store.Query{
					URI:     uri,
					OrderBy: c.OrderBy,
					Limit:   c.Limit,
					Offset:  c.Offset,
				}

				page, err := st.ListRecords(ctx, q)
				require.NoError(t, err, c.Name)
				assert.Equal(t, c.Want, storetest.RecordIDs(page.Items), c.Name)

				_, err = d.QueryTuples(ctx, q)
				if tt.pushdown {
					require.NoError(t, err, c.Name)
				} else {
					require.ErrorIs(t, err, store.ErrUnsupportedOrder, c.Name)
				}
			}
		})
	}
}

func TestListRecords_OrderByWithFilterOnKV(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/ranked")
	d := newTestDriver(t)
	st := store.New(d)

	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeFlexible,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}))
	storetest.SeedRanked(t, st, "test", "ranked")

	q := &store.Query{
		URI:     uri,
		Filter:  `title == "x"`,
		OrderBy: []string{"-rank"},
		Limit:   2,
	}

	page, err := st.ListRecords(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "c"}, storetest.RecordIDs(page.Items))
	assert.Equal(t, 3, page.Total)

	unordered := &store.Query{
		URI:    uri,
		Filter: q.Filter,
	}

	_, err = d.QueryTuples(ctx, unordered)
	require.NoError(t, err, "the KV engine pushes the filter down when the order is gone")
}

func TestListRecords_OrderByTimeOnColumnTable(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://test/events")
	d := newTestDriver(t)
	st := store.New(d)

	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"start": {Type: core.TypeTime},
		},
	}))
	storetest.SeedTimed(t, st, "test", "events")

	for _, c := range storetest.TimedOrderCases() {
		q := &store.Query{
			URI:     uri,
			OrderBy: c.OrderBy,
		}

		page, err := st.ListRecords(ctx, q)
		require.NoError(t, err, c.Name)
		assert.Equal(t, c.Want, storetest.RecordIDs(page.Items), c.Name)

		_, err = d.QueryTuples(ctx, q)
		require.NoError(t, err, "%s: the column table pushes the order down", c.Name)
	}
}
