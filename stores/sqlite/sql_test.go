package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb"
	"github.com/xdb-dev/xdb/schema"
	"zombiezen.com/go/sqlite"
)

func TestSQLiteStore(t *testing.T) {
	schema := &schema.Schema{
		Records: []schema.Record{
			{
				Kind:  "User",
				Table: "users",
				Attributes: []schema.Attribute{
					{Name: "name", Type: schema.String},
				},
			},
		},
	}

	userKey := xdb.NewKey("User", "1")
	userRecord := xdb.NewRecord(userKey,
		xdb.NewTuple(userKey, "name", "John Doe"),
	)

	db, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	require.NoError(t, err)

	defer db.Close()

	m := NewMigration(db, schema)

	err = m.Run(context.Background())
	require.NoError(t, err)

	store := NewSQLiteStore(db, schema)

	t.Run("PutRecord", func(t *testing.T) {
		err := store.PutRecord(context.Background(), userRecord)
		require.NoError(t, err)
	})

	t.Run("GetRecord", func(t *testing.T) {
		got, err := store.GetRecord(context.Background(), userKey)
		require.NoError(t, err)
		require.EqualValues(t, userRecord.Key(), got.Key())
	})

	t.Run("DeleteRecord", func(t *testing.T) {
		err := store.DeleteRecord(context.Background(), userKey)
		require.NoError(t, err)
	})
}
