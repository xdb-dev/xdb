package xdbjson_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbjson"
	"github.com/xdb-dev/xdb/schema"
)

var defaultDecoder = xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("users"))

func TestDecoder_BasicDecoding(t *testing.T) {
	data := []byte(`{"_id":"123","name":"John Doe","email":"john@example.com"}`)

	record, err := defaultDecoder.ToRecord(data)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, "com.example", record.URI().NS())
	assert.Equal(t, "users", record.URI().Schema())
	assert.Equal(t, "123", record.URI().ID())
	assert.Equal(t, "xdb://com.example/users/123", record.URI().String())

	assert.Equal(t, "John Doe", vStr(record.Get("name").Value()))
	assert.Equal(t, "john@example.com", vStr(record.Get("email").Value()))
}

func TestDecoder_WithMetadata(t *testing.T) {
	data := []byte(`{"_id":"123","_ns":"custom.ns","_schema":"custom_schema","name":"John Doe"}`)

	decoder := xdbjson.NewDecoder()

	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "custom.ns", record.URI().NS())
	assert.Equal(t, "custom_schema", record.URI().Schema())
	assert.Equal(t, "123", record.URI().ID())
	assert.Equal(t, "John Doe", vStr(record.Get("name").Value()))
}

func TestDecoder_CustomFieldNames(t *testing.T) {
	data := []byte(`{"userId":"123","namespace":"com.custom","type":"accounts","name":"John"}`)

	decoder := xdbjson.NewDecoder(
		xdbjson.WithIDField("userId"),
		xdbjson.WithNSField("namespace"),
		xdbjson.WithSchemaField("type"),
	)

	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, "123", record.URI().ID())
	assert.Equal(t, "com.custom", record.URI().NS())
	assert.Equal(t, "accounts", record.URI().Schema())

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

	assert.Equal(t, "John Doe", vStr(record.Get("name").Value()))
	assert.Equal(t, "123 Main St", vStr(record.Get("address.street").Value()))
	assert.Equal(t, "Boston", vStr(record.Get("address.city").Value()))
	assert.Equal(t, 42.3601, vFloat(record.Get("address.location.lat").Value()))
	assert.Equal(t, -71.0589, vFloat(record.Get("address.location.lon").Value()))

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
			getValue: func(t *core.Tuple) any { return vBool(t.Value()) },
			expected: true,
		},
		{
			name:     "boolean_false",
			json:     `{"_id":"123","val":false}`,
			attr:     "val",
			typeID:   core.TIDBoolean,
			getValue: func(t *core.Tuple) any { return vBool(t.Value()) },
			expected: false,
		},
		{
			// Whole-looking numbers decode as int64 by default now that
			// UseNumber is always on (see TestDecoder_DefaultNumberInference).
			name:     "integer",
			json:     `{"_id":"123","val":42}`,
			attr:     "val",
			typeID:   core.TIDInteger,
			getValue: func(t *core.Tuple) any { return vInt(t.Value()) },
			expected: int64(42),
		},
		{
			name:     "float",
			json:     `{"_id":"123","val":3.14159}`,
			attr:     "val",
			typeID:   core.TIDFloat,
			getValue: func(t *core.Tuple) any { return vFloat(t.Value()) },
			expected: 3.14159,
		},
		{
			name:     "string",
			json:     `{"_id":"123","val":"hello"}`,
			attr:     "val",
			typeID:   core.TIDString,
			getValue: func(t *core.Tuple) any { return vStr(t.Value()) },
			expected: "hello",
		},
		{
			name:     "empty_string",
			json:     `{"_id":"123","val":""}`,
			attr:     "val",
			typeID:   core.TIDString,
			getValue: func(t *core.Tuple) any { return vStr(t.Value()) },
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

	// An empty JSON array has no derivable element type, so it is skipped —
	// the attribute is absent, consistent with how null values are handled.
	assert.Nil(t, record.Get("empty"))
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

	assert.Equal(t, "12345", record.URI().ID())
}

func TestDecoder_ToExistingRecord(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("existing", "value")

	data := []byte(`{"_id":"ignored","_ns":"ignored","_schema":"ignored","name":"John Doe"}`)

	err := defaultDecoder.ToExistingRecord(data, record)
	require.NoError(t, err)

	assert.Equal(t, "com.example", record.URI().NS())
	assert.Equal(t, "users", record.URI().Schema())
	assert.Equal(t, "123", record.URI().ID())
	assert.Equal(t, "John Doe", vStr(record.Get("name").Value()))
	assert.Equal(t, "value", vStr(record.Get("existing").Value()))
}

func TestDecoder_FallbackToOptions(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		opts       []xdbjson.Option
		expectedNS string
		expectedSc string
	}{
		{
			name:       "ns_from_options",
			json:       `{"_id":"123","_schema":"orders"}`,
			opts:       []xdbjson.Option{xdbjson.WithNS("default.ns")},
			expectedNS: "default.ns",
			expectedSc: "orders",
		},
		{
			name:       "schema_from_options",
			json:       `{"_id":"123","_ns":"custom.ns"}`,
			opts:       []xdbjson.Option{xdbjson.WithSchema("default_schema")},
			expectedNS: "custom.ns",
			expectedSc: "default_schema",
		},
		{
			name:       "both_from_options",
			json:       `{"_id":"123"}`,
			opts:       []xdbjson.Option{xdbjson.WithNS("default.ns"), xdbjson.WithSchema("default_schema")},
			expectedNS: "default.ns",
			expectedSc: "default_schema",
		},
		{
			name:       "json_overrides_options",
			json:       `{"_id":"123","_ns":"json.ns","_schema":"json_schema"}`,
			opts:       []xdbjson.Option{xdbjson.WithNS("default.ns"), xdbjson.WithSchema("default_schema")},
			expectedNS: "json.ns",
			expectedSc: "json_schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := xdbjson.NewDecoder(tt.opts...)

			record, err := decoder.ToRecord([]byte(tt.json))
			require.NoError(t, err)

			assert.Equal(t, tt.expectedNS, record.URI().NS())
			assert.Equal(t, tt.expectedSc, record.URI().Schema())
		})
	}
}

func TestDecoder_RoundTrip(t *testing.T) {
	original := core.NewRecord("com.example", "users", "user-789").
		Set("name", "Alice").
		Set("score", 100).
		Set("address.city", "Boston")

	encoder := xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())

	data, err := encoder.FromRecord(original)
	require.NoError(t, err)

	decoder := xdbjson.NewDecoder()

	decoded, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, original.URI().String(), decoded.URI().String())
	assert.Equal(t, vStr(original.Get("name").Value()), vStr(decoded.Get("name").Value()))
	assert.Equal(t, vStr(original.Get("address.city").Value()), vStr(decoded.Get("address.city").Value()))
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

	decoder := xdbjson.NewDecoder()

	record, err := decoder.ToRecord(data)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbjson.ErrMissingNamespace)
}

func TestDecoder_ErrorMissingSchema(t *testing.T) {
	data := []byte(`{"_id":"123","_ns":"com.example","name":"John"}`)

	decoder := xdbjson.NewDecoder()

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

func TestDecoder_WithSchema_AllTypes(t *testing.T) {
	uri := core.MustNewURI("com.example", "test")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"timestamp": {Type: core.TypeTime},
			"count":     {Type: core.TypeInt},
			"size":      {Type: core.TypeUnsigned},
			"data":      {Type: core.TypeBytes},
		},
	}

	tests := []struct {
		name     string
		json     string
		attr     string
		expected core.TID
	}{
		{"time", `{"_id":"1","timestamp":"2025-01-26T10:00:00Z"}`, "timestamp", core.TIDTime},
		{"integer", `{"_id":"1","count":42}`, "count", core.TIDInteger},
		{"unsigned", `{"_id":"1","size":100}`, "size", core.TIDUnsigned},
		{"bytes", `{"_id":"1","data":"SGVsbG8="}`, "data", core.TIDBytes},
	}

	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("test"),
		xdbjson.WithDef(def),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := decoder.ToRecord([]byte(tt.json))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, record.Get(tt.attr).Value().Type().ID())
		})
	}
}

func TestDecoder_WithSchema_RoundTrip(t *testing.T) {
	uri := core.MustNewURI("com.example", "test")
	def := &schema.Def{
		URI: uri,
		Fields: map[string]schema.Field{
			"created_at": {Type: core.TypeTime},
			"count":      {Type: core.TypeInt},
			"data":       {Type: core.TypeBytes},
		},
	}

	original := core.NewRecord("com.example", "test", "123").
		Set("created_at", time.Date(2025, 1, 26, 10, 0, 0, 0, time.UTC)).
		Set("count", int64(42)).
		Set("data", []byte("Hello"))

	encoder := xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIncludeSchema())
	data, err := encoder.FromRecord(original)
	require.NoError(t, err)

	decoder := xdbjson.NewDecoder(xdbjson.WithDef(def))
	decoded, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, core.TIDTime, decoded.Get("created_at").Value().Type().ID())
	assert.Equal(t, core.TIDInteger, decoded.Get("count").Value().Type().ID())
	assert.Equal(t, core.TIDBytes, decoded.Get("data").Value().Type().ID())

	assert.Equal(t, vTime(original.Get("created_at").Value()), vTime(decoded.Get("created_at").Value()))
	assert.Equal(t, vInt(original.Get("count").Value()), vInt(decoded.Get("count").Value()))
	assert.Equal(t, vBytes(original.Get("data").Value()), vBytes(decoded.Get("data").Value()))
}

func objectArrayDef() *schema.Def {
	return &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/orders"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"lines": {
				Type: core.NewArrayType(core.TIDJSON),
				Items: map[string]schema.Field{
					"sku":    {Type: core.TypeString, Required: true},
					"qty":    {Type: core.TypeInt},
					"placed": {Type: core.TypeTime},
				},
			},
		},
	}
}

func TestDecoder_ObjectArray(t *testing.T) {
	def := objectArrayDef()
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("orders"),
		xdbjson.WithDef(def),
	)

	data := []byte(`{
		"_id": "o1",
		"lines": [
			{"sku": "A-1", "qty": 3, "placed": "2026-07-17T10:00:00Z"},
			{"sku": "B-2", "qty": 5, "placed": "2026-07-18T11:30:00Z"}
		]
	}`)

	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	lines := record.Get("lines").Value()
	assert.Equal(t, core.TIDArray, lines.Type().ID())
	assert.Equal(t, core.TIDJSON, lines.Type().ElemTypeID())

	elems, err := lines.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 2)

	// Members are stored in typed JSON form: time as RFC3339, int as a number.
	raw, err := elems[0].AsJSON()
	require.NoError(t, err)

	var obj map[string]any
	require.NoError(t, json.Unmarshal(raw, &obj))
	assert.Equal(t, "A-1", obj["sku"])
	assert.Equal(t, float64(3), obj["qty"])

	placed, err := time.Parse(time.RFC3339, obj["placed"].(string))
	require.NoError(t, err)
	assert.True(t, placed.Equal(time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)))
}

// A time.Time nested inside an object-array element must round-trip typed
// through encode -> decode: the decoded element re-parses to the same instant,
// not to a raw or mangled string.
func TestDecoder_ObjectArray_TimeRoundTrip(t *testing.T) {
	def := objectArrayDef()
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("orders"),
		xdbjson.WithDef(def),
	)

	want := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)

	original, err := decoder.ToRecord([]byte(
		`{"_id":"o1","lines":[{"sku":"A-1","qty":3,"placed":"2026-07-17T10:00:00Z"}]}`,
	))
	require.NoError(t, err)

	encoder := xdbjson.New()
	encoded, err := encoder.FromRecord(original)
	require.NoError(t, err)

	decoded, err := decoder.ToRecord(encoded)
	require.NoError(t, err)

	origElems, err := original.Get("lines").Value().AsArray()
	require.NoError(t, err)
	decElems, err := decoded.Get("lines").Value().AsArray()
	require.NoError(t, err)
	require.Len(t, decElems, len(origElems))

	origRaw, err := origElems[0].AsJSON()
	require.NoError(t, err)
	decRaw, err := decElems[0].AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, string(origRaw), string(decRaw))

	var obj map[string]any
	require.NoError(t, json.Unmarshal(decRaw, &obj))
	got, err := time.Parse(time.RFC3339, obj["placed"].(string))
	require.NoError(t, err)
	assert.True(t, want.Equal(got))
}

func TestDecoder_WithoutSchema_NoConversion(t *testing.T) {
	decoder := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("events"))

	data := []byte(`{"_id": "123", "created_at": "2025-01-26T10:00:00Z"}`)
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, core.TIDString, record.Get("created_at").Value().Type().ID())
}

// WithNumberInference is now a documented no-op: number inference is always
// on, so this test only proves the option remains harmless to pass.
func TestDecoder_WithNumberInference(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("events"),
		xdbjson.WithNumberInference(),
	)

	data := []byte(`{"_id":"1","count":42,"rating":4.5,"tags":[1,2]}`)
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, core.TIDInteger, record.Get("count").Value().Type().ID())
	assert.Equal(t, core.TIDFloat, record.Get("rating").Value().Type().ID())

	tags := record.Get("tags").Value()
	assert.Equal(t, core.TIDArray, tags.Type().ID())
	elems, err := tags.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 2)
	assert.Equal(t, core.TIDInteger, elems[0].Type().ID())
}

// A declared FLOAT/UNSIGNED field holding an integral value (encoded as a bare
// JSON number) must be typed by the schema, not by number inference — otherwise
// inference would round an integral float down to INTEGER.
func TestDecoder_WithNumberInference_DefTypesWin(t *testing.T) {
	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/metrics"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"ratio": {Type: core.TypeFloat},
			"size":  {Type: core.TypeUnsigned},
			"count": {Type: core.TypeInt},
		},
	}

	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithNumberInference(),
		xdbjson.WithDef(def),
	)

	data := []byte(`{"_id":"1","ratio":5,"size":9,"count":42}`)
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, core.TIDFloat, record.Get("ratio").Value().Type().ID())
	assert.Equal(t, core.TIDUnsigned, record.Get("size").Value().Type().ID())
	assert.Equal(t, core.TIDInteger, record.Get("count").Value().Type().ID())
}

// Without a schema, whole-looking numbers decode as int64 and fractional
// numbers as float64 — number inference is always on now.
func TestDecoder_DefaultNumberInference(t *testing.T) {
	decoder := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("events"))

	data := []byte(`{"_id":"1","count":42,"rating":4.2}`)
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, core.TIDInteger, record.Get("count").Value().Type().ID())
	assert.Equal(t, int64(42), vInt(record.Get("count").Value()))

	assert.Equal(t, core.TIDFloat, record.Get("rating").Value().Type().ID())
	assert.Equal(t, 4.2, vFloat(record.Get("rating").Value()))
}

// A declared INTEGER field, and an INTEGER array element, must survive a
// value beyond 2^53 (the float64 exact-integer boundary) without precision
// loss — proof that numbers flow through as json.Number, never float64.
func TestDecoder_BigIntegerExact(t *testing.T) {
	const big = 9007199254740993 // 2^53 + 1

	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/metrics"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"count": {Type: core.TypeInt},
			"nums":  {Type: core.NewArrayType(core.TIDInteger)},
		},
	}

	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(def),
	)

	data := []byte(fmt.Sprintf(`{"_id":"1","count":%d,"nums":[%d]}`, big, big))
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Equal(t, int64(big), vInt(record.Get("count").Value()))

	elems, err := record.Get("nums").Value().AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 1)
	assert.Equal(t, core.TIDInteger, elems[0].Type().ID())
	i, err := elems[0].AsInt()
	require.NoError(t, err)
	assert.Equal(t, int64(big), i)
}

func jsonFieldDef() *schema.Def {
	return &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/events"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"settings": {Type: core.TypeJSON},
		},
	}
}

// A JSON-typed field's nested object must decode as ONE tuple (the field's
// own attribute), never flattened into synthetic dotted sub-attributes —
// otherwise strict mode would reject the (fictitious) unknown sub-fields.
func TestDecoder_JSONField_Object(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("events"),
		xdbjson.WithDef(jsonFieldDef()),
	)

	data := []byte(`{"_id":"1","settings":{"theme":"dark","nested":{"level":2}}}`)
	record, err := decoder.ToRecord(data)
	require.NoError(t, err)

	assert.Nil(t, record.Get("settings.theme"), "no synthetic dotted sub-attribute")
	assert.Nil(t, record.Get("settings.nested"), "no synthetic dotted sub-attribute")

	settings := record.Get("settings")
	require.NotNil(t, settings)
	assert.Equal(t, core.TIDJSON, settings.Value().Type().ID())

	raw, err := settings.Value().AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"theme":"dark","nested":{"level":2}}`, string(raw))
}

// A JSON-typed field given a scalar or array (not an object) is still
// accepted, stored as a JSON value rather than rejected or type-coerced.
func TestDecoder_JSONField_ScalarAndArray(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"scalar_number", `{"_id":"1","settings":42}`, `42`},
		{"scalar_string", `{"_id":"1","settings":"dark"}`, `"dark"`},
		{"array", `{"_id":"1","settings":[1,"two",true]}`, `[1,"two",true]`},
	}

	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("events"),
		xdbjson.WithDef(jsonFieldDef()),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := decoder.ToRecord([]byte(tt.json))
			require.NoError(t, err)

			settings := record.Get("settings")
			require.NotNil(t, settings)
			assert.Equal(t, core.TIDJSON, settings.Value().Type().ID())

			raw, err := settings.Value().AsJSON()
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(raw))
		})
	}
}

// A JSON-typed field round-trips through encode -> decode stably.
func TestDecoder_JSONField_RoundTrip(t *testing.T) {
	def := jsonFieldDef()
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("events"),
		xdbjson.WithDef(def),
	)

	original, err := decoder.ToRecord([]byte(
		`{"_id":"1","settings":{"theme":"dark","nested":{"level":2}}}`,
	))
	require.NoError(t, err)

	encoder := xdbjson.New()
	encoded, err := encoder.FromRecord(original)
	require.NoError(t, err)

	decoded, err := decoder.ToRecord(encoded)
	require.NoError(t, err)

	origRaw, err := original.Get("settings").Value().AsJSON()
	require.NoError(t, err)
	decRaw, err := decoded.Get("settings").Value().AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, string(origRaw), string(decRaw))
}

func arrayFieldDef(elemType core.TID) *schema.Def {
	return &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/metrics"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"nums": {Type: core.NewArrayType(elemType)},
		},
	}
}

func TestDecoder_ArrayInteger(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDInteger)),
	)

	record, err := decoder.ToRecord([]byte(`{"_id":"1","nums":[1,2,3]}`))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDArray, nums.Type().ID())
	assert.Equal(t, core.TIDInteger, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 3)
	for i, want := range []int64{1, 2, 3} {
		assert.Equal(t, core.TIDInteger, elems[i].Type().ID())
		got, err := elems[i].AsInt()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

// A non-integral element in a declared ARRAY<INTEGER> is never silently
// truncated (1.5 must not become 1) — it decodes as its own natural numeric
// type, which then fails to match the declared array type.
func TestDecoder_ArrayInteger_LossyElementNotTruncated(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDInteger)),
	)

	_, err := decoder.ToRecord([]byte(`{"_id":"1","nums":[1.5]}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrSchemaViolation)
	assert.Contains(t, err.Error(), "nums")
}

func TestDecoder_ArrayFloat(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDFloat)),
	)

	// Whole-looking numbers must still decode as floats: the declared
	// elem_type wins over first-element inference.
	record, err := decoder.ToRecord([]byte(`{"_id":"1","nums":[1,2]}`))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDFloat, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 2)
	for i, want := range []float64{1, 2} {
		assert.Equal(t, core.TIDFloat, elems[i].Type().ID())
		got, err := elems[i].AsFloat()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestDecoder_ArrayTime(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDTime)),
	)

	record, err := decoder.ToRecord([]byte(
		`{"_id":"1","nums":["2026-07-17T10:00:00Z","2026-07-18T11:30:00Z"]}`,
	))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDTime, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 2)
	want := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	got, err := elems[0].AsTime()
	require.NoError(t, err)
	assert.True(t, want.Equal(got))
}

func TestDecoder_ArrayBytes(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDBytes)),
	)

	record, err := decoder.ToRecord([]byte(`{"_id":"1","nums":["SGVsbG8=","V29ybGQ="]}`))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDBytes, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 2)
	got, err := elems[0].AsBytes()
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello"), got)
}

// An ARRAY<JSON> field with no Items has no member schema to validate
// against — every element (object, scalar, or array) must still round-trip
// as a plain JSON value rather than being silently dropped.
func TestDecoder_ArrayJSON_NoItems(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDJSON)),
	)

	record, err := decoder.ToRecord([]byte(`{"_id":"1","nums":[{"a":1},"two",3]}`))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDJSON, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	require.Len(t, elems, 3)

	raw, err := elems[0].AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(raw))

	raw, err = elems[1].AsJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `"two"`, string(raw))
}

func TestDecoder_ArrayEmpty_DeclaredType(t *testing.T) {
	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(arrayFieldDef(core.TIDInteger)),
	)

	record, err := decoder.ToRecord([]byte(`{"_id":"1","nums":[]}`))
	require.NoError(t, err)

	nums := record.Get("nums").Value()
	assert.Equal(t, core.TIDArray, nums.Type().ID())
	assert.Equal(t, core.TIDInteger, nums.Type().ElemTypeID())

	elems, err := nums.AsArray()
	require.NoError(t, err)
	assert.Len(t, elems, 0)
}

// A declared field whose value cannot decode as its declared type is a
// decode-time error naming the field and the expected type. An undeclared
// attribute with the same kind of unusable value is skipped silently, same
// as always (e.g. TestDecoder_EmptyArray).
func TestDecoder_DeclaredFieldError(t *testing.T) {
	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/metrics"),
		Mode: schema.ModeStrict,
		Fields: map[string]schema.Field{
			"count": {Type: core.TypeInt},
		},
	}

	decoder := xdbjson.NewDecoder(
		xdbjson.WithNS("com.example"),
		xdbjson.WithSchema("metrics"),
		xdbjson.WithDef(def),
	)

	t.Run("declared garbage errors", func(t *testing.T) {
		_, err := decoder.ToRecord([]byte(`{"_id":"1","count":"abc"}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, core.ErrSchemaViolation)
		assert.Contains(t, err.Error(), `"count"`)
		assert.Contains(t, err.Error(), "INTEGER")
	})

	t.Run("undeclared garbage is skipped silently", func(t *testing.T) {
		record, err := decoder.ToRecord([]byte(`{"_id":"1","count":42,"junk":[1,"two"]}`))
		require.NoError(t, err)
		assert.Equal(t, int64(42), vInt(record.Get("count").Value()))
		assert.Nil(t, record.Get("junk"))
	})
}
