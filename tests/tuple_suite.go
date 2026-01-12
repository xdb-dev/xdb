package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/x"
)

type TupleStoreTestSuite struct {
	driver store.TupleStore
}

func NewTupleStoreTestSuite(d store.TupleStore) *TupleStoreTestSuite {
	return &TupleStoreTestSuite{driver: d}
}

func (s *TupleStoreTestSuite) supportsSchemaValidation() bool {
	_, ok := s.driver.(store.SchemaStore)
	return ok
}

func (s *TupleStoreTestSuite) setupSchema(t *testing.T, def *schema.Def) {
	t.Helper()

	if sd, ok := s.driver.(store.SchemaStore); ok {
		ctx := context.Background()
		schemaURI := core.New().NS("com.example").Schema(def.Name).MustURI()
		err := sd.PutSchema(ctx, schemaURI, def)
		require.NoError(t, err)
	}
}

func (s *TupleStoreTestSuite) Basic(t *testing.T) {
	t.Helper()

	s.setupSchema(t, FakeAllTypesSchema())

	ctx := context.Background()
	tuples := FakeTuples()
	uris := x.URIs(tuples...)

	t.Run("PutTuples", func(t *testing.T) {
		err := s.driver.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})

	t.Run("GetTuples", func(t *testing.T) {
		got, missing, err := s.driver.GetTuples(ctx, uris)
		require.NoError(t, err)
		require.Empty(t, missing)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("GetTuples with some missing", func(t *testing.T) {
		notFound := []*core.URI{
			core.New().NS("com.example").Schema("all_types").ID("not_found_1").MustURI(),
			core.New().NS("com.example").Schema("all_types").ID("not_found_2").MustURI(),
		}

		got, missing, err := s.driver.GetTuples(ctx, append(uris, notFound...))
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		AssertEqualURIs(t, missing, notFound)
		AssertEqualTuples(t, tuples, got)
	})

	t.Run("DeleteTuples", func(t *testing.T) {
		err := s.driver.DeleteTuples(ctx, uris)
		require.NoError(t, err)
	})

	t.Run("GetTuples all missing", func(t *testing.T) {
		got, missing, err := s.driver.GetTuples(ctx, uris)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Empty(t, got)
		AssertEqualURIs(t, missing, uris)
	})
}

func (s *TupleStoreTestSuite) ValidationStrict(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("PutTuples fails without schema", func(t *testing.T) {
		tuples := []*core.Tuple{
			core.NewTuple("com.example/no_schema/1", "name", "Alice"),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Rejects unknown fields", func(t *testing.T) {
		def := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "strict_test",
			Mode: schema.ModeStrict,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		tuples := []*core.Tuple{
			core.NewTuple("com.example/strict_test/1", "unknown_field", "value"),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.ErrorIs(t, err, schema.ErrUnknownField)
	})

	t.Run("Rejects type mismatch", func(t *testing.T) {
		def := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "strict_type_test",
			Mode: schema.ModeStrict,
			Fields: []*schema.FieldDef{
				{Name: "age", Type: core.TypeInt},
			},
		}
		s.setupSchema(t, def)

		tuples := []*core.Tuple{
			core.NewTuple("com.example/strict_type_test/1", "age", "not an int"),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})
}

func (s *TupleStoreTestSuite) ValidationFlexible(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("Accepts unknown fields", func(t *testing.T) {
		def := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "flexible_test",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		tuples := []*core.Tuple{
			core.NewTuple("com.example/flexible_test/1", "unknown_field", "value"),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.NoError(t, err)
	})
}

func (s *TupleStoreTestSuite) ValidationDynamic(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("Infers new fields", func(t *testing.T) {
		def := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "dynamic_test",
			Mode: schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		tuples := []*core.Tuple{
			core.NewTuple("com.example/dynamic_test/1", "name", "Alice"),
			core.NewTuple("com.example/dynamic_test/1", "age", int64(30)),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.NoError(t, err)

		sr := s.driver.(store.SchemaReader)
		schemaURI := core.New().NS("com.example").Schema("dynamic_test").MustURI()
		got, err := sr.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		require.NotNil(t, got.GetField("age"))
		require.Equal(t, core.TypeInt, got.GetField("age").Type)
	})

	t.Run("Rejects type mismatch on existing fields", func(t *testing.T) {
		def := &schema.Def{
			NS:   core.NewNS("com.example"),
			Name: "dynamic_type_test",
			Mode: schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "count", Type: core.TypeInt},
			},
		}
		s.setupSchema(t, def)

		tuples := []*core.Tuple{
			core.NewTuple("com.example/dynamic_type_test/1", "count", "not an int"),
		}

		err := s.driver.PutTuples(ctx, tuples)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})
}
