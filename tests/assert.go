package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/types"
)

// AssertEqualRecord asserts that two records are equal.
// It checks for the following:
// - The record keys are equal.
// - The record tuples are equal.
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

// AssertEqualTuples asserts that two lists of tuples are equal.
func AssertEqualTuples(t *testing.T, expected, actual []*types.Tuple) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "tuple lists have different lengths")

	for i, expected := range expected {
		actual := actual[i]
		AssertEqualTuple(t, expected, actual)
	}
}

// AssertEqualTuple asserts that two tuples are equal.
func AssertEqualTuple(t *testing.T, expected, actual *types.Tuple) {
	t.Helper()

	assert.Equalf(t,
		expected.Key().String(),
		actual.Key().String(),
		"tuple key mismatch",
	)
	assert.EqualValues(t,
		expected.Value().Unwrap(),
		actual.Value().Unwrap(),
		"tuple value mismatch",
	)
}
