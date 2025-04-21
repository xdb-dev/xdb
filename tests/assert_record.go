package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/types"
)

func AssertEqualRecord(t *testing.T, expected, actual *types.Record) {
	t.Helper()

	require.EqualValues(t,
		expected.Key(),
		actual.Key(),
		"record key mismatch",
	)

	gotTuples := make(map[string]*types.Tuple)
	for _, tuple := range actual.Tuples() {
		gotTuples[tuple.Key().String()] = tuple
	}

	for _, tuple := range expected.Tuples() {
		gotTuple, ok := gotTuples[tuple.Key().String()]

		require.True(t, ok, "tuple %s not found", tuple.Key().String())
		AssertEqualTuple(t, tuple, gotTuple)
	}
}
