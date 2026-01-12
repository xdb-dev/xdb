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
