package xdbsqlite_test

// import (
// 	"bytes"
// 	"context"
// 	"database/sql"
// 	"strings"
// 	"testing"

// 	_ "github.com/mattn/go-sqlite3"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/xdb-dev/xdb/core"
// 	"github.com/xdb-dev/xdb/driver/xdbsqlite"
// 	"github.com/xdb-dev/xdb/tests"
// )

// func TestGenerateMigrations(t *testing.T) {
// 	t.Parallel()

// 	db, err := sql.Open("sqlite3", ":memory:")
// 	require.NoError(t, err)

// 	defer db.Close()

// 	migrator := xdbsqlite.NewMigrator(db)

// 	schemas := []*core.Schema{
// 		tests.FakePostSchema(),
// 	}

// 	t.Run("Generate Create Table", func(t *testing.T) {
// 		buf := bytes.NewBuffer(nil)
// 		err := migrator.GenerateMigrations(context.Background(), schemas, buf)
// 		require.NoError(t, err)

// 		expected := normalize(`CREATE TABLE IF NOT EXISTS "Post" (
// 			"title" TEXT,
// 			"content" TEXT,
// 			"tags" TEXT,
// 			"rating" REAL,
// 			"published" INTEGER,
// 			"comments.count" INTEGER,
// 			"views.count" INTEGER,
// 			"likes.count" INTEGER,
// 			"shares.count" INTEGER,
// 			"favorites.count" INTEGER,
// 			"author.id" TEXT,
// 			"author.name" TEXT
// 		);`)
// 		got := normalize(buf.String())

// 		assert.Equal(t, expected, got)

// 		_, err = db.Exec(buf.String())
// 		require.NoError(t, err)
// 	})

// 	t.Run("Generate Alter Table", func(t *testing.T) {
// 		buf := bytes.NewBuffer(nil)

// 		schemas[0].Attributes = append(schemas[0].Attributes,
// 			core.Attribute{
// 				Name: "created_at",
// 				Type: core.NewType(core.TypeIDTime),
// 			},
// 		)

// 		err := migrator.GenerateMigrations(context.Background(), schemas, buf)
// 		require.NoError(t, err)

// 		expected := normalize(`ALTER TABLE "Post" ADD COLUMN "created_at" INTEGER;`)
// 		got := normalize(buf.String())

// 		assert.Equal(t, expected, got)

// 		_, err = db.Exec(buf.String())
// 		require.NoError(t, err)
// 	})
// }

// func normalize(s string) string {
// 	s = strings.ReplaceAll(s, "\t", "  ")
// 	s = strings.ReplaceAll(s, "\n", "")
// 	s = strings.ReplaceAll(s, " ", "")

// 	return s
// }
