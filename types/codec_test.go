package types

import (
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestNewCodec(t *testing.T) {
	c := New("sqlite")
	assert.Equal(t, "sqlite", c.Name())
}

func TestCodecEncodePassthrough(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	got, err := c.Encode(core.StringVal("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestCodecDecodePassthrough(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	got, err := c.Decode(core.TypeString, "hello")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, core.TIDString, got.Type().ID())
	assert.Equal(t, "hello", got.Unwrap())
}

func TestCodecTypeName(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	got, err := c.TypeName(core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, "TEXT", got)
}

func TestCodecUnregistered(t *testing.T) {
	c := New("test")

	t.Run("encode", func(t *testing.T) {
		_, err := c.Encode(core.IntVal(42))
		assert.ErrorIs(t, err, ErrNoMapping)
	})

	t.Run("decode", func(t *testing.T) {
		_, err := c.Decode(core.TypeInt, int64(42))
		assert.ErrorIs(t, err, ErrNoMapping)
	})

	t.Run("type name", func(t *testing.T) {
		_, err := c.TypeName(core.TypeString)
		assert.ErrorIs(t, err, ErrNoMapping)
	})
}

func TestCodecRegisterReplaces(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "VARCHAR"))

	got, err := c.TypeName(core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, "VARCHAR", got)

	c.Register(Passthrough(core.TypeString, "TEXT"))

	got, err = c.TypeName(core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, "TEXT", got)
}

func TestCodecCustomMapping(t *testing.T) {
	c := New("sqlite")
	c.Register(Mapping{
		Type:     core.TypeBool,
		TypeName: "INTEGER",
		Encode: func(v *core.Value) (driver.Value, error) {
			b, err := v.AsBool()
			if err != nil {
				return nil, err
			}
			if b {
				return int64(1), nil
			}
			return int64(0), nil
		},
		Decode: func(_ core.Type, src any) (*core.Value, error) {
			n, ok := src.(int64)
			if !ok {
				return nil, fmt.Errorf("expected int64, got %T", src)
			}
			return core.BoolVal(n != 0), nil
		},
	})

	t.Run("encode true", func(t *testing.T) {
		got, err := c.Encode(core.BoolVal(true))
		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
	})

	t.Run("encode false", func(t *testing.T) {
		got, err := c.Encode(core.BoolVal(false))
		require.NoError(t, err)
		assert.Equal(t, int64(0), got)
	})

	t.Run("decode 1", func(t *testing.T) {
		got, err := c.Decode(core.TypeBool, int64(1))
		require.NoError(t, err)
		assert.Equal(t, true, got.Unwrap())
	})

	t.Run("decode 0", func(t *testing.T) {
		got, err := c.Decode(core.TypeBool, int64(0))
		require.NoError(t, err)
		assert.Equal(t, false, got.Unwrap())
	})

	t.Run("type name", func(t *testing.T) {
		got, err := c.TypeName(core.TypeBool)
		require.NoError(t, err)
		assert.Equal(t, "INTEGER", got)
	})
}

func TestCodecCustomTID(t *testing.T) {
	const TIDMoney core.TID = "MONEY"
	moneyType := core.NewType(TIDMoney)

	c := New("test")
	c.Register(Mapping{
		Type:     moneyType,
		TypeName: "INTEGER",
		Encode: func(v *core.Value) (driver.Value, error) {
			return v.Unwrap().(int64), nil
		},
		Decode: func(_ core.Type, src any) (*core.Value, error) {
			cents, ok := src.(int64)
			if !ok {
				return nil, fmt.Errorf("expected int64, got %T", src)
			}
			return core.NewTypedValue(moneyType, cents), nil
		},
	})

	t.Run("encode", func(t *testing.T) {
		v := core.NewTypedValue(moneyType, int64(1999))
		got, err := c.Encode(v)
		require.NoError(t, err)
		assert.Equal(t, int64(1999), got)
	})

	t.Run("decode", func(t *testing.T) {
		got, err := c.Decode(moneyType, int64(1999))
		require.NoError(t, err)
		assert.Equal(t, TIDMoney, got.Type().ID())
		assert.Equal(t, int64(1999), got.Unwrap())
	})

	t.Run("type name", func(t *testing.T) {
		got, err := c.TypeName(moneyType)
		require.NoError(t, err)
		assert.Equal(t, "INTEGER", got)
	})
}

func TestCodecEncodeNil(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	got, err := c.Encode(nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPassthroughRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		typ  core.Type
		val  *core.Value
	}{
		{"string", core.TypeString, core.StringVal("hello")},
		{"integer", core.TypeInt, core.IntVal(42)},
		{"float", core.TypeFloat, core.FloatVal(3.14)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New("test")
			c.Register(Passthrough(tt.typ, "IGNORED"))

			encoded, err := c.Encode(tt.val)
			require.NoError(t, err)

			decoded, err := c.Decode(tt.typ, encoded)
			require.NoError(t, err)
			assert.Equal(t, tt.val.Unwrap(), decoded.Unwrap())
		})
	}
}

// --- Column (driver.Valuer + sql.Scanner) ---

func TestColumnValue(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	col := c.Column(core.TypeString, core.StringVal("hello"))

	got, err := col.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestColumnScan(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	col := c.Column(core.TypeString, nil)
	err := col.Scan("hello")
	require.NoError(t, err)

	v := col.Val()
	require.NotNil(t, v)
	assert.Equal(t, core.TIDString, v.Type().ID())
	assert.Equal(t, "hello", v.Unwrap())
}

func TestColumnScanNil(t *testing.T) {
	c := New("test")
	c.Register(Passthrough(core.TypeString, "TEXT"))

	col := c.Column(core.TypeString, nil)
	err := col.Scan(nil)
	require.NoError(t, err)
	assert.True(t, col.Val().IsNil())
}

// --- KV (Marshal / Unmarshal) ---

func TestKVMarshal(t *testing.T) {
	c := New("redis")
	c.Register(Mapping{
		Type:     core.TypeInt,
		TypeName: "string",
		Marshal: func(v *core.Value) ([]byte, error) {
			return []byte(v.String()), nil
		},
		Unmarshal: func(t core.Type, data []byte) (*core.Value, error) {
			return core.NewSafeValue(string(data))
		},
	})

	kv := c.KV(core.TypeInt, core.IntVal(42))

	got, err := kv.Marshal()
	require.NoError(t, err)
	assert.Equal(t, []byte("42"), got)
}

func TestKVUnmarshal(t *testing.T) {
	c := New("redis")
	c.Register(Mapping{
		Type:     core.TypeString,
		TypeName: "string",
		Marshal: func(v *core.Value) ([]byte, error) {
			return []byte(v.String()), nil
		},
		Unmarshal: func(_ core.Type, data []byte) (*core.Value, error) {
			return core.StringVal(string(data)), nil
		},
	})

	kv := c.KV(core.TypeString, nil)
	err := kv.Unmarshal([]byte("hello"))
	require.NoError(t, err)

	v := kv.Val()
	require.NotNil(t, v)
	assert.Equal(t, core.TIDString, v.Type().ID())
	assert.Equal(t, "hello", v.Unwrap())
}

func TestKVUnmarshalNil(t *testing.T) {
	c := New("redis")
	c.Register(Mapping{
		Type:     core.TypeString,
		TypeName: "string",
		Marshal: func(v *core.Value) ([]byte, error) {
			return []byte(v.String()), nil
		},
		Unmarshal: func(_ core.Type, data []byte) (*core.Value, error) {
			return core.StringVal(string(data)), nil
		},
	})

	kv := c.KV(core.TypeString, nil)
	err := kv.Unmarshal(nil)
	require.NoError(t, err)
	assert.True(t, kv.Val().IsNil())
}

func TestCodecMarshalUnmarshal(t *testing.T) {
	c := New("redis")
	c.Register(Mapping{
		Type:     core.TypeBool,
		TypeName: "string",
		Marshal: func(v *core.Value) ([]byte, error) {
			if v.MustBool() {
				return []byte("1"), nil
			}
			return []byte("0"), nil
		},
		Unmarshal: func(_ core.Type, data []byte) (*core.Value, error) {
			return core.BoolVal(string(data) == "1"), nil
		},
	})

	t.Run("marshal true", func(t *testing.T) {
		got, err := c.Marshal(core.BoolVal(true))
		require.NoError(t, err)
		assert.Equal(t, []byte("1"), got)
	})

	t.Run("unmarshal", func(t *testing.T) {
		got, err := c.Unmarshal(core.TypeBool, []byte("0"))
		require.NoError(t, err)
		assert.Equal(t, false, got.Unwrap())
	})

	t.Run("marshal nil", func(t *testing.T) {
		got, err := c.Marshal(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestCodecMarshalUnregistered(t *testing.T) {
	c := New("test")

	_, err := c.Marshal(core.IntVal(42))
	assert.ErrorIs(t, err, ErrNoMapping)
}

func TestCodecUnmarshalUnregistered(t *testing.T) {
	c := New("test")

	_, err := c.Unmarshal(core.TypeInt, []byte("42"))
	assert.ErrorIs(t, err, ErrNoMapping)
}
