package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestTypeID(t *testing.T) {
	t.Parallel()

	t.Run("String Representation", func(t *testing.T) {
		testcases := []struct {
			typeID   core.TID
			expected string
		}{
			{core.TypeIDUnknown, "UNKNOWN"},
			{core.TypeIDBoolean, "BOOLEAN"},
			{core.TypeIDInteger, "INTEGER"},
			{core.TypeIDUnsigned, "UNSIGNED"},
			{core.TypeIDFloat, "FLOAT"},
			{core.TypeIDString, "STRING"},
			{core.TypeIDBytes, "BYTES"},
			{core.TypeIDTime, "TIME"},
			{core.TypeIDArray, "ARRAY"},
			{core.TypeIDMap, "MAP"},
		}

		for _, tc := range testcases {
			t.Run(tc.expected, func(t *testing.T) {
				assert.Equal(t, tc.expected, tc.typeID.String())
			})
		}
	})
}

func TestParseType(t *testing.T) {
	t.Parallel()

	t.Run("Valid Types", func(t *testing.T) {
		testcases := []struct {
			input    string
			expected core.TID
		}{
			{"BOOLEAN", core.TypeIDBoolean},
			{"INTEGER", core.TypeIDInteger},
			{"UNSIGNED", core.TypeIDUnsigned},
			{"FLOAT", core.TypeIDFloat},
			{"STRING", core.TypeIDString},
			{"BYTES", core.TypeIDBytes},
			{"TIME", core.TypeIDTime},
			{"ARRAY", core.TypeIDArray},
			{"MAP", core.TypeIDMap},
			{"boolean", core.TypeIDBoolean}, // Case insensitive
			{"Integer", core.TypeIDInteger}, // Mixed case
			{"  FLOAT  ", core.TypeIDFloat}, // With whitespace
		}

		for _, tc := range testcases {
			t.Run(tc.input, func(t *testing.T) {
				result, err := core.ParseType(tc.input)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Invalid Types", func(t *testing.T) {
		testcases := []struct {
			input    string
			expected string
		}{
			{"INVALID", "unknown type"},
			{"", "unknown type"},
			{"BOOLEANX", "unknown type"},
			{"123", "unknown type"},
			{"!@#$", "unknown type"},
		}

		for _, tc := range testcases {
			t.Run(tc.input, func(t *testing.T) {
				result, err := core.ParseType(tc.input)
				assert.Error(t, err)
				assert.Equal(t, core.TypeIDUnknown, result)
				assert.Contains(t, err.Error(), tc.expected)
			})
		}
	})

	t.Run("UNKNOWN Type", func(t *testing.T) {
		// UNKNOWN is actually a valid type in the system
		result, err := core.ParseType("UNKNOWN")
		assert.NoError(t, err)
		assert.Equal(t, core.TypeIDUnknown, result)
	})
}

func TestType(t *testing.T) {
	t.Parallel()

	t.Run("Basic Type Creation", func(t *testing.T) {
		typ := core.TypeBool
		assert.Equal(t, core.TypeIDBoolean, typ.ID())
		assert.Equal(t, "BOOLEAN", typ.String())
		assert.Equal(t, core.TypeIDUnknown, typ.KeyTypeID())
		assert.Equal(t, core.TypeIDUnknown, typ.ValueTypeID())
	})

	t.Run("Array Type Creation", func(t *testing.T) {
		typ := core.NewArrayType(core.TypeIDString)
		assert.Equal(t, core.TypeIDArray, typ.ID())
		assert.Equal(t, "ARRAY", typ.String())
		assert.Equal(t, core.TypeIDUnknown, typ.KeyTypeID())
		assert.Equal(t, core.TypeIDString, typ.ValueTypeID())
	})

	t.Run("Map Type Creation", func(t *testing.T) {
		typ := core.NewMapType(core.TypeIDString, core.TypeIDInteger)
		assert.Equal(t, core.TypeIDMap, typ.ID())
		assert.Equal(t, "MAP", typ.String())
		assert.Equal(t, core.TypeIDString, typ.KeyTypeID())
		assert.Equal(t, core.TypeIDInteger, typ.ValueTypeID())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Unknown Type", func(t *testing.T) {
			typ := core.TypeUnknown
			assert.Equal(t, core.TypeIDUnknown, typ.ID())
			assert.Equal(t, "UNKNOWN", typ.String())
		})

		t.Run("Array with Unknown Value Type", func(t *testing.T) {
			typ := core.NewArrayType(core.TypeIDUnknown)
			assert.Equal(t, core.TypeIDArray, typ.ID())
			assert.Equal(t, core.TypeIDUnknown, typ.ValueTypeID())
		})

		t.Run("Map with Unknown Types", func(t *testing.T) {
			typ := core.NewMapType(core.TypeIDUnknown, core.TypeIDUnknown)
			assert.Equal(t, core.TypeIDMap, typ.ID())
			assert.Equal(t, core.TypeIDUnknown, typ.KeyTypeID())
			assert.Equal(t, core.TypeIDUnknown, typ.ValueTypeID())
		})

		t.Run("Nested Array Types", func(t *testing.T) {
			// Array of arrays
			innerArrayType := core.NewArrayType(core.TypeIDString)
			outerArrayType := core.NewArrayType(innerArrayType.ID())
			assert.Equal(t, core.TypeIDArray, outerArrayType.ID())
			assert.Equal(t, core.TypeIDArray, outerArrayType.ValueTypeID())
		})

		t.Run("Complex Map Types", func(t *testing.T) {
			// Map with array values
			mapType := core.NewMapType(core.TypeIDString, core.TypeIDArray)
			assert.Equal(t, core.TypeIDMap, mapType.ID())
			assert.Equal(t, core.TypeIDString, mapType.KeyTypeID())
			assert.Equal(t, core.TypeIDArray, mapType.ValueTypeID())
		})
	})
}

func TestType_Equals(t *testing.T) {
	t.Parallel()

	t.Run("Scalar Types Equality", func(t *testing.T) {
		type1 := core.TypeBool
		type2 := core.TypeBool
		type3 := core.TypeInt

		assert.True(t, type1.Equals(type2))
		assert.False(t, type1.Equals(type3))
	})

	t.Run("Array Types Equality", func(t *testing.T) {
		arr1 := core.NewArrayType(core.TypeIDString)
		arr2 := core.NewArrayType(core.TypeIDString)
		arr3 := core.NewArrayType(core.TypeIDInteger)

		assert.True(t, arr1.Equals(arr2))
		assert.False(t, arr1.Equals(arr3))
	})

	t.Run("Map Types Equality", func(t *testing.T) {
		map1 := core.NewMapType(core.TypeIDString, core.TypeIDInteger)
		map2 := core.NewMapType(core.TypeIDString, core.TypeIDInteger)
		map3 := core.NewMapType(core.TypeIDString, core.TypeIDString)
		map4 := core.NewMapType(core.TypeIDInteger, core.TypeIDInteger)

		assert.True(t, map1.Equals(map2))
		assert.False(t, map1.Equals(map3)) // Different value type
		assert.False(t, map1.Equals(map4)) // Different key type
	})

	t.Run("Different Type Kinds", func(t *testing.T) {
		scalar := core.TypeString
		array := core.NewArrayType(core.TypeIDString)
		mapType := core.NewMapType(core.TypeIDString, core.TypeIDString)

		assert.False(t, scalar.Equals(array))
		assert.False(t, scalar.Equals(mapType))
		assert.False(t, array.Equals(mapType))
	})

	t.Run("Complex Nested Types", func(t *testing.T) {
		// Array of strings
		arr1 := core.NewArrayType(core.TypeIDString)
		// Array of integers
		arr2 := core.NewArrayType(core.TypeIDInteger)

		assert.False(t, arr1.Equals(arr2))

		// Map with string keys and string values
		map1 := core.NewMapType(core.TypeIDString, core.TypeIDString)
		// Map with string keys and integer values
		map2 := core.NewMapType(core.TypeIDString, core.TypeIDInteger)

		assert.False(t, map1.Equals(map2))
	})

	t.Run("Unknown Types", func(t *testing.T) {
		unknown1 := core.TypeUnknown
		unknown2 := core.TypeUnknown

		assert.True(t, unknown1.Equals(unknown2))
	})
}

func TestType_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Type Comparison", func(t *testing.T) {
		t.Run("Same Types", func(t *testing.T) {
			typ1 := core.TypeBool
			typ2 := core.TypeBool
			assert.Equal(t, typ1.ID(), typ2.ID())
			assert.Equal(t, typ1.String(), typ2.String())
		})

		t.Run("Different Types", func(t *testing.T) {
			typ1 := core.TypeBool
			typ2 := core.TypeInt
			assert.NotEqual(t, typ1.ID(), typ2.ID())
			assert.NotEqual(t, typ1.String(), typ2.String())
		})

		t.Run("Array vs Scalar", func(t *testing.T) {
			scalarType := core.TypeString
			arrayType := core.NewArrayType(core.TypeIDString)
			assert.NotEqual(t, scalarType.ID(), arrayType.ID())
			assert.Equal(t, core.TypeIDString, arrayType.ValueTypeID())
		})
	})

	t.Run("ParseType Edge Cases", func(t *testing.T) {
		t.Run("Whitespace Handling", func(t *testing.T) {
			testcases := []string{
				"  BOOLEAN  ",
				"\tINTEGER\t",
				"\nFLOAT\n",
				"  \t  STRING  \t  ",
			}

			for _, input := range testcases {
				t.Run(input, func(t *testing.T) {
					result, err := core.ParseType(input)
					assert.NoError(t, err)
					assert.NotEqual(t, core.TypeIDUnknown, result)
				})
			}
		})

		t.Run("Case Sensitivity", func(t *testing.T) {
			testcases := []struct {
				input    string
				expected core.TID
			}{
				{"boolean", core.TypeIDBoolean},
				{"BOOLEAN", core.TypeIDBoolean},
				{"Boolean", core.TypeIDBoolean},
				{"bOOLEAN", core.TypeIDBoolean},
			}

			for _, tc := range testcases {
				t.Run(tc.input, func(t *testing.T) {
					result, err := core.ParseType(tc.input)
					assert.NoError(t, err)
					assert.Equal(t, tc.expected, result)
				})
			}
		})

		t.Run("Special Characters", func(t *testing.T) {
			testcases := []string{
				"BOOLEAN!",
				"INTEGER@",
				"FLOAT#",
				"STRING$",
			}

			for _, input := range testcases {
				t.Run(input, func(t *testing.T) {
					result, err := core.ParseType(input)
					assert.Error(t, err)
					assert.Equal(t, core.TypeIDUnknown, result)
				})
			}
		})

		t.Run("Whitespace Edge Cases", func(t *testing.T) {
			testcases := []string{
				"BOOLEAN ",
				" INTEGER",
			}

			for _, input := range testcases {
				t.Run(input, func(t *testing.T) {
					// These should be trimmed and work
					result, err := core.ParseType(input)
					assert.NoError(t, err)
					assert.NotEqual(t, core.TypeIDUnknown, result)
				})
			}
		})
	})
}
