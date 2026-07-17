package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// ModeStoreSuite runs tests that verify schema mode enforcement
// on record write operations. It requires [store.Store] because
// tests create schemas and then write records against them.
type ModeStoreSuite struct {
	newStore func() store.Store
}

// NewModeStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewModeStoreSuite(fn func() store.Store) *ModeStoreSuite {
	return &ModeStoreSuite{newStore: fn}
}

// Run runs all mode enforcement tests as subtests of t.
func (s *ModeStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Flexible", s.testFlexible)
	t.Run("Strict", s.testStrict)
	t.Run("Dynamic", s.testDynamic)
	t.Run("NoSchema", s.testNoSchema)
	t.Run("ArrayElemType", s.testArrayElemType)
	t.Run("ObjectArray", s.testObjectArray)
}

// mustJSONElem marshals a map to a canonical JSON object for use as an
// ARRAY<JSON> element. json.Marshal sorts keys, so every backend round-trips
// byte-identical bytes.
func mustJSONElem(m map[string]any) *core.Value {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return core.JSONVal(data)
}

func (s *ModeStoreSuite) testObjectArray(t *testing.T) {
	ctx := context.Background()

	objectArraySchema := func(uri *core.URI) *schema.Def {
		return &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"lines": {
					Type: core.NewArrayType(core.TIDJSON),
					Items: map[string]schema.Field{
						"sku": {Type: core.TypeString, Required: true},
						"qty": {Type: core.TypeInt},
					},
				},
			},
		}
	}

	t.Run("round-trips object array", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/orders")
		require.NoError(t, st.CreateSchema(ctx, uri, objectArraySchema(uri)))

		r := core.NewRecord("com.example", "orders", "1")
		r.Set("lines", core.ArrayVal(core.TIDJSON,
			mustJSONElem(map[string]any{"sku": "A-1", "qty": int64(3)}),
			mustJSONElem(map[string]any{"sku": "B-2", "qty": int64(5)}),
		))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		AssertEqualRecord(t, r, got)
	})

	t.Run("rejects element member type mismatch", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/orders_mismatch")
		require.NoError(t, st.CreateSchema(ctx, uri, objectArraySchema(uri)))

		r := core.NewRecord("com.example", "orders_mismatch", "1")
		r.Set("lines", core.ArrayVal(core.TIDJSON,
			core.JSONVal(json.RawMessage(`{"sku":"A-1","qty":"three"}`)),
		))

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "qty")
	})

	t.Run("rejects missing required member in element", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/orders_required")
		require.NoError(t, st.CreateSchema(ctx, uri, objectArraySchema(uri)))

		r := core.NewRecord("com.example", "orders_required", "1")
		r.Set("lines", core.ArrayVal(core.TIDJSON,
			core.JSONVal(json.RawMessage(`{"qty":3}`)),
		))

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "sku")
	})
}

func (s *ModeStoreSuite) testFlexible(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/posts")
	def := &schema.Def{
		URI:  schemaURI,
		Mode: schema.ModeFlexible,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString, Required: true},
		},
	}
	require.NoError(t, st.CreateSchema(ctx, schemaURI, def))

	t.Run("accepts defined fields", func(t *testing.T) {
		r := core.NewRecord("com.example", "posts", "flex-1")
		r.Set("title", "Hello")
		require.NoError(t, st.CreateRecord(ctx, r))
	})

	t.Run("accepts unknown fields", func(t *testing.T) {
		r := core.NewRecord("com.example", "posts", "flex-2")
		r.Set("title", "Hello")
		r.Set("extra", "not in schema")
		require.NoError(t, st.CreateRecord(ctx, r))
	})

	// Fixed mode semantics: declared fields type-check in every mode, so a
	// flexible schema now rejects a type mismatch on a declared field.
	t.Run("rejects wrong type for defined field", func(t *testing.T) {
		r := core.NewRecord("com.example", "posts", "flex-3")
		r.Set("title", 42) // schema says STRING, sending INTEGER

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "title")
	})

	// Required is a declared-field property enforced in every mode.
	t.Run("rejects missing required field", func(t *testing.T) {
		r := core.NewRecord("com.example", "posts", "flex-4")
		r.Set("extra", "no title present")

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "title")
	})
}

func (s *ModeStoreSuite) testStrict(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/articles")
	def := &schema.Def{
		URI:  schemaURI,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title":  {Type: core.TypeString, Required: true},
			"rating": {Type: core.TypeFloat},
		},
	}
	require.NoError(t, st.CreateSchema(ctx, schemaURI, def))

	t.Run("accepts valid record", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-1")
		r.Set("title", "Valid")
		r.Set("rating", 4.5)
		require.NoError(t, st.CreateRecord(ctx, r))
	})

	t.Run("rejects unknown field", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-2")
		r.Set("title", "Valid")
		r.Set("extra", "not in schema")

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("rejects wrong type", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-3")
		r.Set("title", 42) // schema says STRING

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)

		// Error should include field-level detail.
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("rejects missing required field", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-required")
		r.Set("rating", 4.5) // title (required) missing

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("rejects on update", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-update")
		r.Set("title", "Valid")
		require.NoError(t, st.CreateRecord(ctx, r))

		updated := core.NewRecord("com.example", "articles", "strict-update")
		updated.Set("title", "Still Valid")
		updated.Set("extra", "not allowed")

		err := st.UpdateRecord(ctx, updated)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("rejects on upsert", func(t *testing.T) {
		r := core.NewRecord("com.example", "articles", "strict-upsert")
		r.Set("extra", "not in schema")

		err := st.UpsertRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})
}

func (s *ModeStoreSuite) testDynamic(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	schemaURI := core.MustParseURI("xdb://com.example/events")
	def := &schema.Def{
		URI:  schemaURI,
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString, Required: true},
		},
	}
	require.NoError(t, st.CreateSchema(ctx, schemaURI, def))

	t.Run("accepts defined fields", func(t *testing.T) {
		r := core.NewRecord("com.example", "events", "dyn-1")
		r.Set("name", "click")
		require.NoError(t, st.CreateRecord(ctx, r))
	})

	t.Run("accepts unknown field and evolves schema", func(t *testing.T) {
		r := core.NewRecord("com.example", "events", "dyn-2")
		r.Set("name", "click")
		r.Set("count", int64(5))
		require.NoError(t, st.CreateRecord(ctx, r))

		// Schema should now include the "count" field.
		got, err := st.GetSchema(ctx, schemaURI)
		require.NoError(t, err)

		countField, ok := got.Fields["count"]
		require.True(t, ok, "schema should have inferred 'count' field")
		assert.Equal(t, core.TIDInteger, countField.Type.ID())
	})

	t.Run("rejects wrong type for existing field", func(t *testing.T) {
		r := core.NewRecord("com.example", "events", "dyn-3")
		r.Set("name", 42) // schema says STRING

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("rejects missing required field", func(t *testing.T) {
		r := core.NewRecord("com.example", "events", "dyn-required")
		r.Set("count", int64(1)) // name (required) missing

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("evolves schema on upsert", func(t *testing.T) {
		r := core.NewRecord("com.example", "events", "dyn-4")
		r.Set("name", "scroll")
		r.Set("active", true)
		require.NoError(t, st.UpsertRecord(ctx, r))

		got, err := st.GetSchema(ctx, schemaURI)
		require.NoError(t, err)

		activeField, ok := got.Fields["active"]
		require.True(t, ok, "schema should have inferred 'active' field")
		assert.Equal(t, core.TIDBoolean, activeField.Type.ID())
	})

	t.Run("evolves schema on update", func(t *testing.T) {
		// Create with known fields first.
		r := core.NewRecord("com.example", "events", "dyn-5")
		r.Set("name", "hover")
		require.NoError(t, st.CreateRecord(ctx, r))

		// Update with a new field.
		updated := core.NewRecord("com.example", "events", "dyn-5")
		updated.Set("name", "hover")
		updated.Set("duration", 3.14)
		require.NoError(t, st.UpdateRecord(ctx, updated))

		got, err := st.GetSchema(ctx, schemaURI)
		require.NoError(t, err)

		durField, ok := got.Fields["duration"]
		require.True(t, ok, "schema should have inferred 'duration' field")
		assert.Equal(t, core.TIDFloat, durField.Type.ID())
	})
}

func (s *ModeStoreSuite) testArrayElemType(t *testing.T) {
	ctx := context.Background()

	t.Run("strict mode roundtrips typed array", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/strict_arrays")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"title": {Type: core.TypeString},
				"tags":  {Type: core.NewArrayType(core.TIDString)},
			},
		}
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		r := core.NewRecord("com.example", "strict_arrays", "1")
		r.Set("title", "hello")
		r.Set("tags", core.ArrayVal(core.TIDString,
			core.StringVal("go"),
			core.StringVal("db"),
		))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		tags, err := got.Get("tags").Value().AsArray()
		require.NoError(t, err)
		assert.Len(t, tags, 2)
	})

	t.Run("strict mode rejects mismatched elem type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/strict_arrays_mismatch")
		def := &schema.Def{
			URI:  uri,
			Mode: schema.ModeStrict,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
			},
		}
		require.NoError(t, st.CreateSchema(ctx, uri, def))

		r := core.NewRecord("com.example", "strict_arrays_mismatch", "1")
		r.Set("tags", core.ArrayVal(core.TIDInteger, core.IntVal(1)))

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})

	t.Run("dynamic mode infers array elem type", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/dyn_arrays")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"name": {Type: core.TypeString, Required: true},
			},
		}))

		r := core.NewRecord("com.example", "dyn_arrays", "1")
		r.Set("name", "click")
		r.Set("tags", core.ArrayVal(core.TIDString,
			core.StringVal("a"),
			core.StringVal("b"),
		))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetSchema(ctx, uri)
		require.NoError(t, err)
		tags, ok := got.Fields["tags"]
		require.True(t, ok)
		assert.Equal(t, core.TIDArray, tags.Type.ID())
		assert.Equal(t, core.TIDString, tags.Type.ElemTypeID())
	})

	t.Run("dynamic mode rejects mismatched elem type on known field", func(t *testing.T) {
		st := s.newStore()
		uri := core.MustParseURI("xdb://com.example/dyn_arrays_mismatch")
		require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
			URI:  uri,
			Mode: schema.ModeDynamic,
			Fields: map[string]schema.Field{
				"tags": {Type: core.NewArrayType(core.TIDString)},
			},
		}))

		r := core.NewRecord("com.example", "dyn_arrays_mismatch", "1")
		r.Set("tags", core.ArrayVal(core.TIDInteger, core.IntVal(1)))

		err := st.CreateRecord(ctx, r)
		require.ErrorIs(t, err, store.ErrSchemaViolation)
	})
}

func (s *ModeStoreSuite) testNoSchema(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	// Create a flexible schema (no field constraints) so the backing table exists.
	schemaURI := core.MustParseURI("xdb://com.example/unschematized")
	require.NoError(t, st.CreateSchema(ctx, schemaURI, &schema.Def{
		URI:  schemaURI,
		Mode: schema.ModeFlexible,
	}))

	t.Run("accepts any record with flexible schema", func(t *testing.T) {
		r := core.NewRecord("com.example", "unschematized", "no-schema-1")
		r.Set("anything", "goes")
		r.Set("count", int64(42))
		require.NoError(t, st.CreateRecord(ctx, r))

		got, err := st.GetRecord(ctx, r.URI())
		require.NoError(t, err)
		AssertEqualRecord(t, r, got)
	})

	t.Run("upsert works with flexible schema", func(t *testing.T) {
		r := core.NewRecord("com.example", "unschematized", "no-schema-2")
		r.Set("field", "value")
		require.NoError(t, st.UpsertRecord(ctx, r))
	})

	t.Run("update works with flexible schema", func(t *testing.T) {
		r := core.NewRecord("com.example", "unschematized", "no-schema-3")
		r.Set("field", "value")
		require.NoError(t, st.CreateRecord(ctx, r))

		updated := core.NewRecord("com.example", "unschematized", "no-schema-3")
		updated.Set("field", "new value")
		updated.Set("extra", true)
		require.NoError(t, st.UpdateRecord(ctx, updated))
	})
}
