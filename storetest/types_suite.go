package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// TypesStoreSuite checks that stored values retain their types and data
// across backends. It uses declared schemas so backends such as xdbfs can
// recover types from their serialized representation.
//
// The suite covers scalars and arrays, including JSON object arrays.
// Standalone JSON values are not covered here.
type TypesStoreSuite struct {
	newStore func() store.Store
}

// NewTypesStoreSuite creates a new suite using the given factory.
// The factory is called before each test group to provide a fresh store.
func NewTypesStoreSuite(fn func() store.Store) *TypesStoreSuite {
	return &TypesStoreSuite{newStore: fn}
}

// Run runs all type round-trip tests as subtests of t.
func (s *TypesStoreSuite) Run(t *testing.T) {
	t.Helper()

	t.Run("Scalars", s.testScalars)
	t.Run("Arrays", s.testArrays)
}

func (s *TypesStoreSuite) testScalars(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	// A fixed whole-second UTC time: [core.Value.String] renders TIME
	// as RFC3339 (second precision), so this compares stably across
	// backends that store millis (sqlite) or RFC3339Nano (redis).
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	cases := []struct {
		value *core.Value
		typ   core.Type
		name  string
	}{
		{core.BoolVal(true), core.TypeBool, "bool"},
		{core.IntVal(-42), core.TypeInt, "int"},
		{core.UintVal(42), core.TypeUnsigned, "unsigned"},
		{core.FloatVal(3.14), core.TypeFloat, "float"},
		{core.StringVal("hello"), core.TypeString, "string"},
		{core.BytesVal([]byte("bin\x00ary")), core.TypeBytes, "bytes"},
		{core.TimeVal(ts), core.TypeTime, "time"},
	}

	uri := core.MustParseURI("xdb://com.example/types")
	fields := make(map[string]schema.Field, len(cases))
	for _, c := range cases {
		fields[c.name] = schema.Field{Type: c.typ}
	}
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:    uri,
		Mode:   schema.ModeStrict,
		Fields: fields,
	}))

	r := core.NewRecord("com.example", "types", "1")
	for _, c := range cases {
		r.Set(c.name, c.value)
	}
	require.NoError(t, st.CreateRecord(ctx, r))

	got, err := st.GetRecord(ctx, r.URI())
	require.NoError(t, err)

	gotAttrs := make(map[string]*core.Tuple)
	for _, tuple := range got.Tuples() {
		gotAttrs[tuple.Attr()] = tuple
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tuple, ok := gotAttrs[c.name]
			require.True(t, ok, "attr %s missing after round-trip", c.name)
			AssertEqualValue(t, c.value, tuple.Value())
		})
	}
}

// testArrays round-trips a typed scalar array and an object array
// (ARRAY<JSON> with per-member constraints), asserting element-wise
// equality. This is the per-backend fidelity check that the columnar
// (xdbsqlite) and def-guided (xdbfs) codecs preserve array shape and
// element types.
func (s *TypesStoreSuite) testArrays(t *testing.T) {
	ctx := context.Background()
	st := s.newStore()

	uri := core.MustParseURI("xdb://com.example/type_arrays")
	require.NoError(t, st.CreateSchema(ctx, uri, &schema.Def{
		URI:  uri,
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"tags": {Type: core.NewArrayType(core.TIDString)},
			"lines": {
				Type: core.NewArrayType(core.TIDJSON),
				Items: map[string]schema.Field{
					"sku": {Type: core.TypeString, Required: true},
					"qty": {Type: core.TypeInt},
				},
			},
		},
	}))

	r := core.NewRecord("com.example", "type_arrays", "1")
	r.Set("tags", core.ArrayVal(core.TIDString,
		core.StringVal("go"),
		core.StringVal("db"),
	))
	r.Set("lines", core.ArrayVal(core.TIDJSON,
		mustJSONElem(map[string]any{"sku": "A-1", "qty": int64(3)}),
		mustJSONElem(map[string]any{"sku": "B-2", "qty": int64(5)}),
	))
	require.NoError(t, st.CreateRecord(ctx, r))

	got, err := st.GetRecord(ctx, r.URI())
	require.NoError(t, err)
	AssertEqualRecord(t, r, got)
}
