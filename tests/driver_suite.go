package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// DriverSuite checks mutation semantics, tuple reads, schema storage, and
// optional capabilities on raw [store.Driver] implementations. Schema
// validation and versioning are tested through the store suites.
type DriverSuite struct {
	newDriver func() store.Driver
}

// NewDriverSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh driver.
func NewDriverSuite(fn func() store.Driver) *DriverSuite {
	return &DriverSuite{newDriver: fn}
}

// Run runs all driver contract tests as subtests of t.
func (s *DriverSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("GetTuples", s.testGetTuples)
	t.Run("ScanTuples", s.testScanTuples)
	t.Run("Patch", s.testPatch)
	t.Run("Create", s.testCreate)
	t.Run("CreateRace", s.testCreateRace)
	t.Run("Put", s.testPut)
	t.Run("Delete", s.testDelete)
	t.Run("Defs", s.testDefs)
	t.Run("DropRecords", s.testDropRecords)
	t.Run("Tx", s.testTx)
}

// --- Helpers ---

// putTuples builds tuples at path from attr/value pairs.
func putTuples(path string, pairs ...any) []*core.Tuple {
	if len(pairs)%2 != 0 {
		panic("putTuples: pairs must be attr/value pairs")
	}
	tuples := make([]*core.Tuple, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		tuples = append(tuples, core.NewTuple(path, pairs[i].(string), pairs[i+1]))
	}
	return tuples
}

// mut builds a mutation at path.
func mut(path string, op store.Op, tuples ...*core.Tuple) store.Mutation {
	return store.Mutation{
		Path:   core.MustParseURI("xdb://" + path),
		Op:     op,
		Tuples: tuples,
	}
}

// scanAll collects every tuple under scope, failing the test on
// iteration errors.
func scanAll(t *testing.T, d store.Driver, scope string) []*core.Tuple {
	t.Helper()

	tuples := make([]*core.Tuple, 0, 8)
	for tuple, err := range d.ScanTuples(context.Background(), core.MustParseURI(scope)) {
		require.NoError(t, err)
		tuples = append(tuples, tuple)
	}
	return tuples
}

// recordAttrs scans a record path and returns its tuples keyed by attr.
func recordAttrs(t *testing.T, d store.Driver, path string) map[string]*core.Tuple {
	t.Helper()

	attrs := make(map[string]*core.Tuple)
	for _, tuple := range scanAll(t, d, path) {
		attrs[tuple.Attr()] = tuple
	}
	return attrs
}

// strAttr returns the string value of attr in the map, failing if absent.
func strAttr(t *testing.T, attrs map[string]*core.Tuple, attr string) string {
	t.Helper()

	tuple, ok := attrs[attr]
	require.True(t, ok, "attr %s not found", attr)
	v, err := tuple.AsStr()
	require.NoError(t, err)
	return v
}

// fakeDef builds a schema definition for testing Def CRUD. The
// revision is stored verbatim by drivers — no stamping.
func fakeDef(path string, revision int64) *schema.Def {
	return &schema.Def{
		URI:      core.MustParseURI("xdb://" + path),
		Mode:     schema.ModeStrict,
		Revision: revision,
		Fields: map[string]schema.Field{
			"title": {Type: core.TypeString, Required: true},
		},
	}
}

// --- Tuple reads ---

func (s *DriverSuite) testGetTuples(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	require.NoError(t, d.Apply(ctx,
		mut("com.example/posts/p1", store.OpPatch,
			putTuples("com.example/posts/p1", "title", "Hello", "author", "Alice")...),
	))

	t.Run("returns present tuples in request order", func(t *testing.T) {
		got, err := d.GetTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/p1#author"),
			core.MustParseURI("xdb://com.example/posts/p1#title"),
		)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "author", got[0].Attr())
		assert.Equal(t, "title", got[1].Attr())
	})

	t.Run("omits absent attrs without error", func(t *testing.T) {
		got, err := d.GetTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/p1#title"),
			core.MustParseURI("xdb://com.example/posts/p1#missing"),
		)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "title", got[0].Attr())
	})

	t.Run("absent record yields no tuples and no error", func(t *testing.T) {
		got, err := d.GetTuples(ctx,
			core.MustParseURI("xdb://com.example/posts/missing#title"),
		)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func (s *DriverSuite) testScanTuples(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	seed := []store.Mutation{
		mut("com.example/posts/p1", store.OpPatch,
			putTuples("com.example/posts/p1", "title", "One", "author", "A")...),
		mut("com.example/posts/p2", store.OpPatch,
			putTuples("com.example/posts/p2", "title", "Two", "author", "B")...),
		mut("com.example/users/u1", store.OpPatch,
			putTuples("com.example/users/u1", "name", "Alice")...),
		mut("com.other/posts/p1", store.OpPatch,
			putTuples("com.other/posts/p1", "title", "Other")...),
	}
	for _, m := range seed {
		require.NoError(t, d.Apply(ctx, m))
	}

	t.Run("record scope yields only that record", func(t *testing.T) {
		tuples := scanAll(t, d, "xdb://com.example/posts/p1")
		require.Len(t, tuples, 2)
		for _, tuple := range tuples {
			assert.Equal(t, "p1", tuple.Path().ID())
			assert.Equal(t, "posts", tuple.Path().Schema())
		}
	})

	t.Run("schema scope yields all records in schema", func(t *testing.T) {
		tuples := scanAll(t, d, "xdb://com.example/posts")
		assert.Len(t, tuples, 4)
	})

	t.Run("namespace scope yields all records in namespace", func(t *testing.T) {
		tuples := scanAll(t, d, "xdb://com.example")
		assert.Len(t, tuples, 5)
	})

	t.Run("tuples of one record are contiguous", func(t *testing.T) {
		tuples := scanAll(t, d, "xdb://com.example")

		seen := make(map[string]bool)
		var prev string
		for _, tuple := range tuples {
			path := tuple.Path().Path()
			if path != prev {
				require.False(t, seen[path],
					"record %s yielded in non-contiguous runs", path)
				seen[path] = true
				prev = path
			}
		}
	})

	t.Run("empty scope yields nothing", func(t *testing.T) {
		tuples := scanAll(t, d, "xdb://com.missing")
		assert.Empty(t, tuples)
	})
}

// --- Ops ---

func (s *DriverSuite) testPatch(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	t.Run("creates record on first merge", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/m1", store.OpPatch,
				putTuples("com.example/posts/m1", "title", "Hello")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/m1")
		assert.Equal(t, "Hello", strAttr(t, attrs, "title"))
	})

	t.Run("adds attrs and keeps existing ones", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/m1", store.OpPatch,
				putTuples("com.example/posts/m1", "author", "Alice")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/m1")
		assert.Equal(t, "Hello", strAttr(t, attrs, "title"))
		assert.Equal(t, "Alice", strAttr(t, attrs, "author"))
	})

	t.Run("overwrites existing attrs", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/m1", store.OpPatch,
				putTuples("com.example/posts/m1", "title", "Updated")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/m1")
		assert.Equal(t, "Updated", strAttr(t, attrs, "title"))
		assert.Equal(t, "Alice", strAttr(t, attrs, "author"))
	})
}

func (s *DriverSuite) testCreate(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	t.Run("writes full tuple set when absent", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/c1", store.OpCreate,
				putTuples("com.example/posts/c1", "title", "Hello", "author", "Alice")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/c1")
		require.Len(t, attrs, 2)
	})

	t.Run("fails when path has any tuples", func(t *testing.T) {
		err := d.Apply(ctx,
			mut("com.example/posts/c1", store.OpCreate,
				putTuples("com.example/posts/c1", "title", "Clobber")...),
		)
		require.ErrorIs(t, err, core.ErrAlreadyExists)

		// Drivers return bare sentinels — batch attribution is the
		// facade's job.
		var merr *store.MutationError
		require.NotErrorAs(t, err, &merr,
			"drivers must not wrap errors in MutationError")

		// The failed create must not have touched the record.
		attrs := recordAttrs(t, d, "xdb://com.example/posts/c1")
		assert.Equal(t, "Hello", strAttr(t, attrs, "title"))
	})
}

func (s *DriverSuite) testCreateRace(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	// N concurrent creates on one path: exactly one must win, and the
	// record must hold the winner's full tuple set — not a mix.
	const racers = 16

	var wg sync.WaitGroup
	errs := make([]error, racers)
	for i := range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = d.Apply(ctx,
				mut("com.example/posts/race", store.OpCreate,
					putTuples("com.example/posts/race",
						"title", "Winner",
						"winner", string(rune('a'+i)),
					)...),
			)
		}()
	}
	wg.Wait()

	var wins int
	for _, err := range errs {
		if err == nil {
			wins++
			continue
		}
		require.ErrorIs(t, err, core.ErrAlreadyExists)
	}
	assert.Equal(t, 1, wins, "exactly one concurrent create must win")

	attrs := recordAttrs(t, d, "xdb://com.example/posts/race")
	require.Len(t, attrs, 2)
	assert.Equal(t, "Winner", strAttr(t, attrs, "title"))
}

func (s *DriverSuite) testPut(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	t.Run("creates when absent", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/r1", store.OpPut,
				putTuples("com.example/posts/r1", "title", "Hello")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/r1")
		assert.Equal(t, "Hello", strAttr(t, attrs, "title"))
	})

	t.Run("replaces full tuple set when present", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/r1", store.OpPut,
				putTuples("com.example/posts/r1", "author", "Alice")...),
		))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/r1")
		require.Len(t, attrs, 1, "replace must drop attrs absent from the mutation")
		assert.Equal(t, "Alice", strAttr(t, attrs, "author"))
	})
}

func (s *DriverSuite) testDelete(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	seed := func(id string) {
		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/"+id, store.OpPut,
				putTuples("com.example/posts/"+id,
					"title", "Hello", "author", "Alice", "status", "draft")...),
		))
	}

	t.Run("removes named attrs and keeps the rest", func(t *testing.T) {
		seed("d1")

		del := mut("com.example/posts/d1", store.OpDelete)
		del.Attrs = []string{"author", "status"}
		require.NoError(t, d.Apply(ctx, del))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/d1")
		require.Len(t, attrs, 1)
		assert.Equal(t, "Hello", strAttr(t, attrs, "title"))
	})

	t.Run("empty attrs removes the whole record", func(t *testing.T) {
		seed("d2")

		require.NoError(t, d.Apply(ctx, mut("com.example/posts/d2", store.OpDelete)))

		assert.Empty(t, scanAll(t, d, "xdb://com.example/posts/d2"))
	})

	t.Run("idempotent on absent record", func(t *testing.T) {
		require.NoError(t, d.Apply(ctx, mut("com.example/posts/d-missing", store.OpDelete)))
	})

	t.Run("idempotent on absent attrs", func(t *testing.T) {
		seed("d3")

		del := mut("com.example/posts/d3", store.OpDelete)
		del.Attrs = []string{"missing"}
		require.NoError(t, d.Apply(ctx, del))

		attrs := recordAttrs(t, d, "xdb://com.example/posts/d3")
		assert.Len(t, attrs, 3)
	})

	t.Run("create succeeds after whole-record delete", func(t *testing.T) {
		seed("d4")
		require.NoError(t, d.Apply(ctx, mut("com.example/posts/d4", store.OpDelete)))

		require.NoError(t, d.Apply(ctx,
			mut("com.example/posts/d4", store.OpCreate,
				putTuples("com.example/posts/d4", "title", "Reborn")...),
		))
	})
}

// --- Defs ---

func (s *DriverSuite) testDefs(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	t.Run("create stores the def verbatim", func(t *testing.T) {
		def := fakeDef("com.example/posts", 7)
		require.NoError(t, d.CreateSchema(ctx, def))

		got, err := d.GetSchema(ctx, core.MustParseURI("xdb://com.example/posts"))
		require.NoError(t, err)
		AssertDefEqual(t, def, got)
		assert.Equal(t, int64(7), got.Revision, "drivers must not stamp revisions")
	})

	t.Run("create fails on duplicate", func(t *testing.T) {
		err := d.CreateSchema(ctx, fakeDef("com.example/posts", 1))
		require.ErrorIs(t, err, core.ErrAlreadyExists)
	})

	t.Run("get absent returns not found", func(t *testing.T) {
		_, err := d.GetSchema(ctx, core.MustParseURI("xdb://com.example/missing"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("put upserts unconditionally", func(t *testing.T) {
		fresh := fakeDef("com.example/users", 3)
		require.NoError(t, d.PutSchema(ctx, fresh))

		replaced := fakeDef("com.example/users", 9)
		require.NoError(t, d.PutSchema(ctx, replaced))

		got, err := d.GetSchema(ctx, core.MustParseURI("xdb://com.example/users"))
		require.NoError(t, err)
		assert.Equal(t, int64(9), got.Revision)
	})

	t.Run("scan with nil scope yields all defs", func(t *testing.T) {
		require.NoError(t, d.PutSchema(ctx, fakeDef("com.other/things", 1)))

		var got []*schema.Def
		for def, err := range d.ScanSchemas(ctx, nil) {
			require.NoError(t, err)
			got = append(got, def)
		}
		assert.Len(t, got, 3)
	})

	t.Run("scan with namespace scope filters", func(t *testing.T) {
		var got []*schema.Def
		for def, err := range d.ScanSchemas(ctx, core.MustParseURI("xdb://com.example")) {
			require.NoError(t, err)
			got = append(got, def)
		}
		assert.Len(t, got, 2)
	})

	t.Run("delete removes the def", func(t *testing.T) {
		require.NoError(t, d.DeleteSchema(ctx, core.MustParseURI("xdb://com.other/things")))

		_, err := d.GetSchema(ctx, core.MustParseURI("xdb://com.other/things"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})

	t.Run("delete absent returns not found", func(t *testing.T) {
		err := d.DeleteSchema(ctx, core.MustParseURI("xdb://com.other/things"))
		require.ErrorIs(t, err, core.ErrNotFound)
	})
}

func (s *DriverSuite) testDropRecords(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	require.NoError(t, d.PutSchema(ctx, fakeDef("com.example/posts", 1)))
	seed := []store.Mutation{
		mut("com.example/posts/p1", store.OpPatch,
			putTuples("com.example/posts/p1", "title", "One")...),
		mut("com.example/posts/p2", store.OpPatch,
			putTuples("com.example/posts/p2", "title", "Two")...),
		mut("com.example/users/u1", store.OpPatch,
			putTuples("com.example/users/u1", "name", "Alice")...),
	}
	for _, m := range seed {
		require.NoError(t, d.Apply(ctx, m))
	}

	t.Run("removes all records of the schema", func(t *testing.T) {
		require.NoError(t, d.DropRecords(ctx,
			core.MustParseURI("xdb://com.example/posts")))

		assert.Empty(t, scanAll(t, d, "xdb://com.example/posts"))
		assert.Len(t, scanAll(t, d, "xdb://com.example/users"), 1,
			"other schemas must be untouched")
	})

	t.Run("keeps the def itself", func(t *testing.T) {
		_, err := d.GetSchema(ctx, core.MustParseURI("xdb://com.example/posts"))
		require.NoError(t, err)
	})

	t.Run("no-op when no records exist", func(t *testing.T) {
		require.NoError(t, d.DropRecords(ctx,
			core.MustParseURI("xdb://com.example/posts")))
	})
}

// --- Optional capabilities ---

func (s *DriverSuite) testTx(t *testing.T) {
	ctx := context.Background()
	d := s.newDriver()

	txd, ok := d.(store.TxDriver)
	if !ok {
		t.Skip("driver does not implement store.TxDriver")
	}

	t.Run("commits on nil", func(t *testing.T) {
		err := txd.Tx(ctx, func(tx store.Driver) error {
			return tx.Apply(ctx,
				mut("com.example/posts/tx1", store.OpCreate,
					putTuples("com.example/posts/tx1", "title", "Committed")...),
			)
		})
		require.NoError(t, err)

		attrs := recordAttrs(t, d, "xdb://com.example/posts/tx1")
		assert.Equal(t, "Committed", strAttr(t, attrs, "title"))
	})

	t.Run("rolls back on error", func(t *testing.T) {
		sentinel := assert.AnError
		err := txd.Tx(ctx, func(tx store.Driver) error {
			applyErr := tx.Apply(ctx,
				mut("com.example/posts/tx2", store.OpCreate,
					putTuples("com.example/posts/tx2", "title", "Discarded")...),
			)
			require.NoError(t, applyErr)
			return sentinel
		})
		require.ErrorIs(t, err, sentinel)

		assert.Empty(t, scanAll(t, d, "xdb://com.example/posts/tx2"))
	})

	t.Run("reads see own writes inside tx", func(t *testing.T) {
		err := txd.Tx(ctx, func(tx store.Driver) error {
			if applyErr := tx.Apply(ctx,
				mut("com.example/posts/tx3", store.OpCreate,
					putTuples("com.example/posts/tx3", "title", "Visible")...),
			); applyErr != nil {
				return applyErr
			}

			var tuples []*core.Tuple
			for tuple, scanErr := range tx.ScanTuples(ctx,
				core.MustParseURI("xdb://com.example/posts/tx3")) {
				require.NoError(t, scanErr)
				tuples = append(tuples, tuple)
			}
			require.Len(t, tuples, 1)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("def writes roll back too", func(t *testing.T) {
		sentinel := assert.AnError
		err := txd.Tx(ctx, func(tx store.Driver) error {
			if putErr := tx.PutSchema(ctx, fakeDef("com.example/rollback", 1)); putErr != nil {
				return putErr
			}
			return sentinel
		})
		require.ErrorIs(t, err, sentinel)

		_, getErr := d.GetSchema(ctx, core.MustParseURI("xdb://com.example/rollback"))
		require.ErrorIs(t, getErr, core.ErrNotFound)
	})
}
