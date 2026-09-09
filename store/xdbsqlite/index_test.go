package xdbsqlite_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
)

// newTestStoreWithDB builds a store over an isolated on-disk database and
// returns the raw *sql.DB so a test can inspect the physical schema.
func newTestStoreWithDB(t *testing.T) (store.Store, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	d, err := xdbsqlite.NewDriver(db)
	require.NoError(t, err)
	return store.New(d), db
}

// indexDDL returns the CREATE INDEX statements for a column table from
// sqlite_master.
func indexDDL(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()

	rows, err := db.QueryContext(
		t.Context(),
		`SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql IS NOT NULL`,
		table,
	)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())
	return out
}

func hasIndexOn(ddls []string, column string, unique bool) bool {
	for _, ddl := range ddls {
		if !strings.Contains(ddl, "("+column+")") && !strings.Contains(ddl, " "+column+" ") {
			continue
		}
		isUnique := strings.Contains(strings.ToUpper(ddl), "UNIQUE INDEX")
		if isUnique == unique {
			return true
		}
	}
	return false
}

func TestIndexedFields_CreateMaterializesIndexes(t *testing.T) {
	ctx := context.Background()
	st, db := newTestStoreWithDB(t)

	uri := core.MustParseURI("xdb://test/users")
	def := &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"email":  {Type: core.TypeString, Unique: true},
			"status": {Type: core.TypeString, Indexed: true},
			"name":   {Type: core.TypeString},
		},
	}
	require.NoError(t, st.CreateSchema(ctx, uri, def))

	ddls := indexDDL(t, db, "t:test/users")
	assert.True(t, hasIndexOn(ddls, "email", true), "unique index on email: %v", ddls)
	assert.True(t, hasIndexOn(ddls, "status", false), "index on status: %v", ddls)
	assert.False(t, hasIndexOn(ddls, "name", false), "no index on name: %v", ddls)
	assert.False(t, hasIndexOn(ddls, "name", true), "no unique index on name: %v", ddls)
}

func TestUniqueField_DuplicateWriteConflicts(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	uri := core.MustParseURI("xdb://test/accounts")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"email": {Type: core.TypeString, Unique: true},
		},
	}))

	r1 := core.NewRecord("test", "accounts", "a1")
	r1.Set("email", "dup@example.com")
	require.NoError(t, st.CreateRecord(ctx, r1))

	r2 := core.NewRecord("test", "accounts", "a2")
	r2.Set("email", "dup@example.com")
	err := st.CreateRecord(ctx, r2)
	require.Error(t, err)
	require.ErrorIs(t, err, core.ErrUniqueViolation)
	assert.Contains(t, err.Error(), "email")

	// A distinct value still succeeds.
	r3 := core.NewRecord("test", "accounts", "a3")
	r3.Set("email", "fresh@example.com")
	require.NoError(t, st.CreateRecord(ctx, r3))
}

func TestUpdateSchema_DropsIndexedField(t *testing.T) {
	ctx := context.Background()
	st, db := newTestStoreWithDB(t)

	uri := core.MustParseURI("xdb://test/items")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"code":  {Type: core.TypeString, Indexed: true},
		},
	}))

	r := core.NewRecord("test", "items", "i1")
	r.Set("title", "hello")
	r.Set("code", "abc")
	require.NoError(t, st.CreateRecord(ctx, r))

	// Removing an indexed field must drop the index before the column,
	// or SQLite refuses the DROP COLUMN.
	require.NoError(t, st.UpdateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}))

	ddls := indexDDL(t, db, "t:test/items")
	assert.False(t, hasIndexOn(ddls, "code", false), "index on code should be gone: %v", ddls)
}

func TestUpdateSchema_AddsIndexedField(t *testing.T) {
	ctx := context.Background()
	st, db := newTestStoreWithDB(t)

	uri := core.MustParseURI("xdb://test/things")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}))

	require.NoError(t, st.UpdateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
			"sku":   {Type: core.TypeString, Unique: true},
		},
	}))

	ddls := indexDDL(t, db, "t:test/things")
	assert.True(t, hasIndexOn(ddls, "sku", true), "unique index on sku: %v", ddls)
}

func TestIndexedFields_StaleIndexDoesNotBlockDropColumn(t *testing.T) {
	ctx := context.Background()
	st, db := newTestStoreWithDB(t)

	uri := core.MustParseURI("xdb://test/members")

	// A unique field materializes a UNIQUE index on the column table.
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"email": {Type: core.TypeString, Unique: true},
			"name":  {Type: core.TypeString},
		},
	}))

	// A delete without cascade leaves the table and its indexes behind.
	require.NoError(t, st.DeleteSchema(ctx, uri))

	// Re-creating the schema without the marker no-ops on the table, so
	// the index from the previous generation survives in sqlite_master
	// while the new definition knows nothing about it.
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"email": {Type: core.TypeString},
			"name":  {Type: core.TypeString},
		},
	}))

	// Removing the field must drop that stale index too. SQLite refuses
	// to drop a column an index still references.
	err := st.UpdateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString},
		},
	})
	require.NoError(t, err, "a stale index must not block dropping the column")

	ddls := indexDDL(t, db, "t:test/members")
	assert.False(t, hasIndexOn(ddls, "email", true), "the stale index is gone")
}
