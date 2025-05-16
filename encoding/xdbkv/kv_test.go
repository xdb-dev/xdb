package xdbkv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
)

func TestTuple_X_KV(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tuple := types.NewTuple("Test", "1", "string", "value")
		flatkey := "Test:1:string"

		encodedKey, encodedValue, err := xdbkv.EncodeTuple(tuple)
		require.NoError(t, err)
		assert.Equal(t, flatkey, string(encodedKey))

		decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
		require.NoError(t, err)
		tests.AssertEqualTuple(t, tuple, decodedTuple)
	})

	t.Run("String Array", func(t *testing.T) {
		t.Parallel()

		tuple := types.NewTuple("Test", "1", "string_array", []string{"value1", "value2"})
		flatkey := "Test:1:string_array"

		encodedKey, encodedValue, err := xdbkv.EncodeTuple(tuple)
		require.NoError(t, err)
		assert.Equal(t, flatkey, string(encodedKey))

		decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
		require.NoError(t, err)
		tests.AssertEqualTuple(t, tuple, decodedTuple)
	})

	t.Run("Integer Array", func(t *testing.T) {
		t.Parallel()

		tuple := types.NewTuple("Test", "1", "integer_array", []int64{1, 2})
		flatkey := "Test:1:integer_array"

		encodedKey, encodedValue, err := xdbkv.EncodeTuple(tuple)
		require.NoError(t, err)
		assert.Equal(t, flatkey, string(encodedKey))

		decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
		require.NoError(t, err)
		tests.AssertEqualTuple(t, tuple, decodedTuple)
	})

	t.Run("Float Array", func(t *testing.T) {
		t.Parallel()

		tuple := types.NewTuple("Test", "1", "float_array", []float64{1.1, 2.2})
		flatkey := "Test:1:float_array"

		encodedKey, encodedValue, err := xdbkv.EncodeTuple(tuple)
		require.NoError(t, err)
		assert.Equal(t, flatkey, string(encodedKey))

		decodedTuple, err := xdbkv.DecodeTuple(encodedKey, encodedValue)
		require.NoError(t, err)
		tests.AssertEqualTuple(t, tuple, decodedTuple)
	})

	t.Run("Boolean Array", func(t *testing.T) {
		t.Parallel()

	})
}
