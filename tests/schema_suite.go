package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

type SchemaStoreTestSuite struct {
	driver store.SchemaStore
}

func NewSchemaStoreTestSuite(d store.SchemaStore) *SchemaStoreTestSuite {
	return &SchemaStoreTestSuite{driver: d}
}

func (s *SchemaStoreTestSuite) Basic(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	allTypesSchema := FakeAllTypesSchema()
	allTypesURI := core.MustParseURI("xdb://com.example/all_types")

	t.Run("PutSchema creates a new schema", func(t *testing.T) {
		err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
		require.NoError(t, err)

		got, err := s.driver.GetSchema(ctx, allTypesURI)
		require.NoError(t, err)
		AssertDefEqual(t, allTypesSchema, got)
	})

	t.Run("PutSchema idempotently updates an existing schema", func(t *testing.T) {
		err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
		require.NoError(t, err)

		got, err := s.driver.GetSchema(ctx, allTypesURI)
		require.NoError(t, err)
		AssertDefEqual(t, allTypesSchema, got)
	})

	t.Run("DeleteSchema removes an existing schema", func(t *testing.T) {
		err := s.driver.DeleteSchema(ctx, allTypesURI)
		require.NoError(t, err)

		got, err := s.driver.GetSchema(ctx, allTypesURI)
		require.Error(t, err)
		require.ErrorIs(t, err, store.ErrNotFound)
		require.Nil(t, got)
	})

	t.Run("DeleteSchema succeeds for non-existent schema", func(t *testing.T) {
		err := s.driver.DeleteSchema(ctx, allTypesURI)
		require.NoError(t, err)
	})

	t.Run("GetSchema returns ErrNotFound for non-existent schema", func(t *testing.T) {
		got, err := s.driver.GetSchema(ctx, allTypesURI)
		require.Error(t, err)
		require.ErrorIs(t, err, store.ErrNotFound)
		require.Nil(t, got)
	})
}

func (s *SchemaStoreTestSuite) ListSchemas(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	allTypesSchema1 := FakeAllTypesSchema()
	allTypesSchema1.Name = "all_types_1"
	allTypesSchema2 := FakeAllTypesSchema()
	allTypesSchema2.Name = "all_types_2"

	allTypesURI1 := core.MustParseURI("xdb://com.example/all_types_1")
	allTypesURI2 := core.MustParseURI("xdb://com.example/all_types_2")

	err := s.driver.PutSchema(ctx, allTypesURI1, allTypesSchema1)
	require.NoError(t, err)

	err = s.driver.PutSchema(ctx, allTypesURI2, allTypesSchema2)
	require.NoError(t, err)

	schemas, err := s.driver.ListSchemas(ctx, core.MustParseURI("xdb://com.example"))
	require.NoError(t, err)
	require.Len(t, schemas, 2)
	AssertDefEqual(t, allTypesSchema1, schemas[0])
	AssertDefEqual(t, allTypesSchema2, schemas[1])
}

func (s *SchemaStoreTestSuite) ListNamespaces(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	t.Run("ListNamespaces returns empty slice when no schemas exist", func(t *testing.T) {
		namespaces, err := s.driver.ListNamespaces(ctx)
		require.NoError(t, err)
		require.NotNil(t, namespaces)
		require.Len(t, namespaces, 0)
	})

	t.Run("ListNamespaces returns unique namespaces sorted alphabetically", func(t *testing.T) {
		schema1 := FakeAllTypesSchema()
		schema1.NS = core.NewNS("com.example")
		schema1.Name = "users"

		schema2 := FakeAllTypesSchema()
		schema2.NS = core.NewNS("com.example")
		schema2.Name = "posts"

		schema3 := FakeAllTypesSchema()
		schema3.NS = core.NewNS("org.test")
		schema3.Name = "products"

		err := s.driver.PutSchema(ctx, core.MustParseURI("xdb://com.example/users"), schema1)
		require.NoError(t, err)

		err = s.driver.PutSchema(ctx, core.MustParseURI("xdb://com.example/posts"), schema2)
		require.NoError(t, err)

		err = s.driver.PutSchema(ctx, core.MustParseURI("xdb://org.test/products"), schema3)
		require.NoError(t, err)

		namespaces, err := s.driver.ListNamespaces(ctx)
		require.NoError(t, err)
		require.Len(t, namespaces, 2)

		require.Equal(t, "com.example", namespaces[0].String())
		require.Equal(t, "org.test", namespaces[1].String())
	})

	t.Run("ListNamespaces returns one namespace after deletion", func(t *testing.T) {
		err := s.driver.DeleteSchema(ctx, core.MustParseURI("xdb://org.test/products"))
		require.NoError(t, err)

		namespaces, err := s.driver.ListNamespaces(ctx)
		require.NoError(t, err)
		require.Len(t, namespaces, 1)
		require.Equal(t, "com.example", namespaces[0].String())
	})
}

func (s *SchemaStoreTestSuite) AddNewFields(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	allTypesSchema := FakeAllTypesSchema()
	allTypesURI := core.MustParseURI("xdb://com.example/all_types")

	err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
	require.NoError(t, err)

	updatedSchema := allTypesSchema.Clone()
	updatedSchema.Fields = append(updatedSchema.Fields,
		&schema.FieldDef{Name: "new_field", Type: core.TypeString},
		&schema.FieldDef{Name: "new_field_2", Type: core.TypeInt},
	)
	err = s.driver.PutSchema(ctx, allTypesURI, updatedSchema)
	require.NoError(t, err)

	got, err := s.driver.GetSchema(ctx, allTypesURI)
	require.NoError(t, err)
	AssertDefEqual(t, updatedSchema, got)
}

func (s *SchemaStoreTestSuite) DropFields(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	allTypesSchema := FakeAllTypesSchema()
	allTypesURI := core.MustParseURI("xdb://com.example/all_types")

	err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
	require.NoError(t, err)

	updatedSchema := allTypesSchema.Clone()
	updatedSchema.Fields = updatedSchema.Fields[:len(updatedSchema.Fields)-2]
	err = s.driver.PutSchema(ctx, allTypesURI, updatedSchema)
	require.NoError(t, err)

	got, err := s.driver.GetSchema(ctx, allTypesURI)
	require.NoError(t, err)
	AssertDefEqual(t, updatedSchema, got)
}

func (s *SchemaStoreTestSuite) ModifyFields(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	allTypesSchema := FakeAllTypesSchema()
	allTypesURI := core.MustParseURI("xdb://com.example/all_types")

	err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
	require.NoError(t, err)

	updatedSchema := allTypesSchema.Clone()
	updatedSchema.Fields[0].Type = core.TypeInt

	err = s.driver.PutSchema(ctx, allTypesURI, updatedSchema)
	require.Error(t, err)
	require.ErrorIs(t, err, store.ErrFieldChangeType)
}

func (s *SchemaStoreTestSuite) EdgeCases(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	t.Run("PutSchema prevents mode change from strict to flexible", func(t *testing.T) {
		allTypesSchema := FakeAllTypesSchema()
		allTypesURI := core.MustParseURI("xdb://com.example/all_types")

		err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
		require.NoError(t, err)

		flexibleSchema := allTypesSchema.Clone()
		flexibleSchema.Mode = schema.ModeFlexible
		err = s.driver.PutSchema(ctx, allTypesURI, flexibleSchema)
		require.Error(t, err)
		require.ErrorIs(t, err, store.ErrSchemaModeChanged)
	})

	t.Run("PutSchema prevents mode change from flexible to strict", func(t *testing.T) {
		allTypesSchema := FakeAllTypesSchema()
		allTypesSchema.Name = "all_types_flexible"
		allTypesSchema.Mode = schema.ModeFlexible

		allTypesURI := core.MustParseURI("xdb://com.example/all_types_flexible")

		err := s.driver.PutSchema(ctx, allTypesURI, allTypesSchema)
		require.NoError(t, err)

		strictSchema := allTypesSchema.Clone()
		strictSchema.Mode = schema.ModeStrict
		err = s.driver.PutSchema(ctx, allTypesURI, strictSchema)
		require.Error(t, err)
		require.ErrorIs(t, err, store.ErrSchemaModeChanged)
	})
}
