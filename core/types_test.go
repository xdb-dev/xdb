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
			typeID   core.TypeID
			expected string
		}{
			{core.TypeIDUnknown, "TypeID(UNKNOWN)"},
			{core.TypeIDBoolean, "TypeID(BOOLEAN)"},
			{core.TypeIDInteger, "TypeID(INTEGER)"},
			{core.TypeIDUnsigned, "TypeID(UNSIGNED)"},
			{core.TypeIDFloat, "TypeID(FLOAT)"},
			{core.TypeIDString, "TypeID(STRING)"},
			{core.TypeIDBytes, "TypeID(BYTES)"},
			{core.TypeIDTime, "TypeID(TIME)"},
			{core.TypeIDArray, "TypeID(ARRAY)"},
			{core.TypeIDMap, "TypeID(MAP)"},
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
			expected core.TypeID
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
		typ := core.NewType(core.TypeIDBoolean)
		assert.Equal(t, core.TypeIDBoolean, typ.ID())
		assert.Equal(t, "BOOLEAN", typ.Name())
		assert.Equal(t, core.TypeIDUnknown, typ.KeyType())
		assert.Equal(t, core.TypeIDUnknown, typ.ValueType())
	})

	t.Run("Array Type Creation", func(t *testing.T) {
		typ := core.NewArrayType(core.TypeIDString)
		assert.Equal(t, core.TypeIDArray, typ.ID())
		assert.Equal(t, "ARRAY", typ.Name())
		assert.Equal(t, core.TypeIDUnknown, typ.KeyType())
		assert.Equal(t, core.TypeIDString, typ.ValueType())
	})

	t.Run("Map Type Creation", func(t *testing.T) {
		typ := core.NewMapType(core.TypeIDString, core.TypeIDInteger)
		assert.Equal(t, core.TypeIDMap, typ.ID())
		assert.Equal(t, "MAP", typ.Name())
		assert.Equal(t, core.TypeIDString, typ.KeyType())
		assert.Equal(t, core.TypeIDInteger, typ.ValueType())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Unknown Type", func(t *testing.T) {
			typ := core.NewType(core.TypeIDUnknown)
			assert.Equal(t, core.TypeIDUnknown, typ.ID())
			assert.Equal(t, "UNKNOWN", typ.Name())
		})

		t.Run("Array with Unknown Value Type", func(t *testing.T) {
			typ := core.NewArrayType(core.TypeIDUnknown)
			assert.Equal(t, core.TypeIDArray, typ.ID())
			assert.Equal(t, core.TypeIDUnknown, typ.ValueType())
		})

		t.Run("Map with Unknown Types", func(t *testing.T) {
			typ := core.NewMapType(core.TypeIDUnknown, core.TypeIDUnknown)
			assert.Equal(t, core.TypeIDMap, typ.ID())
			assert.Equal(t, core.TypeIDUnknown, typ.KeyType())
			assert.Equal(t, core.TypeIDUnknown, typ.ValueType())
		})

		t.Run("Nested Array Types", func(t *testing.T) {
			// Array of arrays
			innerArrayType := core.NewArrayType(core.TypeIDString)
			outerArrayType := core.NewArrayType(innerArrayType.ID())
			assert.Equal(t, core.TypeIDArray, outerArrayType.ID())
			assert.Equal(t, core.TypeIDArray, outerArrayType.ValueType())
		})

		t.Run("Complex Map Types", func(t *testing.T) {
			// Map with array values
			mapType := core.NewMapType(core.TypeIDString, core.TypeIDArray)
			assert.Equal(t, core.TypeIDMap, mapType.ID())
			assert.Equal(t, core.TypeIDString, mapType.KeyType())
			assert.Equal(t, core.TypeIDArray, mapType.ValueType())
		})
	})
}

func TestType_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Type Comparison", func(t *testing.T) {
		t.Run("Same Types", func(t *testing.T) {
			typ1 := core.NewType(core.TypeIDBoolean)
			typ2 := core.NewType(core.TypeIDBoolean)
			assert.Equal(t, typ1.ID(), typ2.ID())
			assert.Equal(t, typ1.Name(), typ2.Name())
		})

		t.Run("Different Types", func(t *testing.T) {
			typ1 := core.NewType(core.TypeIDBoolean)
			typ2 := core.NewType(core.TypeIDInteger)
			assert.NotEqual(t, typ1.ID(), typ2.ID())
			assert.NotEqual(t, typ1.Name(), typ2.Name())
		})

		t.Run("Array vs Scalar", func(t *testing.T) {
			scalarType := core.NewType(core.TypeIDString)
			arrayType := core.NewArrayType(core.TypeIDString)
			assert.NotEqual(t, scalarType.ID(), arrayType.ID())
			assert.Equal(t, core.TypeIDString, arrayType.ValueType())
		})
	})

	t.Run("Type Information Edge Cases", func(t *testing.T) {
		t.Run("All Type IDs", func(t *testing.T) {
			typeIDs := []core.TypeID{
				core.TypeIDUnknown,
				core.TypeIDBoolean,
				core.TypeIDInteger,
				core.TypeIDUnsigned,
				core.TypeIDFloat,
				core.TypeIDString,
				core.TypeIDBytes,
				core.TypeIDTime,
				core.TypeIDArray,
				core.TypeIDMap,
			}

			for _, typeID := range typeIDs {
				typ := core.NewType(typeID)
				assert.Equal(t, typeID, typ.ID())
				assert.NotEmpty(t, typ.Name())
			}
		})

		t.Run("Boundary Type IDs", func(t *testing.T) {
			// Test with invalid type ID (beyond known range)
			invalidTypeID := core.TypeID(999)
			typ := core.NewType(invalidTypeID)
			assert.Equal(t, invalidTypeID, typ.ID())
			// Should return empty string for unknown type
			assert.Equal(t, "", typ.Name())
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
				expected core.TypeID
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
