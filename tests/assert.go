// Package tests provides shared test helpers and assertions for XDB.
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

// AssertEqualRecords asserts that two lists of records are equal.
func AssertEqualRecords(t *testing.T, expected, actual []*core.Record) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "record lists have different lengths")

	for i, expected := range expected {
		actual := actual[i]
		AssertEqualRecord(t, expected, actual)
	}
}

// AssertEqualRecord asserts that two records are equal.
// It checks for the following:
// - The record URIs are equal.
// - The record tuples are equal.
func AssertEqualRecord(t *testing.T, expected, actual *core.Record) {
	t.Helper()

	require.EqualValues(t,
		expected.URI(),
		actual.URI(),
		"record URI mismatch",
	)

	gotTuples := make(map[string]*core.Tuple)
	for _, tuple := range actual.Tuples() {
		gotTuples[tuple.URI().String()] = tuple
	}

	for _, tuple := range expected.Tuples() {
		gotTuple, ok := gotTuples[tuple.URI().String()]

		require.True(t, ok, "tuple %s not found", tuple.URI().String())
		AssertEqualTuple(t, tuple, gotTuple)
	}
}

// AssertEqualTuples asserts that two lists of tuples are equal.
func AssertEqualTuples(t *testing.T, expected, actual []*core.Tuple) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "tuple lists have different lengths")

	for i, expected := range expected {
		actual := actual[i]
		AssertEqualTuple(t, expected, actual)
	}
}

// AssertEqualTuple asserts that two tuples are equal.
func AssertEqualTuple(t *testing.T, expected, actual *core.Tuple) {
	t.Helper()

	assert.Equalf(t,
		expected.URI().String(),
		actual.URI().String(),
		"tuple URI mismatch: %s",
		expected.URI().String(),
	)
	assert.EqualValuesf(t,
		expected.Value(),
		actual.Value(),
		"tuple value mismatch: %s",
		expected.URI().String(),
	)
}

// AssertEqualKeys asserts that two lists of keys are equal.
// Deprecated: Use AssertEqualURIs instead. Removed due to core.Key no longer existing.
// func AssertEqualKeys(t *testing.T, expected, actual []*core.Key) {
// 	t.Helper()
//
// 	require.Equal(t, len(expected), len(actual), "key lists have different lengths")
//
// 	for i, expected := range expected {
// 		actual := actual[i]
// 		AssertEqualKey(t, expected, actual)
// 	}
// }

// AssertEqualKey asserts that two keys are equal.
// Deprecated: Use AssertEqualURI instead. Removed due to core.Key no longer existing.
// func AssertEqualKey(t *testing.T, expected, actual *core.Key) {
// 	t.Helper()
//
// 	assert.Equal(t, expected.String(), actual.String(), "key mismatch")
// }

// AssertEqualURIs asserts that two lists of URIs are equal.
func AssertEqualURIs(t *testing.T, expected, actual []*core.URI) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "URI lists have different lengths")

	for i, expected := range expected {
		actual := actual[i]
		AssertEqualURI(t, expected, actual)
	}
}

// AssertEqualURI asserts that two URIs are equal.
func AssertEqualURI(t *testing.T, expected, actual *core.URI) {
	t.Helper()

	assert.Equal(t, expected.Repo(), actual.Repo(), "URI: repo mismatch")
	assert.Truef(t, expected.ID().Equals(actual.ID()), "URI: id mismatch: %s != %s", expected.ID().String(), actual.ID().String())
	assert.Truef(t, expected.Attr().Equals(actual.Attr()), "URI: attr mismatch: %s != %s", expected.Attr().String(), actual.Attr().String())
	assert.Equal(t, expected.String(), actual.String(), "URI: string mismatch")
}

// AssertEqualValues asserts that two values are equal.
func AssertEqualValues(t *testing.T, expected, actual any) {
	t.Helper()

	ev := core.NewValue(expected)
	av := core.NewValue(actual)

	assert.Equal(t, ev.Type(), av.Type(), "value type mismatch")

	// Handle maps specially since map keys are *core.Value pointers
	// and can't be compared using reflect.DeepEqual
	if ev != nil && av != nil && ev.Type().ID() == core.TypeIDMap {
		assertEqualMapValues(t, ev, av)
		return
	}

	assert.EqualValues(t, ev, av, "value mismatch")
}

// assertEqualMapValues asserts that two map values are equal by comparing
// each key-value pair individually.
func assertEqualMapValues(t *testing.T, expected, actual *core.Value) {
	t.Helper()

	expectedMap := expected.Unwrap().(map[*core.Value]*core.Value)
	actualMap := actual.Unwrap().(map[*core.Value]*core.Value)

	require.Equal(t, len(expectedMap), len(actualMap), "map length mismatch")

	// Build a lookup map for actual key-value pairs using string representation of keys
	actualLookup := make(map[string]struct {
		key   *core.Value
		value *core.Value
	})
	for k, v := range actualMap {
		actualLookup[k.String()] = struct {
			key   *core.Value
			value *core.Value
		}{key: k, value: v}
	}

	// Compare each expected key-value pair
	for expectedKey, expectedValue := range expectedMap {
		keyStr := expectedKey.String()
		actualPair, ok := actualLookup[keyStr]
		require.True(t, ok, "key %s not found in actual map", keyStr)

		// Compare the key types
		assert.Equal(t, expectedKey.Type(), actualPair.key.Type(), "key type mismatch for %s", keyStr)

		// Recursively compare values
		if expectedValue != nil && actualPair.value != nil &&
			expectedValue.Type().ID() == core.TypeIDMap {
			assertEqualMapValues(t, expectedValue, actualPair.value)
		} else {
			assert.EqualValues(t, expectedValue, actualPair.value, "value mismatch for key %s", keyStr)
		}
	}
}

// AssertSchemaEqual asserts that two schemas are equal.
func AssertSchemaEqual(t *testing.T, expected, actual *core.Schema) {
	t.Helper()

	assert.Equal(t, expected.Description, actual.Description, "schema description mismatch")
	assert.Equal(t, expected.Version, actual.Version, "schema version mismatch")
	assert.Equal(t, expected.Required, actual.Required, "schema required fields mismatch")
	require.Len(t, actual.Fields, len(expected.Fields), "schema fields length mismatch")

	for i, expectedField := range expected.Fields {
		actualField := actual.Fields[i]
		assert.Equal(t, expectedField.Name, actualField.Name,
			"field[%d] name mismatch", i)
		assert.Equal(t, expectedField.Description, actualField.Description,
			"field[%d] description mismatch", i)
		assert.True(t, expectedField.Type.Equals(actualField.Type),
			"field[%d] (%s): expected type %v, got %v",
			i, expectedField.Name, expectedField.Type, actualField.Type)
	}
}
