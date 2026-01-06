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

type RecordStoreTestSuite struct {
	driver store.RecordStore
}

func NewRecordStoreTestSuite(d store.RecordStore) *RecordStoreTestSuite {
	return &RecordStoreTestSuite{driver: d}
}

func (s *RecordStoreTestSuite) supportsSchemaValidation() bool {
	_, ok := s.driver.(store.SchemaStore)
	return ok
}

func (s *RecordStoreTestSuite) setupSchema(t *testing.T, def *schema.Def) {
	t.Helper()

	if sd, ok := s.driver.(store.SchemaStore); ok {
		ctx := context.Background()
		schemaURI := core.New().NS("com.example").Schema(def.Name).MustURI()
		err := sd.PutSchema(ctx, schemaURI, def)
		require.NoError(t, err)
	}
}

func (s *RecordStoreTestSuite) Basic(t *testing.T) {
	t.Helper()

	s.setupSchema(t, FakePostSchema())

	ctx := context.Background()
	records := FakePosts(10)
	uris := x.URIs(records...)

	t.Run("PutRecords", func(t *testing.T) {
		err := s.driver.PutRecords(ctx, records)
		require.NoError(t, err)
	})

	t.Run("GetRecords", func(t *testing.T) {
		got, missing, err := s.driver.GetRecords(ctx, uris)
		require.NoError(t, err)
		require.Empty(t, missing)
		AssertEqualRecords(t, records, got)
	})

	t.Run("GetRecordsSomeMissing", func(t *testing.T) {
		notFound := []*core.URI{
			core.New().NS("com.example").Schema("posts").ID("not_found_1").MustURI(),
			core.New().NS("com.example").Schema("posts").ID("not_found_2").MustURI(),
		}

		got, missing, err := s.driver.GetRecords(ctx, append(uris, notFound...))
		require.NoError(t, err)
		AssertEqualURIs(t, notFound, missing)
		AssertEqualRecords(t, records, got)
	})

	t.Run("DeleteRecords", func(t *testing.T) {
		err := s.driver.DeleteRecords(ctx, uris)
		require.NoError(t, err)
	})

	t.Run("GetRecordsAllMissing", func(t *testing.T) {
		got, missing, err := s.driver.GetRecords(ctx, uris)
		require.NoError(t, err)
		require.NotEmpty(t, missing)
		require.Empty(t, got)
		AssertEqualURIs(t, missing, uris)
	})
}

func (s *RecordStoreTestSuite) ValidationStrict(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("PutRecords fails without schema", func(t *testing.T) {
		records := []*core.Record{
			core.NewRecord("com.example", "no_schema", "1").Set("name", "Alice"),
		}

		err := s.driver.PutRecords(ctx, records)
		require.ErrorIs(t, err, store.ErrNotFound)
	})

	t.Run("Rejects unknown fields", func(t *testing.T) {
		def := &schema.Def{
			Name: "strict_record_test",
			Mode: schema.ModeStrict,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		records := []*core.Record{
			core.NewRecord("com.example", "strict_record_test", "1").Set("unknown_field", "value"),
		}

		err := s.driver.PutRecords(ctx, records)
		require.ErrorIs(t, err, schema.ErrUnknownField)
	})

	t.Run("Rejects type mismatch", func(t *testing.T) {
		def := &schema.Def{
			Name: "strict_record_type_test",
			Mode: schema.ModeStrict,
			Fields: []*schema.FieldDef{
				{Name: "age", Type: core.TypeInt},
			},
		}
		s.setupSchema(t, def)

		records := []*core.Record{
			core.NewRecord("com.example", "strict_record_type_test", "1").Set("age", "not an int"),
		}

		err := s.driver.PutRecords(ctx, records)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})
}

func (s *RecordStoreTestSuite) ValidationFlexible(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("Accepts unknown fields", func(t *testing.T) {
		def := &schema.Def{
			Name: "flexible_record_test",
			Mode: schema.ModeFlexible,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		records := []*core.Record{
			core.NewRecord("com.example", "flexible_record_test", "1").Set("unknown_field", "value"),
		}

		err := s.driver.PutRecords(ctx, records)
		require.NoError(t, err)
	})
}

func (s *RecordStoreTestSuite) ValidationDynamic(t *testing.T) {
	t.Helper()

	if !s.supportsSchemaValidation() {
		t.Skip("driver does not support schema validation")
	}

	ctx := context.Background()

	t.Run("Infers new fields", func(t *testing.T) {
		def := &schema.Def{
			Name: "dynamic_record_test",
			Mode: schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "name", Type: core.TypeString},
			},
		}
		s.setupSchema(t, def)

		records := []*core.Record{
			core.NewRecord("com.example", "dynamic_record_test", "1").
				Set("name", "Alice").
				Set("age", int64(30)),
		}

		err := s.driver.PutRecords(ctx, records)
		require.NoError(t, err)

		sr := s.driver.(store.SchemaReader)
		schemaURI := core.New().NS("com.example").Schema("dynamic_record_test").MustURI()
		got, err := sr.GetSchema(ctx, schemaURI)
		require.NoError(t, err)
		require.NotNil(t, got.GetField("age"))
		require.Equal(t, core.TypeInt, got.GetField("age").Type)
	})

	t.Run("Rejects type mismatch on existing fields", func(t *testing.T) {
		def := &schema.Def{
			Name: "dynamic_record_type_test",
			Mode: schema.ModeDynamic,
			Fields: []*schema.FieldDef{
				{Name: "count", Type: core.TypeInt},
			},
		}
		s.setupSchema(t, def)

		records := []*core.Record{
			core.NewRecord("com.example", "dynamic_record_type_test", "1").Set("count", "not an int"),
		}

		err := s.driver.PutRecords(ctx, records)
		require.ErrorIs(t, err, schema.ErrTypeMismatch)
	})
}
