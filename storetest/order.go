package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// OrderCase is one ordered list over the records of [SeedRanked].
type OrderCase struct {
	Name    string
	OrderBy []string
	Want    []string
	Limit   int
	Offset  int
}

// SeedRanked creates five records in ns/table. Records b and e have no
// rank, so they show where a missing value sorts.
//
//	id  rank  title
//	a   3     x
//	b   -     y
//	c   1     x
//	d   2     y
//	e   -     x
func SeedRanked(t *testing.T, st store.RecordStore, ns, table string) {
	t.Helper()

	rows := []struct {
		id    string
		title string
		rank  int64
	}{
		{id: "a", title: "x", rank: 3},
		{id: "b", title: "y"},
		{id: "c", title: "x", rank: 1},
		{id: "d", title: "y", rank: 2},
		{id: "e", title: "x"},
	}

	for _, row := range rows {
		r := core.NewRecord(ns, table, row.id)
		r.Set("title", row.title)
		if row.rank != 0 {
			r.Set("rank", row.rank)
		}

		require.NoError(t, st.CreateRecord(context.Background(), r))
	}
}

// RankedOrderCases returns the orders that every store must agree on for
// the records of [SeedRanked]. A missing value sorts first ascending and
// last descending, and the record ID breaks a tie.
func RankedOrderCases() []OrderCase {
	return []OrderCase{
		{
			Name:    "ascending puts missing first",
			OrderBy: []string{"rank"},
			Want:    []string{"b", "e", "c", "d", "a"},
		},
		{
			Name:    "descending puts missing last",
			OrderBy: []string{"-rank"},
			Want:    []string{"a", "d", "c", "b", "e"},
		},
		{
			Name:    "second key breaks a tie",
			OrderBy: []string{"title", "-rank"},
			Want:    []string{"a", "c", "e", "d", "b"},
		},
		{
			Name:    "order applies before the page",
			OrderBy: []string{"-rank"},
			Limit:   2,
			Offset:  2,
			Want:    []string{"c", "b"},
		},
		{
			Name:    "descending id",
			OrderBy: []string{"-_id"},
			Want:    []string{"e", "d", "c", "b", "a"},
		},
	}
}

// SeedTimed creates three records in ns/table. Records a and b start in
// the same millisecond, with b 0.4 ms before a. Record c starts 1 ms
// before both.
//
//	id  start
//	a   12:00:00.1005
//	b   12:00:00.1001
//	c   12:00:00.0995
func SeedTimed(t *testing.T, st store.RecordStore, ns, table string) {
	t.Helper()

	base := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

	rows := []struct {
		id    string
		start time.Duration
	}{
		{id: "a", start: 100500 * time.Microsecond},
		{id: "b", start: 100100 * time.Microsecond},
		{id: "c", start: 99500 * time.Microsecond},
	}

	for _, row := range rows {
		r := core.NewRecord(ns, table, row.id)
		r.Set("start", base.Add(row.start))

		require.NoError(t, st.CreateRecord(context.Background(), r))
	}
}

// TimedOrderCases returns the orders that every store must agree on for
// the records of [SeedTimed]. A store compares times in milliseconds,
// because SQLite stores milliseconds. Records a and b then tie, and the
// record ID breaks the tie.
func TimedOrderCases() []OrderCase {
	return []OrderCase{
		{
			Name:    "ascending ties in one millisecond",
			OrderBy: []string{"start"},
			Want:    []string{"c", "a", "b"},
		},
		{
			Name:    "descending ties in one millisecond",
			OrderBy: []string{"-start"},
			Want:    []string{"a", "b", "c"},
		},
	}
}

// RecordIDs returns the IDs of records in order.
func RecordIDs(records []*core.Record) []string {
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = r.URI().ID()
	}

	return ids
}
