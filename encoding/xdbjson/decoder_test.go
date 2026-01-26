package xdbjson_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
)

var defaultDecoder = xdbjson.NewDefaultDecoder("com.example", "users")

func TestDecoder_BasicDecoding(t *testing.T) {
	data := []byte(`{"_id":"123","name":"John Doe","email":"john@example.com"}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, "com.example", record.NS().String())
	assert.Equal(t, "users", record.Schema().String())
	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "xdb://com.example/users/123", record.URI().String())

	assert.Equal(t, "John Doe", record.Get("name").Value().ToString())
	assert.Equal(t, "john@example.com", record.Get("email").Value().ToString())
}

func TestDecoder_WithMetadata(t *testing.T) {
	data := []byte(`{"_id":"123","_ns":"custom.ns","_schema":"custom_schema","name":"John Doe"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{})

	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "custom.ns", record.NS().String())
	assert.Equal(t, "custom_schema", record.Schema().String())
	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "John Doe", record.Get("name").Value().ToString())
}

func TestDecoder_CustomFieldNames(t *testing.T) {
	data := []byte(`{"userId":"123","namespace":"com.custom","type":"accounts","name":"John"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{
		IDField:     "userId",
		NSField:     "namespace",
		SchemaField: "type",
	})

	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "com.custom", record.NS().String())
	assert.Equal(t, "accounts", record.Schema().String())

	assert.Nil(t, record.Get("userId"))
	assert.Nil(t, record.Get("namespace"))
	assert.Nil(t, record.Get("type"))
}

func TestDecoder_NestedObjects(t *testing.T) {
	data := []byte(`{
		"_id": "123",
		"name": "John Doe",
		"address": {
			"street": "123 Main St",
			"city": "Boston",
			"location": {
				"lat": 42.3601,
				"lon": -71.0589
			}
		}
	}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", record.Get("name").Value().ToString())
	assert.Equal(t, "123 Main St", record.Get("address.street").Value().ToString())
	assert.Equal(t, "Boston", record.Get("address.city").Value().ToString())
	assert.Equal(t, 42.3601, record.Get("address.location.lat").Value().ToFloat())
	assert.Equal(t, -71.0589, record.Get("address.location.lon").Value().ToFloat())

	assert.Nil(t, record.Get("address"))
	assert.Nil(t, record.Get("address.location"))
}

func TestDecoder_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		attr     string
		typeID   core.TID
		getValue func(*core.Tuple) any
		expected any
	}{
		{
			name:     "boolean_true",
			json:     `{"_id":"123","val":true}`,
			attr:     "val",
			typeID:   core.TIDBoolean,
			getValue: func(t *core.Tuple) any { return t.Value().ToBool() },
			expected: true,
		},
		{
			name:     "boolean_false",
			json:     `{"_id":"123","val":false}`,
			attr:     "val",
			typeID:   core.TIDBoolean,
			getValue: func(t *core.Tuple) any { return t.Value().ToBool() },
			expected: false,
		},
		{
			name:     "integer",
			json:     `{"_id":"123","val":42}`,
			attr:     "val",
			typeID:   core.TIDFloat,
			getValue: func(t *core.Tuple) any { return t.Value().ToFloat() },
			expected: float64(42),
		},
		{
			name:     "float",
			json:     `{"_id":"123","val":3.14159}`,
			attr:     "val",
			typeID:   core.TIDFloat,
			getValue: func(t *core.Tuple) any { return t.Value().ToFloat() },
			expected: 3.14159,
		},
		{
			name:     "string",
			json:     `{"_id":"123","val":"hello"}`,
			attr:     "val",
			typeID:   core.TIDString,
			getValue: func(t *core.Tuple) any { return t.Value().ToString() },
			expected: "hello",
		},
		{
			name:     "empty_string",
			json:     `{"_id":"123","val":""}`,
			attr:     "val",
			typeID:   core.TIDString,
			getValue: func(t *core.Tuple) any { return t.Value().ToString() },
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := defaultDecoder.ToRecord([]byte(tt.json))
			require.NoError(t, err)

			tuple := record.Get(tt.attr)
			require.NotNil(t, tuple, "tuple should not be nil")
			assert.Equal(t, tt.typeID, tuple.Value().Type().ID())
			assert.Equal(t, tt.expected, tt.getValue(tuple))
		})
	}
}

func TestDecoder_ArrayTypes(t *testing.T) {
	data := []byte(`{
		"_id": "123",
		"strings": ["go", "rust", "python"],
		"numbers": [10, 20, 30],
		"bools": [true, false, true]
	}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)

	strings := record.Get("strings")
	require.NotNil(t, strings)
	assert.Equal(t, core.TIDArray, strings.Value().Type().ID())

	numbers := record.Get("numbers")
	require.NotNil(t, numbers)
	assert.Equal(t, core.TIDArray, numbers.Value().Type().ID())

	bools := record.Get("bools")
	require.NotNil(t, bools)
	assert.Equal(t, core.TIDArray, bools.Value().Type().ID())
}

func TestDecoder_EmptyArray(t *testing.T) {
	data := []byte(`{"_id": "123", "empty": []}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)

	empty := record.Get("empty")
	require.NotNil(t, empty)
	assert.Nil(t, empty.Value())
}

func TestDecoder_NullValue(t *testing.T) {
	data := []byte(`{"_id":"123","name":"John","nickname":null}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)

	assert.NotNil(t, record.Get("name"))
	assert.Nil(t, record.Get("nickname"))
}

func TestDecoder_NumericID(t *testing.T) {
	data := []byte(`{"_id":12345,"name":"John"}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "12345", record.ID().String())
}

func TestDecoder_ToExistingRecord(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("existing", "value")

	data := []byte(`{"_id":"ignored","_ns":"ignored","_schema":"ignored","name":"John Doe"}`)

	err := defaultDecoder.ToExistingRecord(data, record)
	require.NoError(t, err)

	assert.Equal(t, "com.example", record.NS().String())
	assert.Equal(t, "users", record.Schema().String())
	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "John Doe", record.Get("name").Value().ToString())
	assert.Equal(t, "value", record.Get("existing").Value().ToString())
}

func TestDecoder_FallbackToOptions(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		opts       xdbjson.Options
		expectedNS string
		expectedSc string
	}{
		{
			name:       "ns_from_options",
			json:       `{"_id":"123","_schema":"orders"}`,
			opts:       xdbjson.Options{NS: "default.ns"},
			expectedNS: "default.ns",
			expectedSc: "orders",
		},
		{
			name:       "schema_from_options",
			json:       `{"_id":"123","_ns":"custom.ns"}`,
			opts:       xdbjson.Options{Schema: "default_schema"},
			expectedNS: "custom.ns",
			expectedSc: "default_schema",
		},
		{
			name:       "both_from_options",
			json:       `{"_id":"123"}`,
			opts:       xdbjson.Options{NS: "default.ns", Schema: "default_schema"},
			expectedNS: "default.ns",
			expectedSc: "default_schema",
		},
		{
			name:       "json_overrides_options",
			json:       `{"_id":"123","_ns":"json.ns","_schema":"json_schema"}`,
			opts:       xdbjson.Options{NS: "default.ns", Schema: "default_schema"},
			expectedNS: "json.ns",
			expectedSc: "json_schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := xdbjson.NewDecoder(tt.opts)

			record, err := decoder.ToRecord([]byte(tt.json))
			require.NoError(t, err)

			assert.Equal(t, tt.expectedNS, record.NS().String())
			assert.Equal(t, tt.expectedSc, record.Schema().String())
		})
	}
}

func TestDecoder_RoundTrip(t *testing.T) {
	original := core.NewRecord("com.example", "users", "user-789").
		Set("name", "Alice").
		Set("score", 100).
		Set("address.city", "Boston")

	encoder := xdbjson.NewEncoder(xdbjson.Options{
		IncludeNS:     true,
		IncludeSchema: true,
	})

	data, err := encoder.FromRecord(original)
	require.NoError(t, err)

	decoder := xdbjson.NewDecoder(xdbjson.Options{})

	decoded, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, original.URI().String(), decoded.URI().String())
	assert.Equal(t, original.Get("name").Value().ToString(), decoded.Get("name").Value().ToString())
	assert.Equal(t, original.Get("address.city").Value().ToString(), decoded.Get("address.city").Value().ToString())
}

func TestDecoder_ErrorInvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"malformed", `{"_id":"123"`},
		{"not_object", `["array"]`},
		{"empty_string", ``},
		{"random_text", `not json at all`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := defaultDecoder.ToRecord([]byte(tt.json))
			assert.Error(t, err)
			assert.Nil(t, record)
			assert.ErrorIs(t, err, xdbjson.ErrInvalidJSON)
		})
	}
}

func TestDecoder_ErrorMissingID(t *testing.T) {
	data := []byte(`{"name":"John Doe"}`)

	record, err := defaultDecoder.ToRecord(data)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbjson.ErrMissingID)
}

func TestDecoder_ErrorEmptyID(t *testing.T) {
	data := []byte(`{"_id":"","name":"John Doe"}`)

	record, err := defaultDecoder.ToRecord(data)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbjson.ErrEmptyID)
}

func TestDecoder_ErrorMissingNamespace(t *testing.T) {
	data := []byte(`{"_id":"123","_schema":"users","name":"John"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{})

	record, err := decoder.ToRecord(data)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbjson.ErrMissingNamespace)
}

func TestDecoder_ErrorMissingSchema(t *testing.T) {
	data := []byte(`{"_id":"123","_ns":"com.example","name":"John"}`)

	decoder := xdbjson.NewDecoder(xdbjson.Options{})

	record, err := decoder.ToRecord(data)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbjson.ErrMissingSchema)
}

func TestDecoder_ErrorNilRecord(t *testing.T) {
	data := []byte(`{"_id":"123","name":"John"}`)

	err := defaultDecoder.ToExistingRecord(data, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, xdbjson.ErrNilRecord)
}

func TestDecoder_ErrorToExistingRecordInvalidJSON(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123")

	err := defaultDecoder.ToExistingRecord([]byte(`not json`), record)
	assert.Error(t, err)
	assert.ErrorIs(t, err, xdbjson.ErrInvalidJSON)
}
