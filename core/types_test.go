package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTIDString(t *testing.T) {
	tests := []struct {
		tid  TID
		want string
	}{
		{TIDUnknown, "UNKNOWN"},
		{TIDBoolean, "BOOLEAN"},
		{TIDInteger, "INTEGER"},
		{TIDUnsigned, "UNSIGNED"},
		{TIDFloat, "FLOAT"},
		{TIDString, "STRING"},
		{TIDBytes, "BYTES"},
		{TIDTime, "TIME"},
		{TIDArray, "ARRAY"},
		{TIDJSON, "JSON"},
		{TID("CUSTOM"), "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tid.String())
		})
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TID
		wantErr bool
	}{
		{"uppercase", "BOOLEAN", TIDBoolean, false},
		{"lowercase", "boolean", TIDBoolean, false},
		{"mixed case", "Boolean", TIDBoolean, false},
		{"with whitespace", "  FLOAT  ", TIDFloat, false},
		{"integer", "INTEGER", TIDInteger, false},
		{"unsigned", "UNSIGNED", TIDUnsigned, false},
		{"string", "STRING", TIDString, false},
		{"bytes", "BYTES", TIDBytes, false},
		{"time", "TIME", TIDTime, false},
		{"array", "ARRAY", TIDArray, false},
		{"json", "JSON", TIDJSON, false},
		{"unknown", "UNKNOWN", TIDUnknown, false},
		{"invalid", "INVALID", TIDUnknown, true},
		{"empty", "", TIDUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrUnknownType)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTypeID(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want TID
	}{
		{"bool", TypeBool, TIDBoolean},
		{"int", TypeInt, TIDInteger},
		{"unsigned", TypeUnsigned, TIDUnsigned},
		{"float", TypeFloat, TIDFloat},
		{"string", TypeString, TIDString},
		{"bytes", TypeBytes, TIDBytes},
		{"time", TypeTime, TIDTime},
		{"json", TypeJSON, TIDJSON},
		{"unknown", TypeUnknown, TIDUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.typ.ID())
		})
	}
}

func TestTypeString(t *testing.T) {
	assert.Equal(t, "BOOLEAN", TypeBool.String())
	assert.Equal(t, "ARRAY", NewArrayType(TIDString).String())
}

func TestTIDLower(t *testing.T) {
	tests := []struct {
		tid  TID
		want string
	}{
		{TIDString, "string"},
		{TIDInteger, "integer"},
		{TIDUnsigned, "unsigned"},
		{TIDFloat, "float"},
		{TIDBoolean, "boolean"},
		{TIDTime, "time"},
		{TIDBytes, "bytes"},
		{TIDJSON, "json"},
		{TIDArray, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tid.Lower())
		})
	}
}

func TestValueTypes(t *testing.T) {
	// ValueTypes must contain all user-facing types (excluding UNKNOWN).
	assert.Len(t, ValueTypes, 9)
	assert.NotContains(t, ValueTypes, TIDUnknown)

	for _, tid := range ValueTypes {
		_, ok := builtinTypes[tid]
		assert.True(t, ok, "ValueTypes contains unregistered TID: %s", tid)
	}
}

func TestTypeElemTypeID(t *testing.T) {
	assert.Equal(t, TID(""), TypeBool.ElemTypeID())
	assert.Equal(t, TIDString, NewArrayType(TIDString).ElemTypeID())
	assert.Equal(t, TIDInteger, NewArrayType(TIDInteger).ElemTypeID())
}

func TestTypeEquals(t *testing.T) {
	tests := []struct {
		name string
		a    Type
		b    Type
		want bool
	}{
		{"same scalar", TypeBool, TypeBool, true},
		{"different scalar", TypeBool, TypeInt, false},
		{"same array", NewArrayType(TIDString), NewArrayType(TIDString), true},
		{"different array elem", NewArrayType(TIDString), NewArrayType(TIDInteger), false},
		{"scalar vs array", TypeString, NewArrayType(TIDString), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Equals(tt.b))
		})
	}
}

func TestTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{"scalar bool", TypeBool, `{"id":"BOOLEAN"}`},
		{"scalar int", TypeInt, `{"id":"INTEGER"}`},
		{"scalar json", TypeJSON, `{"id":"JSON"}`},
		{"array string", NewArrayType(TIDString), `{"id":"ARRAY","elem_type":"STRING"}`},
		{"array int", NewArrayType(TIDInteger), `{"id":"ARRAY","elem_type":"INTEGER"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.typ)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(data))
		})
	}
}

func TestTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Type
		wantErr bool
	}{
		{"scalar bool", `{"id":"BOOLEAN"}`, TypeBool, false},
		{"scalar unsigned", `{"id":"UNSIGNED"}`, TypeUnsigned, false},
		{"array string", `{"id":"ARRAY","elem_type":"STRING"}`, NewArrayType(TIDString), false},
		{"invalid type", `{"id":"INVALID"}`, TypeUnknown, true},
		{"invalid json", `{bad}`, TypeUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Type
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.want.Equals(got))
		})
	}
}

func TestTypeRoundTrip(t *testing.T) {
	types := []Type{
		TypeBool,
		TypeInt,
		TypeUnsigned,
		TypeFloat,
		TypeString,
		TypeBytes,
		TypeTime,
		TypeJSON,
		NewArrayType(TIDString),
		NewArrayType(TIDInteger),
		NewArrayType(TIDBoolean),
	}

	for _, typ := range types {
		t.Run(typ.String(), func(t *testing.T) {
			data, err := json.Marshal(typ)
			require.NoError(t, err)

			var got Type
			err = json.Unmarshal(data, &got)
			require.NoError(t, err)

			assert.True(t, typ.Equals(got))
		})
	}
}
