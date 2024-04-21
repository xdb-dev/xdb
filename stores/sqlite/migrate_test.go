package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/schema"
	"zombiezen.com/go/sqlite"
)

func TestMigration_Generate(t *testing.T) {
	db, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	require.NoError(t, err)

	defer db.Close()

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

	expected := []string{
		`CREATE TABLE IF NOT EXISTS users (name STRING);`,
	}

	m := NewMigration(db, schema)

	queries, err := m.Generate(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, expected, queries)
}
