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
			{core.TIDUnknown, "UNKNOWN"},
			{core.TIDBoolean, "BOOLEAN"},
			{core.TIDInteger, "INTEGER"},
			{core.TIDUnsigned, "UNSIGNED"},
			{core.TIDFloat, "FLOAT"},
			{core.TIDString, "STRING"},
			{core.TIDBytes, "BYTES"},
			{core.TIDTime, "TIME"},
			{core.TIDArray, "ARRAY"},
			{core.TIDMap, "MAP"},
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
			{"BOOLEAN", core.TIDBoolean},
			{"INTEGER", core.TIDInteger},
			{"UNSIGNED", core.TIDUnsigned},
			{"FLOAT", core.TIDFloat},
			{"STRING", core.TIDString},
			{"BYTES", core.TIDBytes},
			{"TIME", core.TIDTime},
			{"ARRAY", core.TIDArray},
			{"MAP", core.TIDMap},
			{"boolean", core.TIDBoolean}, // Case insensitive
			{"Integer", core.TIDInteger}, // Mixed case
			{"  FLOAT  ", core.TIDFloat}, // With whitespace
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
				assert.Equal(t, core.TIDUnknown, result)
				assert.Contains(t, err.Error(), tc.expected)
			})
		}
	})

	t.Run("UNKNOWN Type", func(t *testing.T) {
		// UNKNOWN is actually a valid type in the system
		result, err := core.ParseType("UNKNOWN")
		assert.NoError(t, err)
		assert.Equal(t, core.TIDUnknown, result)
	})
}

func TestType(t *testing.T) {
	t.Parallel()

	t.Run("Basic Type Creation", func(t *testing.T) {
		typ := core.TypeBool
		assert.Equal(t, core.TIDBoolean, typ.ID())
		assert.Equal(t, "BOOLEAN", typ.String())
		assert.Equal(t, core.TIDUnknown, typ.KeyTypeID())
		assert.Equal(t, core.TIDUnknown, typ.ValueTypeID())
	})

	t.Run("Array Type Creation", func(t *testing.T) {
		typ := core.NewArrayType(core.TIDString)
		assert.Equal(t, core.TIDArray, typ.ID())
		assert.Equal(t, "ARRAY", typ.String())
		assert.Equal(t, core.TIDUnknown, typ.KeyTypeID())
		assert.Equal(t, core.TIDString, typ.ValueTypeID())
	})

	t.Run("Map Type Creation", func(t *testing.T) {
		typ := core.NewMapType(core.TIDString, core.TIDInteger)
		assert.Equal(t, core.TIDMap, typ.ID())
		assert.Equal(t, "MAP", typ.String())
		assert.Equal(t, core.TIDString, typ.KeyTypeID())
		assert.Equal(t, core.TIDInteger, typ.ValueTypeID())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Unknown Type", func(t *testing.T) {
			typ := core.TypeUnknown
			assert.Equal(t, core.TIDUnknown, typ.ID())
			assert.Equal(t, "UNKNOWN", typ.String())
		})

		t.Run("Array with Unknown Value Type", func(t *testing.T) {
			typ := core.NewArrayType(core.TIDUnknown)
			assert.Equal(t, core.TIDArray, typ.ID())
			assert.Equal(t, core.TIDUnknown, typ.ValueTypeID())
		})

		t.Run("Map with Unknown Types", func(t *testing.T) {
			typ := core.NewMapType(core.TIDUnknown, core.TIDUnknown)
			assert.Equal(t, core.TIDMap, typ.ID())
			assert.Equal(t, core.TIDUnknown, typ.KeyTypeID())
			assert.Equal(t, core.TIDUnknown, typ.ValueTypeID())
		})

		t.Run("Nested Array Types", func(t *testing.T) {
			// Array of arrays
			innerArrayType := core.NewArrayType(core.TIDString)
			outerArrayType := core.NewArrayType(innerArrayType.ID())
			assert.Equal(t, core.TIDArray, outerArrayType.ID())
			assert.Equal(t, core.TIDArray, outerArrayType.ValueTypeID())
		})

		t.Run("Complex Map Types", func(t *testing.T) {
			// Map with array values
			mapType := core.NewMapType(core.TIDString, core.TIDArray)
			assert.Equal(t, core.TIDMap, mapType.ID())
			assert.Equal(t, core.TIDString, mapType.KeyTypeID())
			assert.Equal(t, core.TIDArray, mapType.ValueTypeID())
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
		arr1 := core.NewArrayType(core.TIDString)
		arr2 := core.NewArrayType(core.TIDString)
		arr3 := core.NewArrayType(core.TIDInteger)

		assert.True(t, arr1.Equals(arr2))
		assert.False(t, arr1.Equals(arr3))
	})

	t.Run("Map Types Equality", func(t *testing.T) {
		map1 := core.NewMapType(core.TIDString, core.TIDInteger)
		map2 := core.NewMapType(core.TIDString, core.TIDInteger)
		map3 := core.NewMapType(core.TIDString, core.TIDString)
		map4 := core.NewMapType(core.TIDInteger, core.TIDInteger)

		assert.True(t, map1.Equals(map2))
		assert.False(t, map1.Equals(map3)) // Different value type
		assert.False(t, map1.Equals(map4)) // Different key type
	})

	t.Run("Different Type Kinds", func(t *testing.T) {
		scalar := core.TypeString
		array := core.NewArrayType(core.TIDString)
		mapType := core.NewMapType(core.TIDString, core.TIDString)

		assert.False(t, scalar.Equals(array))
		assert.False(t, scalar.Equals(mapType))
		assert.False(t, array.Equals(mapType))
	})

	t.Run("Complex Nested Types", func(t *testing.T) {
		// Array of strings
		arr1 := core.NewArrayType(core.TIDString)
		// Array of integers
		arr2 := core.NewArrayType(core.TIDInteger)

		assert.False(t, arr1.Equals(arr2))

		// Map with string keys and string values
		map1 := core.NewMapType(core.TIDString, core.TIDString)
		// Map with string keys and integer values
		map2 := core.NewMapType(core.TIDString, core.TIDInteger)

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
			arrayType := core.NewArrayType(core.TIDString)
			assert.NotEqual(t, scalarType.ID(), arrayType.ID())
			assert.Equal(t, core.TIDString, arrayType.ValueTypeID())
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
					assert.NotEqual(t, core.TIDUnknown, result)
				})
			}
		})

		t.Run("Case Sensitivity", func(t *testing.T) {
			testcases := []struct {
				input    string
				expected core.TID
			}{
				{"boolean", core.TIDBoolean},
				{"BOOLEAN", core.TIDBoolean},
				{"Boolean", core.TIDBoolean},
				{"bOOLEAN", core.TIDBoolean},
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
					assert.Equal(t, core.TIDUnknown, result)
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
					assert.NotEqual(t, core.TIDUnknown, result)
				})
			}
		})
	})
}
