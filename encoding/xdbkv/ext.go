package xdbkv

import (
	"bytes"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/xdb-dev/xdb/types"
)

type boolExt struct {
	types.Bool
}

func (b *boolExt) MarshalMsgpack() ([]byte, error) {
	if b.Bool {
		return []byte{1}, nil
	}
	return []byte{0}, nil
}

func (b *boolExt) UnmarshalMsgpack(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	b.Bool = data[0] == 1
	return nil
}

type int64Ext struct {
	types.Int64
}

func (i *int64Ext) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(i.Int64)
}

func (i *int64Ext) UnmarshalMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, &i.Int64)
}

type uint64Ext struct {
	types.Uint64
}

func (u *uint64Ext) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(u.Uint64)
}

func (u *uint64Ext) UnmarshalMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, &u.Uint64)
}

type float64Ext struct {
	types.Float64
}

func (f *float64Ext) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(f.Float64)
}

func (f *float64Ext) UnmarshalMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, &f.Float64)
}

type stringExt struct {
	types.String
}

func (s *stringExt) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(string(s.String))
}

func (s *stringExt) UnmarshalMsgpack(data []byte) error {
	var str string
	err := msgpack.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	s.String = types.String(str)
	return nil
}

type bytesExt struct {
	types.Bytes
}

func (b *bytesExt) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal([]byte(b.Bytes))
}

func (b *bytesExt) UnmarshalMsgpack(data []byte) error {
	var bs []byte
	err := msgpack.Unmarshal(data, &bs)
	if err != nil {
		return err
	}
	b.Bytes = types.Bytes(bs)
	return nil
}

type timeExt struct {
	types.Time
}

func (t *timeExt) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(time.Time(t.Time).UnixMilli())
}

func (t *timeExt) UnmarshalMsgpack(data []byte) error {
	var ms int64
	err := msgpack.Unmarshal(data, &ms)
	if err != nil {
		return err
	}
	t.Time = types.Time(time.UnixMilli(ms).UTC())
	return nil
}

type arrayExt struct {
	*types.Array
}

func (a *arrayExt) MarshalMsgpack() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := msgpack.NewEncoder(buf)
	values := a.Array.Values()

	enc.EncodeInt(int64(a.ValueType()))
	enc.EncodeArrayLen(len(values))

	for _, v := range values {
		vv, err := EncodeValue(v)
		if err != nil {
			return nil, err
		}

		enc.EncodeBytes(vv)
	}

	return buf.Bytes(), nil
}

func (a *arrayExt) UnmarshalMsgpack(data []byte) error {
	dec := msgpack.NewDecoder(bytes.NewReader(data))

	typeID, err := dec.DecodeInt()
	if err != nil {
		return err
	}

	arrLen, err := dec.DecodeArrayLen()
	if err != nil {
		return err
	}

	values := make([]types.Value, 0, arrLen)

	for i := 0; i < arrLen; i++ {
		bs, err := dec.DecodeBytes()
		if err != nil {
			return err
		}

		vv, err := DecodeValue(bs)
		if err != nil {
			return err
		}

		values = append(values, vv)
	}

	a.Array = types.NewArray(types.TypeID(typeID), values...)

	return nil
}

func init() {
	msgpack.RegisterExt(1, &boolExt{})
	msgpack.RegisterExt(2, &int64Ext{})
	msgpack.RegisterExt(3, &uint64Ext{})
	msgpack.RegisterExt(4, &float64Ext{})
	msgpack.RegisterExt(5, &stringExt{})
	msgpack.RegisterExt(6, &bytesExt{})
	msgpack.RegisterExt(7, &timeExt{})
	msgpack.RegisterExt(8, &arrayExt{})
}
