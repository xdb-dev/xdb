package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/schema"
	"zombiezen.com/go/sqlite"
)

func TestMigration(t *testing.T) {
	db, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	require.NoError(t, err)

	defer db.Close()

	s := &schema.Schema{
		Records: []schema.Record{
			{
				Kind:  "User",
				Table: "users",
				Attributes: []schema.Attribute{
					{Name: "name", Type: schema.String},
					{Name: "birth_date", Type: schema.Time},
					{Name: "tags", Type: schema.String, Repeated: true},
					{Name: "is_active", Type: schema.Bool},
					{Name: "score", Type: schema.Float},
				},
			},
		},
	}

	t.Run("Generate", func(t *testing.T) {
		expected := []string{
			`CREATE TABLE IF NOT EXISTS users (id VARCHAR PRIMARY KEY, name VARCHAR, birth_date TIMESTAMP, tags VARCHAR[], is_active BOOLEAN, score REAL);`,
		}

		m := NewMigration(db, s)

		queries, err := m.Generate(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, expected, queries)

		err = m.Run(context.Background())
		require.NoError(t, err)
	})

	t.Run("Generate alter table", func(t *testing.T) {
		s.Records[0].Attributes = append(
			s.Records[0].Attributes,
			schema.Attribute{Name: "age", Type: schema.Int},
			schema.Attribute{Name: "is_admin", Type: schema.Bool},
		)

		expected := []string{
			`ALTER TABLE users ADD COLUMN age INTEGER, ADD COLUMN is_admin BOOLEAN;`,
		}

		m := NewMigration(db, s)

		queries, err := m.Generate(context.Background())
		require.NoError(t, err)
		assert.EqualValues(t, expected, queries)

		err = m.Run(context.Background())
		require.NoError(t, err)
	})
}
