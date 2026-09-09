package xdbstruct_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/encoding/xdbstruct"
	"github.com/xdb-dev/xdb/tests"
)

// Struct fixtures for type mapping and round-trip tests.

// UserID and Age are named scalar types.
type UserID string

type Age int

// Address is a nested value struct.
type Address struct {
	City string `xdb:"city"`
	Zip  string `xdb:"zip"`
}

// Profile is a nested pointer struct.
type Profile struct {
	Bio string `xdb:"bio"`
	Age Age    `xdb:"age"`
}

// Order is an object-array element with a required member.
type Order struct {
	SKU    string    `xdb:"sku,required"`
	Qty    int       `xdb:"qty"`
	Placed time.Time `xdb:"placed"`
}

// Meta is an embedded struct.
type Meta struct {
	Version int `xdb:"version"`
}

// User exercises scalars, time.Time, []byte, []scalar, object arrays, nested
// value/pointer structs, embedded structs, named types, and a json-opt-in map.
type User struct {
	Meta
	ID      UserID            `xdb:"id"`
	Name    string            `xdb:"name,required"`
	Score   float64           `xdb:"score"`
	Active  bool              `xdb:"active"`
	Count   uint64            `xdb:"count"`
	Joined  time.Time         `xdb:"joined"`
	Avatar  []byte            `xdb:"avatar"`
	Tags    []string          `xdb:"tags"`
	Orders  []Order           `xdb:"orders"`
	Address Address           `xdb:"address"`
	Profile *Profile          `xdb:"profile"`
	Attrs   map[string]string `xdb:"attrs,json"`
}

// PtrFields exercises nullable scalar pointers (absent vs present).
type PtrFields struct {
	Name string  `xdb:"name"`
	Nick *string `xdb:"nick"`
}

const userURI = "xdb://com.example/users/u1"

func fullUser() User {
	return User{
		Meta:   Meta{Version: 3},
		ID:     UserID("u1"),
		Name:   "Ada",
		Score:  9.5,
		Active: true,
		Count:  42,
		Joined: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Avatar: []byte{0x01, 0x02, 0x03},
		Tags:   []string{"a", "b"},
		Orders: []Order{
			{SKU: "s1", Qty: 2, Placed: time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)},
			{SKU: "s2", Qty: 5, Placed: time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)},
		},
		Address: Address{City: "Paris", Zip: "75001"},
		Profile: &Profile{Bio: "hi", Age: Age(30)},
		Attrs:   map[string]string{"team": "core"},
	}
}

func TestRoundTrip(t *testing.T) {
	nick := "Ace"

	t.Run("full user", func(t *testing.T) {
		def, err := xdbstruct.Def[User](userURI)
		require.NoError(t, err)

		tests.RunRoundTrip(t, tests.RoundTrip{
			Def:       def,
			Value:     fullUser(),
			Marshal:   func(v any) (*core.Record, error) { return xdbstruct.Marshal(userURI, v) },
			Unmarshal: func(rec *core.Record, dst any) error { return xdbstruct.Unmarshal(rec, dst) },
		})
	})

	t.Run("absent nested pointer and nil slices", func(t *testing.T) {
		def, err := xdbstruct.Def[User](userURI)
		require.NoError(t, err)

		u := User{
			Meta:    Meta{Version: 1},
			ID:      UserID("u2"),
			Name:    "Bob",
			Address: Address{City: "Rome", Zip: "00100"},
			// Profile, Orders, Tags, Avatar, Attrs all absent/nil.
		}

		tests.RunRoundTrip(t, tests.RoundTrip{
			Def:       def,
			Value:     u,
			Marshal:   func(v any) (*core.Record, error) { return xdbstruct.Marshal(userURI, v) },
			Unmarshal: func(rec *core.Record, dst any) error { return xdbstruct.Unmarshal(rec, dst) },
		})
	})

	t.Run("scalar pointer present", func(t *testing.T) {
		def, err := xdbstruct.Def[PtrFields]("xdb://com.example/ptr")
		require.NoError(t, err)

		tests.RunRoundTrip(t, tests.RoundTrip{
			Def:       def,
			Value:     PtrFields{Name: "x", Nick: &nick},
			Marshal:   func(v any) (*core.Record, error) { return xdbstruct.Marshal("xdb://com.example/ptr/p1", v) },
			Unmarshal: func(rec *core.Record, dst any) error { return xdbstruct.Unmarshal(rec, dst) },
		})
	})

	t.Run("scalar pointer absent", func(t *testing.T) {
		def, err := xdbstruct.Def[PtrFields]("xdb://com.example/ptr")
		require.NoError(t, err)

		tests.RunRoundTrip(t, tests.RoundTrip{
			Def:       def,
			Value:     PtrFields{Name: "y", Nick: nil},
			Marshal:   func(v any) (*core.Record, error) { return xdbstruct.Marshal("xdb://com.example/ptr/p2", v) },
			Unmarshal: func(rec *core.Record, dst any) error { return xdbstruct.Unmarshal(rec, dst) },
		})
	})
}

func TestDefFields(t *testing.T) {
	def, err := xdbstruct.Def[User](userURI)
	require.NoError(t, err)
	require.NoError(t, def.Validate())

	// Embedded promotion.
	assert.Equal(t, core.TIDInteger, def.Fields["version"].Type.ID())

	// Named type records go.type annotation.
	idField := def.Fields["id"]
	assert.Equal(t, core.TIDString, idField.Type.ID())
	assert.Equal(t, "xdbstruct_test.UserID", idField.Annotations["go.type"])

	// Required tag.
	assert.True(t, def.Fields["name"].Required)

	// time.Time, []byte.
	assert.Equal(t, core.TIDTime, def.Fields["joined"].Type.ID())
	assert.Equal(t, core.TIDBytes, def.Fields["avatar"].Type.ID())

	// []scalar.
	tags := def.Fields["tags"]
	assert.Equal(t, core.TIDArray, tags.Type.ID())
	assert.Equal(t, core.TIDString, tags.Type.ElemTypeID())

	// Object array with element Items (required member preserved).
	orders := def.Fields["orders"]
	assert.Equal(t, core.TIDArray, orders.Type.ID())
	assert.Equal(t, core.TIDJSON, orders.Type.ElemTypeID())
	assert.Equal(t, core.TIDString, orders.Items["sku"].Type.ID())
	assert.True(t, orders.Items["sku"].Required)
	assert.Equal(t, core.TIDTime, orders.Items["placed"].Type.ID())

	// Nested structs flatten to dotted attributes.
	assert.Equal(t, core.TIDString, def.Fields["address.city"].Type.ID())
	assert.Equal(t, core.TIDString, def.Fields["profile.bio"].Type.ID())
	assert.Equal(t, core.TIDInteger, def.Fields["profile.age"].Type.ID())

	// json opt-in map.
	assert.Equal(t, core.TIDJSON, def.Fields["attrs"].Type.ID())

	// The nested-parent names are namespaces, not fields.
	_, hasProfile := def.Fields["profile"]
	assert.False(t, hasProfile)
}
