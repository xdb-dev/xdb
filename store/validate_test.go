package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func requiredStrictDef(uri *core.URI) *schema.Def {
	return &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString, Required: true},
			"count": {Type: core.TypeInt},
		},
	}
}

func TestValidateRecord(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	v := s.(store.Validator)

	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, s.CreateSchema(ctx, uri, requiredStrictDef(uri)))

	t.Run("valid create passes and writes nothing", func(t *testing.T) {
		rec := core.NewRecord("com.example", "posts", "p1").Set("title", "A")
		require.NoError(t, v.ValidateRecord(ctx, rec, store.OpCreate))

		_, err := s.GetRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1"))
		assert.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("unknown field under strict mode fails", func(t *testing.T) {
		rec := core.NewRecord("com.example", "posts", "p2").
			Set("title", "A").
			Set("bogus", "x")
		err := v.ValidateRecord(ctx, rec, store.OpCreate)
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("missing required field on create fails", func(t *testing.T) {
		rec := core.NewRecord("com.example", "posts", "p3").Set("count", int64(1))
		err := v.ValidateRecord(ctx, rec, store.OpCreate)
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("patch on existing record may omit required fields", func(t *testing.T) {
		seeded := core.NewRecord("com.example", "posts", "p4").Set("title", "A")
		require.NoError(t, s.CreateRecord(ctx, seeded))

		patch := core.NewRecord("com.example", "posts", "p4").Set("count", int64(2))
		assert.NoError(t, v.ValidateRecord(ctx, patch, store.OpPatch))
	})

	t.Run("no schema means no policy", func(t *testing.T) {
		rec := core.NewRecord("no.schema", "things", "t1").Set("anything", "goes")
		assert.NoError(t, v.ValidateRecord(ctx, rec, store.OpCreate))
	})
}

func TestValidateRecord_DynamicModeDoesNotPersistEvolution(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	v := s.(store.Validator)

	uri := core.MustParseURI("xdb://com.example/ticks")
	def := &schema.Def{
		URI:  uri,
		Mode: schema.ModeDynamic,
		Fields: map[string]schema.Field{
			"symbol": {Type: core.TypeString},
		},
	}
	require.NoError(t, s.CreateSchema(ctx, uri, def))

	before, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)

	rec := core.NewRecord("com.example", "ticks", "t1").
		Set("symbol", "RELIANCE").
		Set("exchange", "NSE")
	require.NoError(t, v.ValidateRecord(ctx, rec, store.OpCreate))

	after, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)
	assert.Equal(t, before.Revision, after.Revision)
	assert.NotContains(t, after.Fields, "exchange")
}

func TestValidateDeleteRecord(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	v := s.(store.Validator)

	uri := core.MustParseURI("xdb://com.example/posts")
	require.NoError(t, s.CreateSchema(ctx, uri, requiredStrictDef(uri)))

	t.Run("whole-record delete passes", func(t *testing.T) {
		err := v.ValidateDeleteRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1"))
		assert.NoError(t, err)
	})

	t.Run("deleting a required attr fails", func(t *testing.T) {
		err := v.ValidateDeleteRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1#title"))
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
	})

	t.Run("deleting an optional attr passes", func(t *testing.T) {
		err := v.ValidateDeleteRecord(ctx, core.MustParseURI("xdb://com.example/posts/p1#count"))
		assert.NoError(t, err)
	})
}
