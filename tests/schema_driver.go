package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
)

type schemaReaderWriter interface {
	driver.SchemaReader
	driver.SchemaWriter
}

// TestSchemaReaderWriter runs a comprehensive test suite for SchemaDriver implementations.
func TestSchemaReaderWriter(t *testing.T, rw schemaReaderWriter) {
	t.Helper()

	ctx := context.Background()

	t.Run("PutSchema", func(t *testing.T) {
		schemaDef := FakePostSchema()
		err := rw.PutSchema(ctx, schemaDef)
		require.NoError(t, err)

		testSchemaDef := FakeTestSchema()
		err = rw.PutSchema(ctx, testSchemaDef)
		require.NoError(t, err)
	})

	t.Run("GetSchema", func(t *testing.T) {
		uri, err := core.ParseURI("xdb://default/posts")
		require.NoError(t, err)

		got, err := rw.GetSchema(ctx, uri)
		require.NoError(t, err)
		AssertDefEqual(t, FakePostSchema(), got)
	})

	t.Run("GetSchemaNotFound", func(t *testing.T) {
		uri, err := core.ParseURI("xdb://default/nonexistent")
		require.NoError(t, err)

		got, err := rw.GetSchema(ctx, uri)
		require.Error(t, err)
		require.ErrorIs(t, err, driver.ErrNotFound)
		require.Nil(t, got)
	})

	t.Run("ListSchemas", func(t *testing.T) {
		ns, err := core.ParseNS("default")
		require.NoError(t, err)

		schemas, err := rw.ListSchemas(ctx, ns)
		require.NoError(t, err)
		require.Len(t, schemas, 2)

		schemaMap := make(map[string]*schema.Def)
		for _, s := range schemas {
			schemaMap[s.Name] = s
		}

		require.Contains(t, schemaMap, "posts")
		require.Contains(t, schemaMap, "all_types")

		AssertDefEqual(t, FakePostSchema(), schemaMap["posts"])
		AssertDefEqual(t, FakeTestSchema(), schemaMap["all_types"])
	})

	t.Run("DeleteSchema", func(t *testing.T) {
		uri, err := core.ParseURI("xdb://default/posts")
		require.NoError(t, err)

		err = rw.DeleteSchema(ctx, uri)
		require.NoError(t, err)

		got, err := rw.GetSchema(ctx, uri)
		require.Error(t, err)
		require.ErrorIs(t, err, driver.ErrNotFound)
		require.Nil(t, got)
	})

	t.Run("DeleteSchemaNotFound", func(t *testing.T) {
		uri, err := core.ParseURI("xdb://default/nonexistent")
		require.NoError(t, err)

		err = rw.DeleteSchema(ctx, uri)
		require.NoError(t, err)
	})

	t.Run("UpdateSchema", func(t *testing.T) {
		schemaDef := FakePostSchema()
		err := rw.PutSchema(ctx, schemaDef)
		require.NoError(t, err)

		updatedSchema := &schema.Def{
			Name:        "posts",
			Description: "Updated blog post schema",
			Version:     "2.0.0",
			Fields: []*schema.FieldDef{
				{Name: "title", Type: core.TypeString},
				{Name: "content", Type: core.TypeString},
				{Name: "author", Type: core.TypeString},
			},
			Required: []string{"title", "content", "author"},
		}

		err = rw.PutSchema(ctx, updatedSchema)
		require.NoError(t, err)

		uri, err := core.ParseURI("xdb://default/posts")
		require.NoError(t, err)

		got, err := rw.GetSchema(ctx, uri)
		require.NoError(t, err)
		AssertDefEqual(t, updatedSchema, got)
	})
}
