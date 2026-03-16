package sql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestCreateKVTable(t *testing.T) {
	db, q := testDB(t)
	ctx := context.Background()

	err := q.CreateKVTable(ctx, xsql.CreateKVTableParams{Table: `"kv:test/t"`})
	require.NoError(t, err)

	// Verify by inserting a row.
	_, err = db.ExecContext(ctx,
		`INSERT INTO "kv:test/t" (_id, _attr, _type, _val) VALUES (?, ?, ?, ?)`,
		"id1", "name", "STRING", []byte(`"hello"`),
	)
	require.NoError(t, err)
}

func TestCreateTable(t *testing.T) {
	t.Run("basic with columns", func(t *testing.T) {
		db, q := testDB(t)
		ctx := context.Background()

		err := q.CreateTable(ctx, xsql.CreateTableParams{
			Table: `"t:test/t"`,
			Columns: []xsql.Column{
				{Name: "title", Type: "TEXT"},
				{Name: "count", Type: "INTEGER"},
			},
		})
		require.NoError(t, err)

		// Verify by inserting and selecting.
		_, err = db.ExecContext(ctx,
			`INSERT INTO "t:test/t" (_id, title, count) VALUES (?, ?, ?)`,
			"id1", "hello", 42,
		)
		require.NoError(t, err)

		var title string
		var count int
		err = db.QueryRowContext(ctx,
			`SELECT title, count FROM "t:test/t" WHERE _id = ?`, "id1",
		).Scan(&title, &count)
		require.NoError(t, err)
		assert.Equal(t, "hello", title)
		assert.Equal(t, 42, count)
	})

	t.Run("NOT NULL constraint", func(t *testing.T) {
		db, q := testDB(t)
		ctx := context.Background()

		err := q.CreateTable(ctx, xsql.CreateTableParams{
			Table: `"t:test/nn"`,
			Columns: []xsql.Column{
				{Name: "name", Type: "TEXT", NotNull: true},
			},
		})
		require.NoError(t, err)

		// Insert with NULL should fail.
		_, err = db.ExecContext(ctx,
			`INSERT INTO "t:test/nn" (_id, name) VALUES (?, NULL)`, "id1",
		)
		assert.Error(t, err)
	})

	t.Run("is idempotent", func(t *testing.T) {
		_, q := testDB(t)
		ctx := context.Background()

		params := xsql.CreateTableParams{
			Table:   `"t:test/idem"`,
			Columns: []xsql.Column{{Name: "x", Type: "TEXT"}},
		}
		require.NoError(t, q.CreateTable(ctx, params))
		require.NoError(t, q.CreateTable(ctx, params))
	})
}

func TestAddColumn(t *testing.T) {
	db, q := testDB(t)
	ctx := context.Background()

	// Create table first.
	require.NoError(t, q.CreateTable(ctx, xsql.CreateTableParams{
		Table:   `"t:test/ac"`,
		Columns: []xsql.Column{{Name: "a", Type: "TEXT"}},
	}))

	// Add column.
	err := q.AddColumn(ctx, xsql.AddColumnParams{
		Table:  `"t:test/ac"`,
		Column: xsql.Column{Name: "b", Type: "INTEGER"},
	})
	require.NoError(t, err)

	// Verify new column works.
	_, err = db.ExecContext(ctx,
		`INSERT INTO "t:test/ac" (_id, a, b) VALUES (?, ?, ?)`, "id1", "x", 5,
	)
	require.NoError(t, err)
}

func TestCreateIndex(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	require.NoError(t, q.CreateTable(ctx, xsql.CreateTableParams{
		Table:   `"t:test/idx"`,
		Columns: []xsql.Column{{Name: "name", Type: "TEXT"}},
	}))

	err := q.CreateIndex(ctx, xsql.CreateIndexParams{
		Table: `"t:test/idx"`,
		Index: xsql.Index{Name: "idx_name", Columns: []string{"name"}},
	})
	require.NoError(t, err)

	// Idempotent.
	err = q.CreateIndex(ctx, xsql.CreateIndexParams{
		Table: `"t:test/idx"`,
		Index: xsql.Index{Name: "idx_name", Columns: []string{"name"}},
	})
	require.NoError(t, err)
}

func TestDropIndex(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	// Drop nonexistent — no error (IF EXISTS).
	err := q.DropIndex(ctx, xsql.DropIndexParams{Index: "nonexistent"})
	require.NoError(t, err)
}

func TestDropTable(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	require.NoError(t, q.CreateTable(ctx, xsql.CreateTableParams{
		Table:   `"t:test/drop"`,
		Columns: []xsql.Column{{Name: "x", Type: "TEXT"}},
	}))

	err := q.DropTable(ctx, xsql.DropTableParams{Table: `"t:test/drop"`})
	require.NoError(t, err)

	// Drop again — no error (IF EXISTS).
	err = q.DropTable(ctx, xsql.DropTableParams{Table: `"t:test/drop"`})
	require.NoError(t, err)
}
