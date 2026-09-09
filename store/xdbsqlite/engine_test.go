package xdbsqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func newDriver(t *testing.T) *Driver {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	d, err := NewDriver(db)
	require.NoError(t, err)
	return d
}

func countScan(t *testing.T, d *Driver, scope *core.URI) int {
	t.Helper()
	n := 0
	for _, err := range d.ScanTuples(context.Background(), scope) {
		require.NoError(t, err)
		n++
	}
	return n
}

// TestDropRecordsThenApply pins the ensure-on-write fix: after
// DropRecords removes a strict schema's column table, the next Apply
// must recreate it from the def rather than fail with "no such table".
func TestDropRecordsThenApply(t *testing.T) {
	ctx := context.Background()
	d := newDriver(t)

	uri := core.MustParseURI("xdb://app/posts")
	require.NoError(t, d.CreateSchema(ctx, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}))

	put := func(id, title string) store.Mutation {
		return store.Mutation{
			Path:   core.MustNewURI("app", "posts", id),
			Op:     store.OpPut,
			Tuples: []*core.Tuple{core.NewTuple("app/posts/"+id, "title", title)},
		}
	}

	require.NoError(t, d.Apply(ctx, put("p1", "first")))
	require.NoError(t, d.DropRecords(ctx, uri))
	assert.Equal(t, 0, countScan(t, d, uri), "records gone after DropRecords")

	// The column table was dropped; this write must recreate it.
	require.NoError(t, d.Apply(ctx, put("p2", "second")))
	assert.Equal(t, 1, countScan(t, d, uri))
}

// TestScanPagingAcrossBatch shrinks the scan batch so a scan must page,
// proving records past the first page are yielded — for both engines,
// whose paging loops are structurally identical.
func TestScanPagingAcrossBatch(t *testing.T) {
	tests := []struct {
		name string
		mode schema.Mode
		def  *schema.Def
	}{
		{"kv engine", schema.ModeFlexible, nil},
		{
			"table engine",
			schema.ModeStrict,
			&schema.Def{Fields: map[string]schema.Field{"body": {Type: core.TypeString}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orig := recordScanBatch
			recordScanBatch = 2
			t.Cleanup(func() { recordScanBatch = orig })

			ctx := context.Background()
			d := newDriver(t)

			uri := core.MustParseURI("xdb://app/notes")
			def := &schema.Def{URI: uri, Mode: tc.mode}
			if tc.def != nil {
				def.Fields = tc.def.Fields
			}
			require.NoError(t, d.CreateSchema(ctx, def))

			for _, id := range []string{"a", "b", "c", "d", "e"} {
				require.NoError(t, d.Apply(ctx, store.Mutation{
					Path:   core.MustNewURI("app", "notes", id),
					Op:     store.OpPut,
					Tuples: []*core.Tuple{core.NewTuple("app/notes/"+id, "body", "x")},
				}))
			}

			assert.Equal(t, 5, countScan(t, d, uri), "all records yielded across batch boundary")
		})
	}
}
