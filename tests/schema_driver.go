package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
)

// TestSchemaReaderWriter runs a comprehensive test suite for SchemaDriver implementations.
func TestSchemaReaderWriter(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()

	t.Run("PutSchema creates a new schema", func(t *testing.T) {
		testPutSchemaCreatesNew(t, rw)
	})

	t.Run("PutSchema updates an existing schema", func(t *testing.T) {
		testPutSchemaUpdatesExisting(t, rw)
	})

	t.Run("GetSchema retrieves an existing schema", func(t *testing.T) {
		testGetSchemaRetrievesExisting(t, rw)
	})

	t.Run("GetSchema returns ErrNotFound for non-existent schema", func(t *testing.T) {
		testGetSchemaReturnsNotFound(t, rw)
	})

	t.Run("ListSchemas returns all schemas in a namespace", func(t *testing.T) {
		testListSchemasReturnsAll(t, rw)
	})

	t.Run("DeleteSchema removes an existing schema", func(t *testing.T) {
		testDeleteSchemaRemovesExisting(t, rw)
	})

	t.Run("DeleteSchema succeeds for non-existent schema", func(t *testing.T) {
		testDeleteSchemaSucceedsForNonExistent(t, rw)
	})
}

func testPutSchemaCreatesNew(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	schemaDef := FakePostSchema()
	uri, err := core.ParseURI("xdb://default/posts")
	require.NoError(t, err)

	err = rw.PutSchema(ctx, uri, schemaDef)
	require.NoError(t, err)

	got, err := rw.GetSchema(ctx, uri)
	require.NoError(t, err)
	AssertDefEqual(t, schemaDef, got)
}

func testPutSchemaUpdatesExisting(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	uri, err := core.ParseURI("xdb://default/posts")
	require.NoError(t, err)

	originalSchema := FakePostSchema()
	err = rw.PutSchema(ctx, uri, originalSchema)
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

	err = rw.PutSchema(ctx, uri, updatedSchema)
	require.NoError(t, err)

	got, err := rw.GetSchema(ctx, uri)
	require.NoError(t, err)
	AssertDefEqual(t, updatedSchema, got)
}

func testGetSchemaRetrievesExisting(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	schemaDef := FakePostSchema()
	uri, err := core.ParseURI("xdb://default/posts")
	require.NoError(t, err)

	err = rw.PutSchema(ctx, uri, schemaDef)
	require.NoError(t, err)

	got, err := rw.GetSchema(ctx, uri)
	require.NoError(t, err)
	AssertDefEqual(t, schemaDef, got)
}

func testGetSchemaReturnsNotFound(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	uri, err := core.ParseURI("xdb://default/nonexistent")
	require.NoError(t, err)

	got, err := rw.GetSchema(ctx, uri)
	require.Error(t, err)
	require.ErrorIs(t, err, driver.ErrNotFound)
	require.Nil(t, got)
}

func testListSchemasReturnsAll(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	postSchema := FakePostSchema()
	postURI, err := core.ParseURI("xdb://default/posts")
	require.NoError(t, err)
	err = rw.PutSchema(ctx, postURI, postSchema)
	require.NoError(t, err)

	testSchema := FakeTestSchema()
	testURI, err := core.ParseURI("xdb://default/all_types")
	require.NoError(t, err)
	err = rw.PutSchema(ctx, testURI, testSchema)
	require.NoError(t, err)

	nsURI, err := core.ParseURI("xdb://default")
	require.NoError(t, err)

	schemas, err := rw.ListSchemas(ctx, nsURI)
	require.NoError(t, err)
	require.Len(t, schemas, 2)

	schemaMap := make(map[string]*schema.Def)
	for _, s := range schemas {
		schemaMap[s.Name] = s
	}

	require.Contains(t, schemaMap, "posts")
	require.Contains(t, schemaMap, "all_types")

	AssertDefEqual(t, postSchema, schemaMap["posts"])
	AssertDefEqual(t, testSchema, schemaMap["all_types"])
}

func testDeleteSchemaRemovesExisting(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	schemaDef := FakePostSchema()
	uri, err := core.ParseURI("xdb://default/posts")
	require.NoError(t, err)

	err = rw.PutSchema(ctx, uri, schemaDef)
	require.NoError(t, err)

	err = rw.DeleteSchema(ctx, uri)
	require.NoError(t, err)

	got, err := rw.GetSchema(ctx, uri)
	require.Error(t, err)
	require.ErrorIs(t, err, driver.ErrNotFound)
	require.Nil(t, got)
}

func testDeleteSchemaSucceedsForNonExistent(t *testing.T, rw driver.SchemaDriver) {
	t.Helper()
	ctx := context.Background()

	uri, err := core.ParseURI("xdb://default/nonexistent")
	require.NoError(t, err)

	err = rw.DeleteSchema(ctx, uri)
	require.NoError(t, err)
}
