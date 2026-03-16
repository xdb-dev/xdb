package sql_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

func TestPutSchema(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	err := q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1",
		Schema:    "posts",
		Data:      json.RawMessage(`{"mode":"strict"}`),
	})
	require.NoError(t, err)

	// Verify via GetSchema.
	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"mode":"strict"}`, string(data))
}

func TestPutSchema_Overwrite(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1", Schema: "posts", Data: json.RawMessage(`{"v":1}`),
	}))
	require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1", Schema: "posts", Data: json.RawMessage(`{"v":2}`),
	}))

	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"v":2}`, string(data))
}

func TestGetSchema_Missing(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{Namespace: "nope", Schema: "nope"})
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestListSchemas(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	// Insert schemas across namespaces.
	for _, s := range []struct{ ns, name string }{
		{"ns1", "a"}, {"ns1", "b"}, {"ns2", "c"},
	} {
		require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
			Namespace: s.ns, Schema: s.name, Data: json.RawMessage(`{}`),
		}))
	}

	t.Run("all", func(t *testing.T) {
		schemas, err := q.ListSchemas(ctx, xsql.ListSchemasParams{Limit: 100})
		require.NoError(t, err)
		require.Len(t, schemas, 3)
		assert.Equal(t, "a", schemas[0].Schema)
		assert.Equal(t, "b", schemas[1].Schema)
		assert.Equal(t, "c", schemas[2].Schema)
	})

	t.Run("filtered by namespace", func(t *testing.T) {
		ns := "ns1"
		schemas, err := q.ListSchemas(ctx, xsql.ListSchemasParams{Namespace: &ns, Limit: 100})
		require.NoError(t, err)
		require.Len(t, schemas, 2)
	})

	t.Run("paginated", func(t *testing.T) {
		schemas, err := q.ListSchemas(ctx, xsql.ListSchemasParams{Limit: 2, Offset: 0})
		require.NoError(t, err)
		require.Len(t, schemas, 2)

		schemas, err = q.ListSchemas(ctx, xsql.ListSchemasParams{Limit: 2, Offset: 2})
		require.NoError(t, err)
		require.Len(t, schemas, 1)
	})
}

func TestDeleteSchema(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1", Schema: "posts", Data: json.RawMessage(`{}`),
	}))

	err := q.DeleteSchema(ctx, xsql.DeleteSchemaParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)

	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestSchemaExists(t *testing.T) {
	_, q := testDB(t)
	ctx := context.Background()

	exists, err := q.SchemaExists(ctx, xsql.SchemaExistsParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: "ns1", Schema: "posts", Data: json.RawMessage(`{}`),
	}))

	exists, err = q.SchemaExists(ctx, xsql.SchemaExistsParams{Namespace: "ns1", Schema: "posts"})
	require.NoError(t, err)
	assert.True(t, exists)
}
