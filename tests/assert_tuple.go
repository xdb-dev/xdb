package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xdb-dev/xdb/types"
)

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
