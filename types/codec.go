package types

import (
	"database/sql/driver"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

// ErrNoMapping is returned when no [Mapping] is registered for a [core.TID].
var ErrNoMapping = errors.New("[xdb/types] no mapping registered")

// Mapping holds the type, native DB type name, and conversion functions
// for a single [core.TID].
type Mapping struct {
	Encode    func(*core.Value) (driver.Value, error)
	Decode    func(core.Type, any) (*core.Value, error)
	Marshal   func(*core.Value) ([]byte, error)
	Unmarshal func(core.Type, []byte) (*core.Value, error)
	Type      core.Type
	TypeName  string
}

// Codec provides encoding, decoding, and type-name mapping
// for a specific database backend.
type Codec struct {
	mappings map[core.TID]Mapping
	name     string
}

// New creates a new [Codec] with the given backend name.
func New(name string) *Codec {
	return &Codec{
		name:     name,
		mappings: make(map[core.TID]Mapping),
	}
}

// Name returns the backend name of this codec.
func (c *Codec) Name() string {
	return c.name
}

// Register adds a [Mapping], keyed by the mapping's [core.Type] ID.
// If a mapping already exists for the TID, it is replaced.
func (c *Codec) Register(m Mapping) {
	c.mappings[m.Type.ID()] = m
}

// Encode converts an XDB [*core.Value] to a [driver.Value].
// Returns [ErrNoMapping] if no mapping is registered for the value's [core.TID].
func (c *Codec) Encode(v *core.Value) (driver.Value, error) {
	if v.IsNil() {
		return nil, nil
	}

	tid := v.Type().ID()

	m, ok := c.mappings[tid]
	if !ok || m.Encode == nil {
		return nil, errors.Wrap(ErrNoMapping, "tid", tid.String())
	}

	return m.Encode(v)
}

// Decode converts a database-returned value to an XDB [*core.Value],
// given the expected [core.Type].
// Returns [ErrNoMapping] if no mapping is registered for the type's [core.TID].
func (c *Codec) Decode(t core.Type, src any) (*core.Value, error) {
	m, ok := c.mappings[t.ID()]
	if !ok || m.Decode == nil {
		return nil, errors.Wrap(ErrNoMapping, "tid", t.ID().String())
	}

	return m.Decode(t, src)
}

// TypeName returns the database-native type name for the given [core.Type].
// Returns [ErrNoMapping] if no mapping is registered for the type's [core.TID].
func (c *Codec) TypeName(t core.Type) (string, error) {
	m, ok := c.mappings[t.ID()]
	if !ok {
		return "", errors.Wrap(ErrNoMapping, "tid", t.ID().String())
	}

	return m.TypeName, nil
}

// Marshal converts an XDB [*core.Value] to bytes for KV storage.
// Returns [ErrNoMapping] if no mapping is registered for the value's [core.TID].
func (c *Codec) Marshal(v *core.Value) ([]byte, error) {
	if v.IsNil() {
		return nil, nil
	}

	tid := v.Type().ID()

	m, ok := c.mappings[tid]
	if !ok || m.Marshal == nil {
		return nil, errors.Wrap(ErrNoMapping, "tid", tid.String())
	}

	return m.Marshal(v)
}

// Unmarshal converts bytes from KV storage to an XDB [*core.Value],
// given the expected [core.Type].
// Returns [ErrNoMapping] if no mapping is registered for the type's [core.TID].
func (c *Codec) Unmarshal(t core.Type, data []byte) (*core.Value, error) {
	m, ok := c.mappings[t.ID()]
	if !ok || m.Unmarshal == nil {
		return nil, errors.Wrap(ErrNoMapping, "tid", t.ID().String())
	}

	return m.Unmarshal(t, data)
}

// Column returns a [Column] for use with database/sql.
// The returned Column implements [driver.Valuer] and [sql.Scanner].
func (c *Codec) Column(t core.Type, v *core.Value) *Column {
	return &Column{
		codec: c,
		typ:   t,
		val:   v,
	}
}

// Column implements [driver.Valuer] and [sql.Scanner] for use with database/sql.
type Column struct {
	codec *Codec
	val   *core.Value
	typ   core.Type
}

// Value implements [driver.Valuer].
func (col *Column) Value() (driver.Value, error) {
	return col.codec.Encode(col.val)
}

// Scan implements [sql.Scanner].
func (col *Column) Scan(src any) error {
	if src == nil {
		col.val = &core.Value{}
		return nil
	}

	v, err := col.codec.Decode(col.typ, src)
	if err != nil {
		return err
	}

	col.val = v

	return nil
}

// Val returns the underlying [*core.Value].
func (col *Column) Val() *core.Value {
	return col.val
}

// KV returns a [KV] for use with key-value stores.
func (c *Codec) KV(t core.Type, v *core.Value) *KV {
	return &KV{
		codec: c,
		typ:   t,
		val:   v,
	}
}

// KV wraps marshal/unmarshal for key-value stores,
// paralleling [Column] for SQL backends.
type KV struct {
	codec *Codec
	val   *core.Value
	typ   core.Type
}

// Marshal converts the value to bytes.
func (kv *KV) Marshal() ([]byte, error) {
	return kv.codec.Marshal(kv.val)
}

// Unmarshal decodes bytes into the value.
func (kv *KV) Unmarshal(data []byte) error {
	if data == nil {
		kv.val = &core.Value{}
		return nil
	}

	v, err := kv.codec.Unmarshal(kv.typ, data)
	if err != nil {
		return err
	}

	kv.val = v

	return nil
}

// Val returns the underlying [*core.Value].
func (kv *KV) Val() *core.Value {
	return kv.val
}

// Passthrough returns a [Mapping] where the value is passed through
// without conversion. Suitable for types whose Go representation
// is already a valid [driver.Value] (e.g., string, int64, float64, bool).
func Passthrough(t core.Type, typeName string) Mapping {
	return Mapping{
		Type:     t,
		TypeName: typeName,
		Encode: func(v *core.Value) (driver.Value, error) {
			dv, ok := v.Unwrap().(driver.Value)
			if ok {
				return dv, nil
			}
			return nil, errors.Wrap(ErrNoMapping, "not a driver.Value")
		},
		Decode: func(_ core.Type, src any) (*core.Value, error) {
			return core.NewSafeValue(src)
		},
	}
}
