package sql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestListTables(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	require.NoError(t, q.CreateKVTable(ctx, xsql.CreateKVTableParams{
		Table: `"kv:com.example/posts"`,
	}))
	require.NoError(t, q.CreateKVTable(ctx, xsql.CreateKVTableParams{
		Table: `"kv:com.example/users"`,
	}))
	require.NoError(t, q.CreateKVTable(ctx, xsql.CreateKVTableParams{
		Table: `"kv:com.other/posts"`,
	}))

	t.Run("matches by prefix pattern", func(t *testing.T) {
		names, err := q.ListTables(ctx, xsql.ListTablesParams{
			Pattern: "kv:com.example/*",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{
			"kv:com.example/posts",
			"kv:com.example/users",
		}, names)
	})

	t.Run("underscore matches literally", func(t *testing.T) {
		require.NoError(t, q.CreateKVTable(ctx, xsql.CreateKVTableParams{
			Table: `"kv:com.example/user_events"`,
		}))

		names, err := q.ListTables(ctx, xsql.ListTablesParams{
			Pattern: "kv:com.example/user_events",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"kv:com.example/user_events"}, names)
	})

	t.Run("no matches yields empty", func(t *testing.T) {
		names, err := q.ListTables(ctx, xsql.ListTablesParams{
			Pattern: "kv:com.missing/*",
		})
		require.NoError(t, err)
		assert.Empty(t, names)
	})
}
