package tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// RoundTrip describes a single import→write→read→decode conformance case for
// the shared importer harness.
//
// Def is the schema an importer produced. Marshal and Unmarshal are that
// importer's typed encode/decode functions. Value is the original Go value; the
// harness asserts the decoded value equals it.
//
// The three importer packages (encoding/xdbstruct, schema/protoimport,
// schema/jsonschemaimport) reuse [RunRoundTrip] with their own
// Marshal/Unmarshal closures, so the round-trip contract is exercised
// identically for every source format.
type RoundTrip struct {
	// Def is the schema produced by the importer, including its URI.
	Def *schema.Def
	// Value is the original value to round-trip.
	Value any
	// Marshal converts a value into a record (typically closes over the URI).
	Marshal func(v any) (*core.Record, error)
	// Unmarshal decodes a record into dst, a non-nil pointer to a fresh value.
	Unmarshal func(rec *core.Record, dst any) error
}

// RunRoundTrip runs
// Def→CreateSchema→Marshal→CreateRecord→GetRecord→Unmarshal against a fresh
// in-memory store and asserts the decoded value equals the original.
//
// The comparison is on the decoded Go value (require.Equal), never on stored
// bytes: TIME and BYTES have no canonical JSON form, so a byte compare would be
// flaky.
func RunRoundTrip(t *testing.T, rc RoundTrip) {
	t.Helper()

	ctx := context.Background()
	st := store.New(xdbmemory.NewDriver())

	require.NoError(t, st.CreateSchema(ctx, rc.Def.URI, rc.Def))

	rec, err := rc.Marshal(rc.Value)
	require.NoError(t, err)

	require.NoError(t, st.CreateRecord(ctx, rec))

	got, err := st.GetRecord(ctx, rec.URI())
	require.NoError(t, err)

	dst := reflect.New(reflect.TypeOf(rc.Value))
	require.NoError(t, rc.Unmarshal(got, dst.Interface()))

	require.Equal(t, rc.Value, dst.Elem().Interface())
}
