package xdbjson_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
)

var defaultEncoder = xdbjson.NewDefaultEncoder()

func TestEncoder_BasicEncoding(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	data, err := defaultEncoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "123", m["_id"])
	assert.Equal(t, "John Doe", m["name"])
	assert.Equal(t, "john@example.com", m["email"])
	assert.NotContains(t, m, "_ns")
	assert.NotContains(t, m, "_schema")
}

func TestEncoder_WithMetadata(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	encoder := xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})

	data, err := encoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "123", m["_id"])
	assert.Equal(t, "com.example", m["_ns"])
	assert.Equal(t, "users", m["_schema"])
	assert.Equal(t, "John Doe", m["name"])
}

func TestEncoder_CustomFieldNames(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	encoder := xdbjson.NewEncoder(xdbjson.Options{
		IDField:       "userId",
		NSField:       "namespace",
		SchemaField:   "type",
		IncludeNS:     true,
		IncludeSchema: true,
	})

	data, err := encoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "123", m["userId"])
	assert.Equal(t, "com.example", m["namespace"])
	assert.Equal(t, "users", m["type"])
	assert.NotContains(t, m, "_id")
	assert.NotContains(t, m, "_ns")
	assert.NotContains(t, m, "_schema")
}

func TestEncoder_NestedAttributes(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.city", "Boston").
		Set("address.location.lat", 42.3601).
		Set("address.location.lon", -71.0589)

	data, err := defaultEncoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	address, ok := m["address"].(map[string]any)
	require.True(t, ok, "address should be a nested object")

	assert.Equal(t, "123 Main St", address["street"])
	assert.Equal(t, "Boston", address["city"])

	location, ok := address["location"].(map[string]any)
	require.True(t, ok, "location should be a nested object")

	assert.Equal(t, 42.3601, location["lat"])
	assert.Equal(t, -71.0589, location["lon"])
}

func TestEncoder_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		attr     string
		value    any
		expected any
	}{
		{"boolean_true", "bool_val", true, true},
		{"boolean_false", "bool_val", false, false},
		{"integer", "int_val", int64(42), float64(42)},
		{"negative_integer", "int_val", int64(-100), float64(-100)},
		{"unsigned", "uint_val", uint64(100), float64(100)},
		{"float", "float_val", 3.14159, 3.14159},
		{"string", "str_val", "hello", "hello"},
		{"empty_string", "str_val", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord("com.example", "test", "123").
				Set(tt.attr, tt.value)

			data, err := defaultEncoder.FromRecord(record)
			require.NoError(t, err)

			var m map[string]any
			err = json.Unmarshal(data, &m)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, m[tt.attr])
		})
	}
}

func TestEncoder_ArrayTypes(t *testing.T) {
	tests := []struct {
		name     string
		attr     string
		value    any
		expected []any
	}{
		{
			name:     "string_array",
			attr:     "tags",
			value:    []string{"go", "rust", "python"},
			expected: []any{"go", "rust", "python"},
		},
		{
			name:     "int_array",
			attr:     "scores",
			value:    []int{10, 20, 30},
			expected: []any{float64(10), float64(20), float64(30)},
		},
		{
			name:     "bool_array",
			attr:     "features",
			value:    []bool{true, false, true},
			expected: []any{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord("com.example", "test", "123").
				Set(tt.attr, tt.value)

			data, err := defaultEncoder.FromRecord(record)
			require.NoError(t, err)

			var m map[string]any
			err = json.Unmarshal(data, &m)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, m[tt.attr])
		})
	}
}

func TestEncoder_EmptyArray(t *testing.T) {
	record := core.NewRecord("com.example", "test", "123").
		Set("empty", []string{})

	data, err := defaultEncoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Contains(t, m, "empty")
	assert.Nil(t, m["empty"])
}

func TestEncoder_IndentOutput(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	data, err := defaultEncoder.FromRecordIndent(record, "", "  ")
	require.NoError(t, err)

	expected := `{
  "_id": "123",
  "name": "John Doe"
}`
	assert.Equal(t, expected, string(data))
}

func TestEncoder_SortedKeys(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("zebra", "last").
		Set("alpha", "first").
		Set("middle", "middle")

	data, err := defaultEncoder.FromRecord(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Contains(t, m, "alpha")
	assert.Contains(t, m, "middle")
	assert.Contains(t, m, "zebra")
}

func TestEncoder_ErrorNilRecord(t *testing.T) {
	data, err := defaultEncoder.FromRecord(nil)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
}

func TestEncoder_ErrorNilRecordIndent(t *testing.T) {
	data, err := defaultEncoder.FromRecordIndent(nil, "", "  ")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
}
