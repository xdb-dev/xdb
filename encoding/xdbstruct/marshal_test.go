package xdbstruct_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
)

type IntTypesStruct struct {
	ID string `xdb:"id,primary_key"`

	// Integer Types
	Int   int   `xdb:"int"`
	Int8  int8  `xdb:"int8"`
	Int16 int16 `xdb:"int16"`
	Int32 int32 `xdb:"int32"`
	Int64 int64 `xdb:"int64"`

	// IntegerArray
	IntArray [3]int `xdb:"int_array"`

	// Unsigned Integer Types
	Uint   uint   `xdb:"uint"`
	Uint8  uint8  `xdb:"uint8"`
	Uint16 uint16 `xdb:"uint16"`
	Uint32 uint32 `xdb:"uint32"`
	Uint64 uint64 `xdb:"uint64"`

	// IntegerArray
	UintArray [3]uint `xdb:"uint_array"`
}

func TestIntTypesStruct(t *testing.T) {
	t.Parallel()

	input := IntTypesStruct{
		ID:  "123-456-789",
		Int: 1, Int8: 1, Int16: 1, Int32: 1, Int64: 1,
		IntArray: [3]int{1, 2, 3},
		Uint:     1, Uint8: 1, Uint16: 1, Uint32: 1, Uint64: 1,
		UintArray: [3]uint{1, 2, 3},
	}
	record := types.NewRecord("IntTypesStruct", input.ID).
		Set("int", input.Int).
		Set("int8", input.Int8).
		Set("int16", input.Int16).
		Set("int32", input.Int32).
		Set("int64", input.Int64).
		Set("int_array", input.IntArray).
		Set("uint", input.Uint).
		Set("uint8", input.Uint8).
		Set("uint16", input.Uint16).
		Set("uint32", input.Uint32).
		Set("uint64", input.Uint64).
		Set("uint_array", input.UintArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type FloatTypesStruct struct {
	ID         string     `xdb:"id,primary_key"`
	Float32    float32    `xdb:"float32"`
	Float64    float64    `xdb:"float64"`
	FloatArray [3]float64 `xdb:"float_array"`
}

func TestFloatTypesStruct(t *testing.T) {
	t.Parallel()

	input := FloatTypesStruct{
		ID:         "123-456-789",
		Float32:    123.456,
		Float64:    123.456,
		FloatArray: [3]float64{123.456, 123.456, 123.456},
	}
	record := types.NewRecord("FloatTypesStruct", input.ID).
		Set("float32", input.Float32).
		Set("float64", input.Float64).
		Set("float_array", input.FloatArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type StringTypesStruct struct {
	ID          string    `xdb:"id,primary_key"`
	String      string    `xdb:"string"`
	StringArray [3]string `xdb:"string_array"`
}

func TestStringTypesStruct(t *testing.T) {
	t.Parallel()

	input := StringTypesStruct{
		ID:          "123-456-789",
		String:      "Hello, World!",
		StringArray: [3]string{"Hello", "World", "!"},
	}
	record := types.NewRecord("StringTypesStruct", input.ID).
		Set("string", input.String).
		Set("string_array", input.StringArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type BoolTypesStruct struct {
	ID        string  `xdb:"id,primary_key"`
	Bool      bool    `xdb:"bool"`
	BoolArray [3]bool `xdb:"bool_array"`
}

func TestBoolTypesStruct(t *testing.T) {
	t.Parallel()

	input := BoolTypesStruct{
		ID:        "123-456-789",
		Bool:      true,
		BoolArray: [3]bool{true, false, true},
	}
	record := types.NewRecord("BoolTypesStruct", input.ID).
		Set("bool", input.Bool).
		Set("bool_array", input.BoolArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type ByteTypesStruct struct {
	ID         string    `xdb:"id,primary_key"`
	Bytes      []byte    `xdb:"bytes"`
	BytesArray [3][]byte `xdb:"bytes_array"`

	// JSON
	JSON      json.RawMessage    `xdb:"json"`
	JSONArray [3]json.RawMessage `xdb:"json_array"`

	// Binary
	Binary      Binary    `xdb:"binary"`
	BinaryArray [3]Binary `xdb:"binary_array"`
}

type Binary struct {
	Data []byte
}

func (b *Binary) MarshalBinary() ([]byte, error) {
	return b.Data, nil
}

func (b *Binary) UnmarshalBinary(data []byte) error {
	b.Data = data
	return nil
}

func TestByteTypesStruct(t *testing.T) {
	t.Parallel()

	input := ByteTypesStruct{
		ID:         "123-456-789",
		Bytes:      []byte("Hello, World!"),
		BytesArray: [3][]byte{[]byte("Hello"), []byte("World"), []byte("!")},
		JSON:       json.RawMessage(`{"hello": "world"}`),
		JSONArray: [3]json.RawMessage{
			json.RawMessage(`{"hello": "world"}`),
			json.RawMessage(`{"hello": "world"}`),
			json.RawMessage(`{"hello": "world"}`),
		},
		Binary: Binary{Data: []byte("Hello, World!")},
		BinaryArray: [3]Binary{
			{Data: []byte("Hello")},
			{Data: []byte("World")},
			{Data: []byte("!")},
		},
	}
	record := types.NewRecord("ByteTypesStruct", input.ID).
		Set("bytes", input.Bytes).
		Set("bytes_array", input.BytesArray).
		Set("json", []byte(input.JSON)).
		Set("json_array", [][]byte{[]byte(input.JSONArray[0]), []byte(input.JSONArray[1]), []byte(input.JSONArray[2])}).
		Set("binary", []byte("Hello, World!")).
		Set("binary_array", [][]byte{[]byte("Hello"), []byte("World"), []byte("!")})

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type MapTypesStruct struct {
	ID       string                    `xdb:"id,primary_key"`
	Map      map[string]interface{}    `xdb:"map"`
	MapArray [3]map[string]interface{} `xdb:"map_array"`
}

func TestMapTypesStruct(t *testing.T) {
	t.Parallel()

	input := MapTypesStruct{
		ID: "123-456-789",
		Map: map[string]interface{}{
			"hello": "world",
			"foo":   123,
		},
		MapArray: [3]map[string]interface{}{
			{"hello": "world", "foo": 123},
			{"hello": "world", "foo": 123},
			{"hello": "world", "foo": 123},
		},
	}
	record := types.NewRecord("MapTypesStruct", input.ID).
		Set("map", input.Map).
		Set("map_array", input.MapArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type CompositeTypesStruct struct {
	ID             string       `xdb:"id,primary_key"`
	Timestamp      time.Time    `xdb:"timestamp"`
	TimestampArray [3]time.Time `xdb:"timestamp_array"`

	Point      types.Point    `xdb:"point"`
	PointArray [3]types.Point `xdb:"point_array"`
}

func TestCompositeTypesStruct(t *testing.T) {
	t.Parallel()

	input := CompositeTypesStruct{
		ID:        "123-456-789",
		Timestamp: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		TimestampArray: [3]time.Time{
			time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC),
			time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		},
		Point: types.Point{Lat: 123.456, Long: 78.910},
		PointArray: [3]types.Point{
			{Lat: 123.456, Long: 78.910},
			{Lat: 123.456, Long: 78.910},
			{Lat: 123.456, Long: 78.910},
		},
	}
	record := types.NewRecord("CompositeTypesStruct", input.ID).
		Set("timestamp", input.Timestamp).
		Set("timestamp_array", input.TimestampArray).
		Set("point", input.Point).
		Set("point_array", input.PointArray)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type NestedStruct struct {
	ID      string  `xdb:"id,primary_key"`
	Name    string  `xdb:"name"`
	Age     int     `xdb:"age"`
	Address Address `xdb:"address"`
}

type Address struct {
	Street string `xdb:"street"`
	City   string `xdb:"city"`
	State  string `xdb:"state"`
	Zip    string `xdb:"zip"`
}

func TestNestedStruct(t *testing.T) {
	t.Parallel()

	input := NestedStruct{
		ID:   "123-456-789",
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Anytown",
			State:  "CA",
			Zip:    "12345",
		},
	}
	record := types.NewRecord("NestedStruct", input.ID).
		Set("name", input.Name).
		Set("age", input.Age).
		Set("address.street", input.Address.Street).
		Set("address.city", input.Address.City).
		Set("address.state", input.Address.State).
		Set("address.zip", input.Address.Zip)

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := xdbstruct.Marshal(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}
