package xdbredis

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestCodecRoundTrip(t *testing.T) {
	ts := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value *core.Value
	}{
		{"bool true", core.BoolVal(true)},
		{"bool false", core.BoolVal(false)},
		{"int positive", core.IntVal(42)},
		{"int negative", core.IntVal(-100)},
		{"int zero", core.IntVal(0)},
		{"uint", core.UintVal(999)},
		{"uint zero", core.UintVal(0)},
		{"float", core.FloatVal(4.5)},
		{"float negative", core.FloatVal(-3.14)},
		{"float zero", core.FloatVal(0)},
		{"string", core.StringVal("hello world")},
		{"string empty", core.StringVal("")},
		{"string with colon", core.StringVal("s:tricky")},
		{"bytes", core.BytesVal([]byte{0xDE, 0xAD, 0xBE, 0xEF})},
		{"bytes empty", core.BytesVal([]byte{})},
		{"time", core.TimeVal(ts)},
		{"json object", core.JSONVal(json.RawMessage(`{"key":"val"}`))},
		{"json array", core.JSONVal(json.RawMessage(`[1,2,3]`))},
		{
			"array of strings",
			core.ArrayVal(
				core.TIDString,
				core.StringVal("a"),
				core.StringVal("b"),
				core.StringVal("c"),
			),
		},
		{
			"array of ints",
			core.ArrayVal(
				core.TIDInteger,
				core.IntVal(1),
				core.IntVal(2),
				core.IntVal(3),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encodeValue(tt.value)
			require.NoError(t, err)

			decoded, err := decodeValue(encoded)
			require.NoError(t, err)

			assert.Equal(t, tt.value.Type().ID(), decoded.Type().ID())
			assert.Equal(t, tt.value.String(), decoded.String())

			if tt.value.Type().ID() == core.TIDArray {
				assert.Equal(t,
					tt.value.Type().ElemTypeID(),
					decoded.Type().ElemTypeID(),
				)
			}
		})
	}
}

func TestEncodePrefix(t *testing.T) {
	tests := []struct {
		name   string
		value  *core.Value
		prefix string
	}{
		{"bool", core.BoolVal(true), "b:"},
		{"int", core.IntVal(42), "i:"},
		{"uint", core.UintVal(1), "u:"},
		{"float", core.FloatVal(1.5), "f:"},
		{"string", core.StringVal("hi"), "s:"},
		{"bytes", core.BytesVal([]byte{1}), "x:"},
		{"time", core.TimeVal(time.Now()), "t:"},
		{"json", core.JSONVal(json.RawMessage(`{}`)), "j:"},
		{
			"array",
			core.ArrayVal(core.TIDString, core.StringVal("a")),
			"a:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encodeValue(tt.value)
			require.NoError(t, err)
			assert.True(t, len(encoded) >= 2)
			assert.Equal(t, tt.prefix, encoded[:2])
		})
	}
}

func TestDecodeInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single char", "x"},
		{"unknown prefix", "z:data"},
		{"bad bool", "b:notabool"},
		{"bad int", "i:notanint"},
		{"bad uint", "u:notauint"},
		{"bad float", "f:notafloat"},
		{"bad base64", "x:!!!invalid!!!"},
		{"bad time", "t:not-a-time"},
		{"bad array json", "a:{invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeValue(tt.input)
			require.Error(t, err)
		})
	}
}
