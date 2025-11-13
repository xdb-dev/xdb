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
	AssertEqualValues(t, expected.Value(), actual.Value())
}

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

	assert.Equal(t, expected.NS().String(), actual.NS().String(), "URI: ns mismatch")
	assert.Equal(t, expected.Schema(), actual.Schema(), "URI: schema mismatch")
	assert.Truef(t, expected.ID().Equals(actual.ID()), "URI: id mismatch: %s != %s", expected.ID().String(), actual.ID().String())
	assert.Truef(t, expected.Attr().Equals(actual.Attr()), "URI: attr mismatch: %s != %s", expected.Attr().String(), actual.Attr().String())
	assert.Equal(t, expected.String(), actual.String(), "URI: string mismatch")
}

// AssertEqualValues asserts that two values are equal.
func AssertEqualValues(t *testing.T, expected, actual *core.Value) {
	t.Helper()

	assert.Equal(t, expected.Type().ID(), actual.Type().ID(), "value type mismatch")
	assert.EqualValues(t, expected.Unwrap(), actual.Unwrap(), "value mismatch")
}

// AssertSchemaDefEqual asserts that two schema definitions are equal.
func AssertSchemaDefEqual(t *testing.T, expected, actual *core.SchemaDef) {
	t.Helper()

	assert.Equal(t, expected.Name, actual.Name, "SchemaDef: name mismatch")
	assert.Equal(t, expected.Description, actual.Description, "SchemaDef: description mismatch")
	assert.Equal(t, expected.Version, actual.Version, "SchemaDef: version mismatch")
	assert.Equal(t, expected.Required, actual.Required, "SchemaDef: required fields mismatch")
	require.Len(t, actual.Fields, len(expected.Fields), "SchemaDef: fields length mismatch")

	for i, expectedField := range expected.Fields {
		actualField := actual.Fields[i]
		AssertFieldDefEqual(t, expectedField, actualField)
	}
}

// AssertFieldDefEqual asserts that two field definitions are equal.
func AssertFieldDefEqual(t *testing.T, expected, actual *core.FieldDef) {
	t.Helper()

	assert.Equal(t, expected.Name, actual.Name, "FieldDef: name mismatch")
	assert.Equal(t, expected.Description, actual.Description, "FieldDef: description mismatch")
	assert.Equal(t, expected.Type, actual.Type, "FieldDef: type mismatch")

	if expected.Default != nil {
		assert.NotNil(t, actual.Default, "FieldDef: default mismatch")
		AssertEqualValues(t, expected.Default, actual.Default)
	} else {
		assert.Nil(t, actual.Default, "FieldDef: default mismatch")
	}
}
