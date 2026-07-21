package store_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

func testDef(uri *core.URI) *schema.Def {
	return &schema.Def{
		URI:  uri,
		Mode: schema.ModeFlexible,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString},
		},
	}
}

// TestSchemaCache_CoherentAcrossTxWrites pins cache coherency on
// TxDriver-backed stores: schema writes run inside transactions that
// bypass the cache, so the facade must keep the cache coherent after
// the commit.
func TestSchemaCache_CoherentAcrossTxWrites(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver(), store.WithSchemaCache())
	uri := core.MustParseURI("xdb://com.example/posts")

	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	// Warm the cache.
	got, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Revision)

	// Update through the tx write path.
	updated := testDef(uri)
	updated.Fields["author"] = schema.Field{Type: core.TypeString}
	require.NoError(t, s.UpdateSchema(ctx, uri, updated))

	// The cached def must not be stale.
	got, err = s.GetSchema(ctx, uri)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got.Revision, "cache returned a stale def")
	assert.Contains(t, got.Fields, "author")
}

// TestSchemaCache_CoherentAfterDelete pins that deletes evict.
func TestSchemaCache_CoherentAfterDelete(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver(), store.WithSchemaCache())
	uri := core.MustParseURI("xdb://com.example/posts")

	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	_, err := s.GetSchema(ctx, uri) // warm
	require.NoError(t, err)

	require.NoError(t, s.DeleteSchema(ctx, uri))

	_, err = s.GetSchema(ctx, uri)
	require.ErrorIs(t, err, core.ErrNotFound)
}

// TestSchemaCache_CoherentAfterDynamicEvolve pins that dynamic-mode
// record writes (which evolve the schema inside the write tx) do not
// leave a stale cached def behind.
func TestSchemaCache_CoherentAfterDynamicEvolve(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver(), store.WithSchemaCache())
	uri := core.MustParseURI("xdb://com.example/events")

	def := testDef(uri)
	def.Mode = schema.ModeDynamic
	require.NoError(t, s.CreateSchema(ctx, uri, def))

	_, err := s.GetSchema(ctx, uri) // warm
	require.NoError(t, err)

	rec := core.NewRecord("com.example", "events", "e1")
	rec.Set("title", "Hello")
	rec.Set("count", int64(2))
	require.NoError(t, s.CreateRecord(ctx, rec))

	got, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)
	assert.Contains(t, got.Fields, "count", "cache returned a pre-evolve def")
}

// TestSchemaCache_CoherentAfterRun pins that schema writes made
// through [store.TX.Run] invalidate the cache on commit, same as the
// facade's own tx write path.
func TestSchemaCache_CoherentAfterRun(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver(), store.WithSchemaCache())
	uri := core.MustParseURI("xdb://com.example/posts")

	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	_, err := s.GetSchema(ctx, uri) // warm
	require.NoError(t, err)

	tx, ok := s.(store.TX)
	require.True(t, ok)
	require.NoError(t, tx.Run(ctx, func(txs store.Store) error {
		updated := testDef(uri)
		updated.Fields["author"] = schema.Field{Type: core.TypeString}
		return txs.UpdateSchema(ctx, uri, updated)
	}))

	got, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got.Revision, "cache returned a stale def after Run")
	assert.Contains(t, got.Fields, "author")
}

// TestTxReadsOwnSchemaWrite pins that a transaction reads its own
// schema write, not a cached one.
func TestTxReadsOwnSchemaWrite(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver(), store.WithSchemaCache())
	uri := core.MustParseURI("xdb://com.example/posts")

	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	_, err := s.GetSchema(ctx, uri) // warm
	require.NoError(t, err)

	tx, ok := s.(store.TX)
	require.True(t, ok)

	err = tx.Run(ctx, func(txs store.Store) error {
		updated := testDef(uri)
		updated.Fields["author"] = schema.Field{Type: core.TypeString}
		if updateErr := txs.UpdateSchema(ctx, uri, updated); updateErr != nil {
			return updateErr
		}

		got, getErr := txs.GetSchema(ctx, uri)
		if getErr != nil {
			return getErr
		}
		assert.Contains(t, got.Fields, "author",
			"tx must read its own schema write")
		return nil
	})
	require.NoError(t, err)
}

// countingDriver counts raw def point-reads so cache hits are
// observable. Embedding the interface hides the concrete driver's
// optional capabilities, so writes take the non-tx path.
type countingDriver struct {
	store.Driver
	defReads int
}

func (c *countingDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	c.defReads++
	return c.Driver.GetSchema(ctx, uri)
}

// TestSchemaCache_ServesFromCache pins that a warmed cache serves
// GetSchema without a raw driver read — and that the cachingDriver
// override intercepts the driver's def point-read at all (a missed
// rename on the embedding cachingDriver would compile but leave the
// override dead).
func TestSchemaCache_ServesFromCache(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://com.example/posts")

	// Seed through a separate store so the cache under test starts cold.
	raw := xdbmemory.NewDriver()
	seed := store.New(raw)
	require.NoError(t, seed.CreateSchema(ctx, uri, testDef(uri)))

	cd := &countingDriver{Driver: raw}
	s := store.New(cd, store.WithSchemaCache())

	_, err := s.GetSchema(ctx, uri)
	require.NoError(t, err)
	require.Equal(t, 1, cd.defReads, "cold read must hit the raw driver once")

	_, err = s.GetSchema(ctx, uri)
	require.NoError(t, err)
	assert.Equal(t, 1, cd.defReads, "warm read must be served from cache")
}

// TestWithLogger_LogsDefWrites pins that def writes pass through the
// logging middleware (a missed rename on the embedding loggingDriver
// would compile but stop logging def writes).
func TestWithLogger_LogsDefWrites(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	s := store.New(xdbmemory.NewDriver(), store.WithLogger(logger))
	uri := core.MustParseURI("xdb://com.example/posts")

	require.NoError(t, s.CreateSchema(ctx, uri, testDef(uri)))

	assert.Contains(t, buf.String(), "store.create_schema")
}

// TestWithLogger_LogsWrites is a smoke test for the logging middleware.
func TestWithLogger_LogsWrites(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	s := store.New(xdbmemory.NewDriver(), store.WithLogger(logger))

	rec := core.NewRecord("com.example", "posts", "p1")
	rec.Set("title", "Hello")
	require.NoError(t, s.CreateRecord(ctx, rec))

	assert.Contains(t, buf.String(), "store.apply")
}

// strictDef returns a strict-mode def with a required title field and
// an optional note field.
func strictDef(uri *core.URI) *schema.Def {
	return &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString, Required: true},
			"note":  {Type: core.TypeString},
		},
	}
}

// TestPutTuples_AttributesFailureToMutation pins that batch tuple
// writes attribute the first failure to its mutation — attribution is
// the facade's job, not the driver's.
func TestPutTuples_AttributesFailureToMutation(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	uri := core.MustParseURI("xdb://com.example/articles")

	require.NoError(t, s.CreateSchema(ctx, uri, strictDef(uri)))

	err := s.PutTuples(ctx,
		core.NewTuple("com.example/articles/a", "title", "Fine"),
		core.NewTuple("com.example/articles/b", "undeclared", "Boom"),
	)
	require.ErrorIs(t, err, core.ErrSchemaViolation)

	var merr *store.MutationError
	require.ErrorAs(t, err, &merr)
	assert.Equal(t, 1, merr.Index)
	assert.Equal(t, "com.example/articles/b", merr.Path.Path())
}

// TestDeleteTuples_AttributesFailureToMutation pins attribution on the
// batch delete path: stripping a required attr fails on the second
// mutation.
func TestDeleteTuples_AttributesFailureToMutation(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())
	uri := core.MustParseURI("xdb://com.example/articles")

	require.NoError(t, s.CreateSchema(ctx, uri, strictDef(uri)))

	recA := core.NewRecord("com.example", "articles", "a")
	recA.Set("title", "A")
	require.NoError(t, s.CreateRecord(ctx, recA))
	recB := core.NewRecord("com.example", "articles", "b")
	recB.Set("title", "B")
	require.NoError(t, s.CreateRecord(ctx, recB))

	err := s.DeleteTuples(ctx,
		core.MustParseURI("xdb://com.example/articles/a#note"),
		core.MustParseURI("xdb://com.example/articles/b#title"),
	)
	require.ErrorIs(t, err, core.ErrSchemaViolation)

	var merr *store.MutationError
	require.ErrorAs(t, err, &merr)
	assert.Equal(t, 1, merr.Index)
	assert.Equal(t, "com.example/articles/b", merr.Path.Path())
}

// TestSingleRecordVerbs_ReturnBareSentinels pins that single-record
// verbs surface bare sentinels, not MutationError wrappers.
func TestSingleRecordVerbs_ReturnBareSentinels(t *testing.T) {
	ctx := context.Background()
	s := store.New(xdbmemory.NewDriver())

	rec := core.NewRecord("com.example", "posts", "p1")
	rec.Set("title", "Hello")
	require.NoError(t, s.CreateRecord(ctx, rec))

	err := s.CreateRecord(ctx, rec)
	require.ErrorIs(t, err, core.ErrAlreadyExists)

	var merr *store.MutationError
	assert.False(t, errors.As(err, &merr),
		"single-record verbs must not leak MutationError")
}

// TestNew_PanicsOnNilDriver pins the constructor guard.
func TestNew_PanicsOnNilDriver(t *testing.T) {
	assert.Panics(t, func() {
		store.New(nil)
	})
}
