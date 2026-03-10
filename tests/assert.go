// Package tests provides shared test helpers, assertions, and reusable
// test suites for XDB store implementations.
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// AssertEqualRecord asserts that two records have equal URIs and tuples.
func AssertEqualRecord(t *testing.T, expected, actual *core.Record) {
	t.Helper()

	require.True(t, expected.URI().Equals(actual.URI()), "record URI mismatch")

	gotTuples := make(map[string]*core.Tuple)
	for _, tuple := range actual.Tuples() {
		gotTuples[tuple.Attr().String()] = tuple
	}

	for _, tuple := range expected.Tuples() {
		attr := tuple.Attr().String()
		gotTuple, ok := gotTuples[attr]

		require.True(t, ok, "tuple %s not found", attr)
		AssertEqualTuple(t, tuple, gotTuple)
	}
}

// AssertEqualTuple asserts that two tuples have equal URIs and values.
func AssertEqualTuple(t *testing.T, expected, actual *core.Tuple) {
	t.Helper()

	assert.Equal(t,
		expected.URI().String(),
		actual.URI().String(),
		"tuple URI mismatch",
	)
	AssertEqualValue(t, expected.Value(), actual.Value())
}

// AssertEqualValue asserts that two values have equal types and data.
func AssertEqualValue(t *testing.T, expected, actual *core.Value) {
	t.Helper()

	assert.Equal(t, expected.Type().ID(), actual.Type().ID(), "value type mismatch")

	if expected.Type().ID() == core.TIDArray {
		expectedArr, err := expected.AsArray()
		require.NoError(t, err)
		actualArr, err := actual.AsArray()
		require.NoError(t, err)

		require.Len(t, actualArr, len(expectedArr), "array length mismatch")
		for i, expectedVal := range expectedArr {
			AssertEqualValue(t, expectedVal, actualArr[i])
		}

		return
	}

	assert.Equal(t, expected.String(), actual.String(), "value mismatch")
}

// AssertDefEqual asserts that two schema definitions are equal.
func AssertDefEqual(t *testing.T, expected, actual *schema.Def) {
	t.Helper()

	assert.True(t, expected.URI.Equals(actual.URI), "Def: URI mismatch")
	assert.Equal(t, expected.Mode, actual.Mode, "Def: mode mismatch")
	require.Len(t, actual.Fields, len(expected.Fields), "Def: fields length mismatch")

	for name, expectedField := range expected.Fields {
		actualField, ok := actual.Fields[name]
		require.True(t, ok, "Def: field %s not found", name)
		assert.Equal(t, expectedField.Type, actualField.Type, "FieldDef: type mismatch for %s", name)
		assert.Equal(t, expectedField.Required, actualField.Required, "FieldDef: required mismatch for %s", name)
	}
}
