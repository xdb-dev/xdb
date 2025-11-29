package xdbstruct_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
)

var defaultEncoder = xdbstruct.NewDefaultEncoder("com.example", "users")

type User struct {
	ID    string `xdb:"id,primary_key"`
	Name  string `xdb:"name"`
	Email string `xdb:"email"`
}

type UserWithMetadata struct {
	User
	NS     string
	Schema string
}

func (u *UserWithMetadata) GetNS() string {
	return u.NS
}

func (u *UserWithMetadata) GetSchema() string {
	return u.Schema
}

func TestEncoder_BasicEncoding(t *testing.T) {
	user := User{
		ID:    "123",
		Name:  "John Doe",
		Email: "john@example.com",
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, "com.example", record.NS().String())
	assert.Equal(t, "users", record.Schema().String())
	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "xdb://com.example/users/123", record.URI().String())

	assert.Equal(t, "John Doe", record.Get("name").ToString())
	assert.Equal(t, "john@example.com", record.Get("email").ToString())
}

func TestEncoder_Encode_withInterfaces(t *testing.T) {
	user := UserWithMetadata{
		User: User{
			ID:    "123",
			Name:  "John Doe",
			Email: "john@example.com",
		},
		NS:     "custom.ns",
		Schema: "custom.schema",
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	assert.Equal(t, "custom.ns", record.NS().String())
	assert.Equal(t, "custom.schema", record.Schema().String())
}

type userWithID struct {
	Name string `xdb:"name"`
	id   string
}

func (u *userWithID) GetID() string {
	return u.id
}

func TestEncoder_InterfaceIDGetter(t *testing.T) {
	u := &userWithID{
		Name: "John",
		id:   "custom-id-456",
	}

	record, err := defaultEncoder.Encode(u)
	require.NoError(t, err)

	assert.Equal(t, "custom-id-456", record.ID().String())
}

func TestEncoder_SkipFields(t *testing.T) {
	type User struct {
		ID       string `xdb:"id,primary_key"`
		Name     string `xdb:"name"`
		Internal string `xdb:"-"`
		secret   string
	}

	user := User{
		ID:       "123",
		Name:     "John",
		Internal: "should be skipped",
		secret:   "unexported",
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	assert.Nil(t, record.Get("internal"))
	assert.Nil(t, record.Get("Internal"))
	assert.Nil(t, record.Get("secret"))
	assert.NotNil(t, record.Get("name"))
}

func TestEncoder_NestedStruct(t *testing.T) {
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

	user := User{
		ID: "123",
		Address: Address{
			Street: "123 Main St",
			Location: Location{
				Lat: 42.3601,
				Lon: -71.0589,
			},
		},
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	assert.Equal(t, "123 Main St", record.Get("address.street").ToString())
	assert.Equal(t, 42.3601, record.Get("address.location.lat").ToFloat())
	assert.Equal(t, -71.0589, record.Get("address.location.lon").ToFloat())
}

func TestEncoder_BasicTypes(t *testing.T) {
	type AllTypes struct {
		ID       string  `xdb:"id,primary_key"`
		BoolVal  bool    `xdb:"bool_val"`
		IntVal   int     `xdb:"int_val"`
		Int64Val int64   `xdb:"int64_val"`
		UintVal  uint    `xdb:"uint_val"`
		FloatVal float64 `xdb:"float_val"`
		StrVal   string  `xdb:"str_val"`
	}

	record, err := defaultEncoder.Encode(&AllTypes{
		ID:       "123",
		BoolVal:  true,
		IntVal:   42,
		Int64Val: 9223372036854775807,
		UintVal:  uint(100),
		FloatVal: 3.14159,
		StrVal:   "hello",
	})

	require.NoError(t, err)

	assert.Equal(t, true, record.Get("bool_val").ToBool())
	assert.Equal(t, int64(42), record.Get("int_val").ToInt())
	assert.Equal(t, int64(9223372036854775807), record.Get("int64_val").ToInt())
	assert.Equal(t, uint64(100), record.Get("uint_val").ToUint())
	assert.Equal(t, 3.14159, record.Get("float_val").ToFloat())
	assert.Equal(t, "hello", record.Get("str_val").ToString())
}

func TestEncoder_ArraysAndSlices(t *testing.T) {
	type User struct {
		ID       string   `xdb:"id,primary_key"`
		Tags     []string `xdb:"tags"`
		Scores   []int    `xdb:"scores"`
		Features []bool   `xdb:"features"`
	}

	user := User{
		ID:       "123",
		Tags:     []string{"go", "rust", "python"},
		Scores:   []int{10, 20, 30},
		Features: []bool{true, false, true},
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	tags := record.Get("tags")
	assert.NotNil(t, tags)
	assert.Equal(t, core.TypeIDArray, tags.Value().Type().ID())
	assert.Equal(t, core.TypeIDString, tags.Value().Type().ValueTypeID())
	assert.Equal(t, []string{"go", "rust", "python"}, tags.ToStringArray())

	scores := record.Get("scores")
	assert.NotNil(t, scores)
	assert.Equal(t, core.TypeIDArray, scores.Value().Type().ID())
	assert.Equal(t, core.TypeIDInteger, scores.Value().Type().ValueTypeID())
	assert.Equal(t, []int64{10, 20, 30}, scores.ToIntArray())

	features := record.Get("features")
	assert.NotNil(t, features)
	assert.Equal(t, core.TypeIDArray, features.Value().Type().ID())
	assert.Equal(t, core.TypeIDBoolean, features.Value().Type().ValueTypeID())
	assert.Equal(t, []bool{true, false, true}, features.ToBoolArray())
}

func TestEncoder_ByteSlice(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Data []byte `xdb:"data"`
	}

	user := User{
		ID:   "123",
		Data: []byte("hello world"),
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	data := record.Get("data")
	assert.NotNil(t, data)
	assert.Equal(t, []byte("hello world"), data.ToBytes())
}

func TestEncoder_CustomMarshaler(t *testing.T) {
	type User struct {
		ID       string          `xdb:"id,primary_key"`
		Metadata json.RawMessage `xdb:"metadata"`
	}

	user := User{
		ID:       "123",
		Metadata: json.RawMessage(`{"role":"admin","level":5}`),
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	metadata := record.Get("metadata")
	assert.NotNil(t, metadata)
	assert.Equal(t, core.TypeIDBytes, metadata.Value().Type().ID())
	assert.Equal(t, []byte(`{"role":"admin","level":5}`), metadata.ToBytes())
}

func TestEncoder_PointerFields(t *testing.T) {
	type User struct {
		ID    string  `xdb:"id,primary_key"`
		Name  *string `xdb:"name"`
		Age   *int    `xdb:"age"`
		Email *string `xdb:"email"`
	}

	name := "John Doe"
	age := 30

	user := User{
		ID:    "123",
		Name:  &name,
		Age:   &age,
		Email: nil,
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", record.Get("name").ToString())
	assert.Equal(t, int64(30), record.Get("age").ToInt())
	assert.Nil(t, record.Get("email"))
}

func TestEncoder_ErrorNoPrimaryKey(t *testing.T) {
	type User struct {
		Name  string `xdb:"name"`
		Email string `xdb:"email"`
	}

	user := User{
		Name:  "John",
		Email: "john@example.com",
	}

	record, err := defaultEncoder.Encode(&user)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbstruct.ErrNoPrimaryKey)
}

func TestEncoder_ErrorNotStruct(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"string", "hello"},
		{"int", 42},
		{"slice", []string{"a", "b"}},
		{"map", map[string]string{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := defaultEncoder.Encode(tt.input)
			assert.Error(t, err)
			assert.Nil(t, record)
			assert.ErrorIs(t, err, xdbstruct.ErrNotStruct)
		})
	}
}

func TestEncoder_ZeroValues(t *testing.T) {
	type User struct {
		ID    string `xdb:"id,primary_key"`
		Name  string `xdb:"name"`
		Age   int    `xdb:"age"`
		Email string `xdb:"email"`
	}

	user := User{
		ID:   "123",
		Name: "",
		Age:  0,
	}

	record, err := defaultEncoder.Encode(&user)
	require.NoError(t, err)

	assert.Equal(t, "", record.Get("name").ToString())
	assert.Equal(t, int64(0), record.Get("age").ToInt())
	assert.Equal(t, "", record.Get("email").ToString())
}

func TestEncoder_CustomTag(t *testing.T) {
	type User struct {
		ID   string `db:"id,primary_key"`
		Name string `db:"name"`
	}

	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
		Tag:    "db",
		NS:     "com.example",
		Schema: "users",
	})

	user := User{
		ID:   "123",
		Name: "John",
	}

	record, err := encoder.Encode(&user)
	require.NoError(t, err)

	assert.Equal(t, "123", record.ID().String())
	assert.Equal(t, "John", record.Get("name").ToString())
}

func TestEncoder_ErrorEmptyNS(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	encoder := xdbstruct.NewDefaultEncoder("", "users")

	user := User{
		ID:   "123",
		Name: "John",
	}

	record, err := encoder.Encode(&user)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbstruct.ErrEmptyNamespace)
}

func TestEncoder_ErrorEmptySchema(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	encoder := xdbstruct.NewDefaultEncoder("com.example", "")

	user := User{
		ID:   "123",
		Name: "John",
	}

	record, err := encoder.Encode(&user)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbstruct.ErrEmptySchema)
}

func TestEncoder_ErrorEmptyID(t *testing.T) {
	type User struct {
		ID   string `xdb:"id,primary_key"`
		Name string `xdb:"name"`
	}

	user := User{
		ID:   "", // empty ID
		Name: "John",
	}

	record, err := defaultEncoder.Encode(&user)
	assert.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, xdbstruct.ErrEmptyID)
}
