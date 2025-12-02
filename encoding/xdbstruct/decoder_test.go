package xdbstruct_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

var defaultDecoder = xdbstruct.NewDefaultDecoder()

func TestDecoder_BasicDecoding(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("email", "john@example.com")

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
}

func TestDecoder_Decode_withInterfaces(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John Doe")

	var account Account
	err := defaultDecoder.FromRecord(record, &account)
	require.NoError(t, err)

	assert.Equal(t, "123", account.GetID())
	assert.Equal(t, "com.example", account.GetNS())
	assert.Equal(t, "users", account.GetSchema())
	assert.Equal(t, "John Doe", account.Name)
}

func TestDecoder_SkipFields(t *testing.T) {
	type User struct {
		ID       string `xdb:"id,primary_key"`
		Name     string `xdb:"name"`
		Internal string `xdb:"-"`
	}

	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John").
		Set("internal", "should be skipped")

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John", user.Name)
	assert.Equal(t, "", user.Internal)
}

func TestDecoder_NestedStruct(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("address.street", "123 Main St").
		Set("address.location.lat", 42.3601).
		Set("address.location.lon", -71.0589)

	var account Account
	err := defaultDecoder.FromRecord(record, &account)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", account.Name)
	assert.Equal(t, "123 Main St", account.Address.Street)
	assert.Equal(t, 42.3601, account.Address.Location.Lat)
	assert.Equal(t, -71.0589, account.Address.Location.Lon)
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

	record := core.NewRecord("com.example", "all_types", "123").
		Set("bool_val", true).
		Set("int_val", 42).
		Set("int64_val", int64(9223372036854775807)).
		Set("uint_val", uint(100)).
		Set("float_val", 3.14159).
		Set("str_val", "hello")

	var obj AllTypes
	err := defaultDecoder.FromRecord(record, &obj)
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

	record := core.NewRecord("com.example", "users", "123").
		Set("tags", []string{"go", "rust", "python"}).
		Set("scores", []int{10, 20, 30}).
		Set("features", []bool{true, false, true})

	var user User
	err := defaultDecoder.FromRecord(record, &user)
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

	record := core.NewRecord("com.example", "users", "123").
		Set("data", []byte("hello world"))

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, []byte("hello world"), user.Data)
}

func TestDecoder_CustomUnmarshaler(t *testing.T) {
	type User struct {
		ID       string          `xdb:"id,primary_key"`
		Metadata json.RawMessage `xdb:"metadata"`
	}

	record := core.NewRecord("com.example", "users", "123").
		Set("metadata", []byte(`{"role":"admin","level":5}`))

	var user User
	err := defaultDecoder.FromRecord(record, &user)
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

	record := core.NewRecord("com.example", "users", "123").
		Set("name", "John Doe").
		Set("age", 30)

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	require.NotNil(t, user.Name)
	assert.Equal(t, "John Doe", *user.Name)

	require.NotNil(t, user.Age)
	assert.Equal(t, 30, *user.Age)

	assert.Nil(t, user.Email)
}

func TestDecoder_ErrorNotPointer(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	record := core.NewRecord("com.example", "users", "123")

	var user User
	err := defaultDecoder.FromRecord(record, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, xdbstruct.ErrNotPointer)
}

func TestDecoder_ErrorNotStruct(t *testing.T) {
	record := core.NewRecord("com.example", "users", "123")

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
			err := defaultDecoder.FromRecord(record, tt.input)
			assert.Error(t, err)
			assert.ErrorIs(t, err, xdbstruct.ErrNotStruct)
		})
	}
}

func TestDecoder_TypeMismatch(t *testing.T) {
	type User struct {
		ID  string `xdb:"id,primary_key"`
		Age int    `xdb:"age"`
	}

	record := core.NewRecord("com.example", "users", "123")
	record.Set("age", "not a number")

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	assert.Error(t, err)
}

func TestDecoder_MissingFields(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Email string `xdb:"email"`
	}

	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John Doe")

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "", user.Email)
}

func TestDecoder_ExtraFields(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John Doe").
		Set("extra_field", "should be ignored").
		Set("another_field", 42)

	var user User
	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.Name)
}

func TestDecoder_ZeroValues(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Age   int    `xdb:"age"`
		Score int    `xdb:"score"`
	}

	record := core.NewRecord("com.example", "users", "123").
		Set("name", "").
		Set("age", 0)

	var user User
	user.Score = 100

	err := defaultDecoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "", user.Name)
	assert.Equal(t, 0, user.Age)
	assert.Equal(t, 100, user.Score)
}

func TestDecoder_CustomTag(t *testing.T) {
	type User struct {
		ID   string `db:"id,primary_key"`
		Name string `db:"name"`
	}

	record := core.NewRecord("com.example", "users", "123")
	record.Set("name", "John")

	decoder := xdbstruct.NewDecoder(xdbstruct.Options{Tag: "db"})

	var user User
	err := decoder.FromRecord(record, &user)
	require.NoError(t, err)

	assert.Equal(t, "123", user.ID)
	assert.Equal(t, "John", user.Name)
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

	original := User{
		ID:   "123",
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Boston",
		},
	}

	record, err := encoder.ToRecord(&original)
	require.NoError(t, err)

	var decoded User
	err = defaultDecoder.FromRecord(record, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Age, decoded.Age)
	assert.Equal(t, original.Address.Street, decoded.Address.Street)
	assert.Equal(t, original.Address.City, decoded.Address.City)
}
