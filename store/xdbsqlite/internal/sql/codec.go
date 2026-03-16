package sql

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Mapping holds the conversion functions and native type name
// for a single [core.TID].
type Mapping struct {
	// ToDriver converts a [*core.Value] to a [driver.Value] for SQL column writes.
	ToDriver func(*core.Value) (driver.Value, error)
	// FromDriver converts a database-scanned value back to a [*core.Value].
	FromDriver func(core.Type, any) (*core.Value, error)
	// ToBytes serializes a [*core.Value] to bytes for KV storage.
	ToBytes func(*core.Value) ([]byte, error)
	// FromBytes deserializes bytes from KV storage into a [*core.Value].
	FromBytes func(core.Type, []byte) (*core.Value, error)
	Type      core.Type
	TypeName  string
}

// Codec provides type-aware encoding between [core.Value] and SQLite storage.
type Codec struct {
	mappings map[core.TID]Mapping
}

// newCodec creates a Codec with all SQLite type mappings registered.
func newCodec() *Codec {
	c := &Codec{mappings: make(map[core.TID]Mapping)}

	c.register(passthroughJSON(core.TypeString, "TEXT", core.StringVal))
	c.register(passthroughJSON(core.TypeInt, "INTEGER", core.IntVal))
	c.register(passthroughJSON(core.TypeFloat, "REAL", core.FloatVal))
	c.register(boolMapping())
	c.register(unsignedMapping())
	c.register(bytesMapping())
	c.register(timeMapping())
	c.register(jsonMapping())

	return c
}

func (c *Codec) register(m Mapping) {
	c.mappings[m.Type.ID()] = m
}

// ToDriver converts a [*core.Value] to a [driver.Value] for SQL column writes.
// Returns nil for nil values.
func (c *Codec) ToDriver(v *core.Value) (driver.Value, error) {
	if v.IsNil() {
		return nil, nil
	}

	m, ok := c.mappings[v.Type().ID()]
	if !ok || m.ToDriver == nil {
		return nil, fmt.Errorf("codec: no ToDriver mapping for %s", v.Type().ID())
	}

	return m.ToDriver(v)
}

// FromDriver converts a database-scanned value to a [*core.Value].
func (c *Codec) FromDriver(t core.Type, src any) (*core.Value, error) {
	m, ok := c.mappings[t.ID()]
	if !ok || m.FromDriver == nil {
		return nil, fmt.Errorf("codec: no FromDriver mapping for %s", t.ID())
	}

	return m.FromDriver(t, src)
}

// ToBytes serializes a [*core.Value] to bytes for KV storage.
// Returns nil for nil values.
func (c *Codec) ToBytes(v *core.Value) ([]byte, error) {
	if v.IsNil() {
		return nil, nil
	}

	m, ok := c.mappings[v.Type().ID()]
	if !ok || m.ToBytes == nil {
		return nil, fmt.Errorf("codec: no ToBytes mapping for %s", v.Type().ID())
	}

	return m.ToBytes(v)
}

// FromBytes deserializes bytes from KV storage into a [*core.Value].
func (c *Codec) FromBytes(t core.Type, data []byte) (*core.Value, error) {
	m, ok := c.mappings[t.ID()]
	if !ok || m.FromBytes == nil {
		return nil, fmt.Errorf("codec: no FromBytes mapping for %s", t.ID())
	}

	return m.FromBytes(t, data)
}

// TypeName returns the SQLite type name for the given [core.Type].
func (c *Codec) TypeName(t core.Type) (string, error) {
	m, ok := c.mappings[t.ID()]
	if !ok {
		return "", fmt.Errorf("codec: no mapping for %s", t.ID())
	}
	return m.TypeName, nil
}

// --- Mapping helpers ---

// passthrough builds a Mapping where the Go value is already a valid driver.Value.
func passthrough(typ core.Type, typeName string) Mapping {
	return Mapping{
		Type:     typ,
		TypeName: typeName,
		ToDriver: func(v *core.Value) (driver.Value, error) {
			dv, ok := v.Unwrap().(driver.Value)
			if ok {
				return dv, nil
			}
			return nil, fmt.Errorf("codec: %T is not a driver.Value", v.Unwrap())
		},
		FromDriver: func(_ core.Type, src any) (*core.Value, error) {
			return core.NewSafeValue(src)
		},
	}
}

// passthroughJSON extends passthrough with JSON-based ToBytes/FromBytes.
func passthroughJSON[T any](typ core.Type, typeName string, ctor func(T) *core.Value) Mapping {
	m := passthrough(typ, typeName)
	m.ToBytes = jsonToBytes
	m.FromBytes = jsonFromBytes(ctor)
	return m
}

func jsonToBytes(v *core.Value) ([]byte, error) {
	return json.Marshal(v.Unwrap())
}

func jsonFromBytes[T any](ctor func(T) *core.Value) func(core.Type, []byte) (*core.Value, error) {
	return func(_ core.Type, data []byte) (*core.Value, error) {
		var v T
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return ctor(v), nil
	}
}

// --- SQLite type mappings ---

func boolMapping() Mapping {
	return Mapping{
		Type:     core.TypeBool,
		TypeName: "INTEGER",
		ToDriver: func(v *core.Value) (driver.Value, error) {
			b, err := v.AsBool()
			if err != nil {
				return nil, err
			}
			if b {
				return int64(1), nil
			}
			return int64(0), nil
		},
		FromDriver: func(_ core.Type, src any) (*core.Value, error) {
			n, ok := src.(int64)
			if !ok {
				return nil, fmt.Errorf("bool: expected int64, got %T", src)
			}
			return core.BoolVal(n != 0), nil
		},
		ToBytes:   jsonToBytes,
		FromBytes: jsonFromBytes(core.BoolVal),
	}
}

func unsignedMapping() Mapping {
	return Mapping{
		Type:     core.TypeUnsigned,
		TypeName: "INTEGER",
		ToDriver: func(v *core.Value) (driver.Value, error) {
			u, err := v.AsUint()
			if err != nil {
				return nil, err
			}
			return int64(u), nil
		},
		FromDriver: func(_ core.Type, src any) (*core.Value, error) {
			n, ok := src.(int64)
			if !ok {
				return nil, fmt.Errorf("unsigned: expected int64, got %T", src)
			}
			return core.UintVal(uint64(n)), nil
		},
		ToBytes:   jsonToBytes,
		FromBytes: jsonFromBytes(core.UintVal),
	}
}

func bytesMapping() Mapping {
	m := passthrough(core.TypeBytes, "BLOB")
	m.ToBytes = func(v *core.Value) ([]byte, error) {
		b, err := v.AsBytes()
		if err != nil {
			return nil, err
		}
		dst := make([]byte, base64.RawStdEncoding.EncodedLen(len(b)))
		base64.RawStdEncoding.Encode(dst, b)
		return dst, nil
	}
	m.FromBytes = func(_ core.Type, data []byte) (*core.Value, error) {
		b, err := base64.RawStdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, err
		}
		return core.BytesVal(b), nil
	}
	return m
}

func timeMapping() Mapping {
	return Mapping{
		Type:     core.TypeTime,
		TypeName: "TEXT",
		ToDriver: func(v *core.Value) (driver.Value, error) {
			t, err := v.AsTime()
			if err != nil {
				return nil, err
			}
			return t.Format(time.RFC3339Nano), nil
		},
		FromDriver: func(_ core.Type, src any) (*core.Value, error) {
			s, ok := src.(string)
			if !ok {
				return nil, fmt.Errorf("time: expected string, got %T", src)
			}
			t, err := time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return nil, err
			}
			return core.TimeVal(t), nil
		},
		ToBytes:   jsonToBytes,
		FromBytes: jsonFromBytes(core.TimeVal),
	}
}

func jsonMapping() Mapping {
	return Mapping{
		Type:     core.TypeJSON,
		TypeName: "TEXT",
		ToDriver: func(v *core.Value) (driver.Value, error) {
			raw, err := v.AsJSON()
			if err != nil {
				return nil, err
			}
			return string(raw), nil
		},
		FromDriver: func(_ core.Type, src any) (*core.Value, error) {
			switch s := src.(type) {
			case string:
				return core.JSONVal(json.RawMessage(s)), nil
			case []byte:
				return core.JSONVal(json.RawMessage(s)), nil
			default:
				return nil, fmt.Errorf("json: expected string or []byte, got %T", src)
			}
		},
		ToBytes: func(v *core.Value) ([]byte, error) {
			raw, err := v.AsJSON()
			if err != nil {
				return nil, err
			}
			return []byte(raw), nil
		},
		FromBytes: func(_ core.Type, data []byte) (*core.Value, error) {
			return core.JSONVal(json.RawMessage(data)), nil
		},
	}
}
