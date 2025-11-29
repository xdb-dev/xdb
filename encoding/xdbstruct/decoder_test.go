package xdbstruct_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

func TestDecoder_BasicDecoding(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Email string `xdb:"email"`
	}

	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John Doe")
	record.Set("email", "john@example.com")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
}

type userWithSetters struct {
	id     string
	ns     string
	schema string
	Name   string `xdb:"name"`
}

func (u *userWithSetters) SetID(id string) {
	u.id = id
}

func (u *userWithSetters) SetNS(ns string) {
	u.ns = ns
}

func (u *userWithSetters) SetSchema(schema string) {
	u.schema = schema
}

func TestDecoder_InterfaceSetters(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John Doe")

	decoder := xdbstruct.NewDefaultDecoder()

	var user userWithSetters
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.id)
	assert.Equal(t, "com.example", user.ns)
	assert.Equal(t, "users", user.schema)
	assert.Equal(t, "John Doe", user.Name)
}

func TestDecoder_NestedStructUnflattening(t *testing.T) {
	type Address struct {
		Street string `xdb:"street"`
		City   string `xdb:"city"`
		Zip    string `xdb:"zip"`
	}

	type User struct {
		ID      string  `xdb:"id,primary_key"`
		Name    string  `xdb:"name"`
		Address Address `xdb:"address"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John Doe")
	record.Set("address.street", "123 Main St")
	record.Set("address.city", "Boston")
	record.Set("address.zip", "02101")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "123 Main St", user.Address.Street)
	assert.Equal(t, "Boston", user.Address.City)
	assert.Equal(t, "02101", user.Address.Zip)
}

func TestDecoder_DeeplyNestedStruct(t *testing.T) {
	type Location struct {
		Lat float64 `xdb:"lat"`
		Lon float64 `xdb:"lon"`
	}

	type Address struct {
		Street   string   `xdb:"street"`
		Location Location `xdb:"location"`
	}

	type User struct {
		ID      string  `xdb:"id,primary_key"`
		Address Address `xdb:"address"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("address.street", "123 Main St")
	record.Set("address.location.lat", 42.3601)
	record.Set("address.location.lon", -71.0589)

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123 Main St", user.Address.Street)
	assert.Equal(t, 42.3601, user.Address.Location.Lat)
	assert.Equal(t, -71.0589, user.Address.Location.Lon)
}

func TestDecoder_BasicTypes(t *testing.T) {
	type AllTypes struct {
		ID       string  `xdb:"id,primary_key"`
		BoolVal  bool    `xdb:"bool_val"`
		IntVal   int     `xdb:"int_val"`
		Int64Val int64   `xdb:"int64_val"`
		UintVal  uint    `xdb:"uint_val"`
		FloatVal float64 `xdb:"float_val"`
		StrVal   string  `xdb:"str_val"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("bool_val", true)
	record.Set("int_val", 42)
	record.Set("int64_val", int64(9223372036854775807))
	record.Set("uint_val", uint(100))
	record.Set("float_val", 3.14159)
	record.Set("str_val", "hello")

	decoder := xdbstruct.NewDefaultDecoder()

	var obj AllTypes
	err := decoder.Decode(record, &obj)
	require.NoError(t, err)

	assert.Equal(t, "123", obj.ID)
	assert.Equal(t, true, obj.BoolVal)
	assert.Equal(t, 42, obj.IntVal)
	assert.Equal(t, int64(9223372036854775807), obj.Int64Val)
	assert.Equal(t, uint(100), obj.UintVal)
	assert.Equal(t, 3.14159, obj.FloatVal)
	assert.Equal(t, "hello", obj.StrVal)
}

func TestDecoder_ArraysAndSlices(t *testing.T) {
	type User struct {
		ID       string   `xdb:"id,primary_key"`
		Tags     []string `xdb:"tags"`
		Scores   []int    `xdb:"scores"`
		Features []bool   `xdb:"features"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("tags", []string{"go", "rust", "python"})
	record.Set("scores", []int{10, 20, 30})
	record.Set("features", []bool{true, false, true})

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, []string{"go", "rust", "python"}, user.Tags)
	assert.Equal(t, []int{10, 20, 30}, user.Scores)
	assert.Equal(t, []bool{true, false, true}, user.Features)
}

func TestDecoder_ByteSlice(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Data []byte `xdb:"data"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("data", []byte("hello world"))

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, []byte("hello world"), user.Data)
}

func TestDecoder_CustomUnmarshaler(t *testing.T) {
	type User struct {
		ID       string          `xdb:"id,primary_key"`
		Metadata json.RawMessage `xdb:"metadata"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("metadata", []byte(`{"role":"admin","level":5}`))

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, json.RawMessage(`{"role":"admin","level":5}`), user.Metadata)
}

func TestDecoder_PointerFields(t *testing.T) {
	type User struct {
		ID    string  `xdb:"id,primary_key"`
		Name  *string `xdb:"name"`
		Age   *int    `xdb:"age"`
		Email *string `xdb:"email"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John Doe")
	record.Set("age", 30)

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	require.NotNil(t, user.Name)
	assert.Equal(t, "John Doe", *user.Name)

	require.NotNil(t, user.Age)
	assert.Equal(t, 30, *user.Age)

	assert.Nil(t, user.Email)
}

func TestDecoder_SkipFields(t *testing.T) {
	type User struct {
		ID       string `xdb:"id,primary_key"`
		Name     string `xdb:"name"`
		Internal string `xdb:"-"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John Doe")
	record.Set("internal", "should be ignored")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "", user.Internal)
}

func TestDecoder_ErrorNotPointer(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	record := core.NewRecord("", "", "123")
	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, xdbstruct.ErrNotImplemented)
}

func TestDecoder_ErrorNotStruct(t *testing.T) {
	record := core.NewRecord("", "", "123")
	decoder := xdbstruct.NewDefaultDecoder()

	tests := []struct {
		name  string
		input any
	}{
		{"string", new(string)},
		{"int", new(int)},
		{"slice", new([]string)},
		{"map", new(map[string]string)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := decoder.Decode(record, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, xdbstruct.ErrNotImplemented)
		})
	}
}

func TestDecoder_TypeMismatch(t *testing.T) {
	type User struct {
		ID  string `xdb:"id,primary_key"`
		Age int    `xdb:"age"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("age", "not a number")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	assert.Error(t, err)
}

func TestDecoder_MissingFields(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Email string `xdb:"email"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John Doe")

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "", user.Email)
}

func TestDecoder_ExtraFields(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John Doe")
	record.Set("extra_field", "should be ignored")
	record.Set("another_field", 42)

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
}

func TestDecoder_CustomTag(t *testing.T) {
	type User struct {
		ID   string `db:"id,primary_key"`
		Name string `db:"name"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "John")

	decoder := xdbstruct.NewDecoder(xdbstruct.Options{Tag: "db"})

	var user User
	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "John", user.Name)
}

func TestDecoder_ZeroValues(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Age   int    `xdb:"age"`
		Score int    `xdb:"score"`
	}

	record := core.NewRecord("", "", "123")
	record.Set("name", "")
	record.Set("age", 0)

	decoder := xdbstruct.NewDefaultDecoder()

	var user User
	user.Score = 100

	err := decoder.Decode(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "", user.Name)
	assert.Equal(t, 0, user.Age)
	assert.Equal(t, 100, user.Score)
}

func TestDecoder_RoundTrip(t *testing.T) {
	type Address struct {
		Street string `xdb:"street"`
		City   string `xdb:"city"`
	}

	type User struct {
		ID      string  `xdb:"id,primary_key"`
		Name    string  `xdb:"name"`
		Age     int     `xdb:"age"`
		Address Address `xdb:"address"`
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "xdb",
		NS:     "com.example",
		Schema: "users",
	})
	decoder := xdbstruct.NewDecoder(xdbstruct.Options{Tag: "xdb"})

	original := User{
		ID:   "123",
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
		},
	}

	record, err := encoder.Encode(&original)
	require.NoError(t, err)

	var decoded User
	err = decoder.Decode(record, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Address.Street, decoded.Address.Street)
	assert.Equal(t, original.Address.City, decoded.Address.City)
}
