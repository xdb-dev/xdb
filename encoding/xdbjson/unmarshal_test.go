package xdbjson_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/schema"
)

func TestUnmarshal_BasicEncoding(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	data, err := xdbjson.Unmarshal(record)
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

func TestUnmarshal_WithMetadata(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	encOpts := []xdbjson.Option{
		xdbjson.WithIncludeNS(),
		xdbjson.WithIncludeSchema(),
	}

	data, err := xdbjson.Unmarshal(record, encOpts...)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Equal(t, "123", m["_id"])
	assert.Equal(t, "com.example", m["_ns"])
	assert.Equal(t, "users", m["_schema"])
	assert.Equal(t, "John Doe", m["name"])
}

func TestUnmarshal_CustomFieldNames(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	encOpts := []xdbjson.Option{
		xdbjson.WithIDField("userId"),
		xdbjson.WithNSField("namespace"),
		xdbjson.WithSchemaField("type"),
		xdbjson.WithIncludeNS(),
		xdbjson.WithIncludeSchema(),
	}

	data, err := xdbjson.Unmarshal(record, encOpts...)
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

func TestUnmarshal_NestedAttributes(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.city", "Boston").
		Set("address.location.lat", 42.3601).
		Set("address.location.lon", -71.0589)

	data, err := xdbjson.Unmarshal(record)
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

func TestUnmarshal_BasicTypes(t *testing.T) {
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

			data, err := xdbjson.Unmarshal(record)
			require.NoError(t, err)

			var m map[string]any
			err = json.Unmarshal(data, &m)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, m[tt.attr])
		})
	}
}

func TestUnmarshal_ArrayTypes(t *testing.T) {
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

			data, err := xdbjson.Unmarshal(record)
			require.NoError(t, err)

			var m map[string]any
			err = json.Unmarshal(data, &m)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, m[tt.attr])
		})
	}
}

func TestUnmarshal_ObjectArray(t *testing.T) {
	record := core.NewRecord("com.example", "orders", "o1").
		Set("lines", core.ArrayVal(core.TIDJSON,
			core.JSONVal([]byte(`{"qty":3,"sku":"A-1"}`)),
			core.JSONVal([]byte(`{"qty":5,"sku":"B-2"}`)),
		))

	data, err := xdbjson.Unmarshal(record)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))

	lines, ok := m["lines"].([]any)
	require.True(t, ok, "lines should be a JSON array")
	require.Len(t, lines, 2)

	first, ok := lines[0].(map[string]any)
	require.True(t, ok, "each element should be a JSON object")
	assert.Equal(t, "A-1", first["sku"])
	assert.Equal(t, float64(3), first["qty"])
}

func TestUnmarshal_EmptyArray(t *testing.T) {
	record := core.NewRecord("com.example", "test", "123").
		Set("empty", []string{})

	data, err := xdbjson.Unmarshal(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Contains(t, m, "empty")
	// An empty typed slice is a real empty array, encoded as [].
	assert.Equal(t, []any{}, m["empty"])
}

func TestUnmarshal_IndentOutput(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe")

	data, err := xdbjson.Unmarshal(record, xdbjson.WithIndent("", "  "))
	require.NoError(t, err)

	expected := `{
  "_id": "123",
  "name": "John Doe"
}`
	assert.Equal(t, expected, string(data))
}

func TestUnmarshal_SortedKeys(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("zebra", "last").
		Set("alpha", "first").
		Set("middle", "middle")

	data, err := xdbjson.Unmarshal(record)
	require.NoError(t, err)

	var m map[string]any
	err = json.Unmarshal(data, &m)
	require.NoError(t, err)

	assert.Contains(t, m, "alpha")
	assert.Contains(t, m, "middle")
	assert.Contains(t, m, "zebra")
}

// A declared ARRAY<INTEGER> field built directly (not from JSON) round-trips
// through encode -> decode with its element type preserved, including a
// value beyond the float64 exact-integer boundary.
func TestUnmarshal_RoundTrip_ArrayInteger(t *testing.T) {
	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/metrics"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"nums": {Type: core.NewArrayType(core.TIDInteger)},
		},
	}

	original := core.NewRecord("com.example", "metrics", "1").
		Set("nums", []int64{1, 2, 9007199254740993})

	encOpts := []xdbjson.Option{xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema()}
	data, err := xdbjson.Unmarshal(original, encOpts...)
	require.NoError(t, err)

	decOpts := []xdbjson.Option{xdbjson.WithDef(def)}
	decoded, err := xdbjson.Marshal(data, decOpts...)
	require.NoError(t, err)

	nums := decoded.Get("nums").Value()
	assert.Equal(t, core.TIDInteger, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 3)
	got, err := elems[2].AsInt()
	require.NoError(t, err)
	assert.Equal(t, int64(9007199254740993), got)
}

// A declared JSON field built directly round-trips through encode -> decode
// as the same nested structure, not flattened into dotted sub-attributes.
func TestUnmarshal_RoundTrip_JSONField(t *testing.T) {
	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/events"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"settings": {Type: core.TypeJSON},
		},
	}

	original := core.NewRecord("com.example", "events", "1").
		Set("settings", core.JSONVal(json.RawMessage(`{"theme":"dark","level":2}`)))

	encOpts := []xdbjson.Option{xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema()}
	data, err := xdbjson.Unmarshal(original, encOpts...)
	require.NoError(t, err)

	decOpts := []xdbjson.Option{xdbjson.WithDef(def)}
	decoded, err := xdbjson.Marshal(data, decOpts...)
	require.NoError(t, err)

	assert.Nil(t, decoded.Get("settings.theme"), "no synthetic dotted sub-attribute")

	raw, err := decoded.Get("settings").Value().AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"theme":"dark","level":2}`, string(raw))
}

func TestUnmarshal_ErrorNilRecord(t *testing.T) {
	data, err := xdbjson.Unmarshal(nil)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
}

func TestUnmarshal_ErrorNilRecordIndent(t *testing.T) {
	data, err := xdbjson.Unmarshal(nil, xdbjson.WithIndent("", "  "))
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
}

func TestUnmarshal_FromRecordFields(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com").
		Set("age", int64(30))

	t.Run("projects to subset", func(t *testing.T) {
		data, err := xdbjson.Unmarshal(record, xdbjson.WithFields("name"))
		require.NoError(t, err)

		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))

		assert.Equal(t, "123", m["_id"], "_id always included")
		assert.Equal(t, "John Doe", m["name"])
		assert.NotContains(t, m, "email")
		assert.NotContains(t, m, "age")
	})

	t.Run("projects multiple fields", func(t *testing.T) {
		data, err := xdbjson.Unmarshal(record, xdbjson.WithFields("name", "age"))
		require.NoError(t, err)

		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))

		assert.Equal(t, "123", m["_id"])
		assert.Equal(t, "John Doe", m["name"])
		assert.Equal(t, float64(30), m["age"])
		assert.NotContains(t, m, "email")
	})

	t.Run("no fields returns all", func(t *testing.T) {
		data, err := xdbjson.Unmarshal(record)
		require.NoError(t, err)

		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))

		assert.Len(t, m, 4) // _id + name + email + age
	})

	t.Run("nil record returns error", func(t *testing.T) {
		_, err := xdbjson.Unmarshal(nil, xdbjson.WithFields("name"))
		assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
	})
}
