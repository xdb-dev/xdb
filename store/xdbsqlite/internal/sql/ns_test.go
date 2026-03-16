package sql_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestNamespaceExists(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	exists, err := q.NamespaceExists(ctx, xsql.NamespaceExistsParams{Namespace: "ns1"})
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1", Schema: "s1", Data: json.RawMessage(`{}`),
	}))

	exists, err = q.NamespaceExists(ctx, xsql.NamespaceExistsParams{Namespace: "ns1"})
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestListNamespaces(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	// Insert schemas in multiple namespaces.
	for _, s := range []struct{ ns, name string }{
		{"beta", "s1"}, {"alpha", "s1"}, {"alpha", "s2"}, {"gamma", "s1"},
	} {
		require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
			Namespace: s.ns, Schema: s.name, Data: json.RawMessage(`{}`),
		}))
	}

	t.Run("distinct and ordered", func(t *testing.T) {
		ns, err := q.ListNamespaces(ctx, xsql.ListNamespacesParams{Limit: 100})
		require.NoError(t, err)
		require.Equal(t, []string{"alpha", "beta", "gamma"}, ns)
	})

	t.Run("paginated", func(t *testing.T) {
		ns, err := q.ListNamespaces(ctx, xsql.ListNamespacesParams{Limit: 2, Offset: 0})
		require.NoError(t, err)
		require.Equal(t, []string{"alpha", "beta"}, ns)

		ns, err = q.ListNamespaces(ctx, xsql.ListNamespacesParams{Limit: 2, Offset: 2})
		require.NoError(t, err)
		require.Equal(t, []string{"gamma"}, ns)
	})
}
