package msgpack

import (
	"errors"
	"strings"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/core"
)

// Codec implements the KVCodec interface using MessagePack encoding.
type Codec struct {
	idSep   string
	attrSep string
}

// New creates a new MessagePack codec.
func New() *Codec {
	return &Codec{
		idSep:   "/",
		attrSep: ".",
	}
}

// NewWithSep creates a new MessagePack codec with the given
// id and attr separators.
func NewWithSep(idSep, attrSep string) *Codec {
	return &Codec{
		idSep:   idSep,
		attrSep: attrSep,
	}
}

// EncodeID encodes a [core.ID] to []byte.
func (c *Codec) EncodeID(id core.ID) ([]byte, error) {
	if len(id) == 0 {
		return nil, nil
	}

	return []byte(strings.Join(id, c.idSep)), nil
}

// DecodeID decodes a []byte to a [core.ID].
func (c *Codec) DecodeID(data []byte) (core.ID, error) {
	if len(data) == 0 {
		return nil, nil
	}

	parts := strings.Split(string(data), c.idSep)
	return core.NewID(parts...), nil
}

// EncodeAttr encodes a [core.Attr] to []byte.
func (c *Codec) EncodeAttr(attr core.Attr) ([]byte, error) {
	if len(attr) == 0 {
		return nil, nil
	}

	return []byte(strings.Join(attr, c.attrSep)), nil
}

// DecodeAttr decodes a []byte to a [core.Attr].
func (c *Codec) DecodeAttr(data []byte) (core.Attr, error) {
	if len(data) == 0 {
		return nil, nil
	}

	parts := strings.Split(string(data), c.attrSep)
	return core.NewAttr(parts...), nil
}

// EncodeKey encodes a [core.Key] to key-attr pair.
func (c *Codec) EncodeKey(key *core.Key) ([]byte, []byte, error) {
	encodedID, idErr := c.EncodeID(key.ID())
	encodedAttr, attrErr := c.EncodeAttr(key.Attr())

	return encodedID, encodedAttr, errors.Join(idErr, attrErr)
}

// DecodeKey decodes a key-attr pair to a [core.Key].
func (c *Codec) DecodeKey(id []byte, attr []byte) (*core.Key, error) {
	decodedID, idErr := c.DecodeID(id)
	decodedAttr, attrErr := c.DecodeAttr(attr)

	return core.NewKey(decodedID, decodedAttr), errors.Join(idErr, attrErr)
}

// EncodeValue encodes a [core.Value] to []byte.
func (c *Codec) EncodeValue(value *core.Value) ([]byte, error) {
	if value == nil || value.IsNil() {
		return nil, nil
	}

	return marshalValue(value)
}

// DecodeValue decodes a []byte to a [core.Value].
func (c *Codec) DecodeValue(data []byte) (*core.Value, error) {
	if len(data) == 0 {
		return nil, nil
	}

	return unmarshalValue(data)
}

// value is the msgpack wire format for a core.Value
type value struct {
	TypeID core.TypeID `msgpack:"t"`
	Data   any         `msgpack:"d"`
}

func marshalValue(v *core.Value) ([]byte, error) {
	if v == nil || v.IsNil() {
		return msgpack.Marshal(value{TypeID: core.TypeIDUnknown, Data: nil})
	}

	switch v.Type().ID() {
	case core.TypeIDArray:
		arr := v.Unwrap().([]*core.Value)
		data := make([][]byte, len(arr))

		for i, elem := range arr {
			b, err := marshalValue(elem)
			if err != nil {
				return nil, err
			}
			data[i] = b
		}

		return msgpack.Marshal(value{TypeID: core.TypeIDArray, Data: data})
	case core.TypeIDMap:
		mp := v.Unwrap().(map[*core.Value]*core.Value)
		data := make(map[string][]byte, len(mp))

		for k, val := range mp {
			kb, err := marshalValue(k)
			if err != nil {
				return nil, err
			}

			vb, err := marshalValue(val)
			if err != nil {
				return nil, err
			}

			data[string(kb)] = vb
		}

		return msgpack.Marshal(value{TypeID: core.TypeIDMap, Data: data})
	case core.TypeIDTime:
		unixtime := v.Unwrap().(time.Time).UnixMilli()

		return msgpack.Marshal(value{TypeID: core.TypeIDTime, Data: unixtime})
	default:
		return msgpack.Marshal(value{TypeID: v.Type().ID(), Data: v.Unwrap()})
	}
}

func unmarshalValue(b []byte) (*core.Value, error) {
	var decoded value
	if err := msgpack.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}

	if decoded.TypeID == core.TypeIDUnknown || decoded.Data == nil {
		return nil, nil
	}

	switch decoded.TypeID {
	case core.TypeIDArray:
		rawArr, ok := decoded.Data.([]any)
		if !ok {
			return nil, codec.ErrDecodingValue
		}

		arr := make([]*core.Value, len(rawArr))
		for i, elem := range rawArr {
			elemBytes, ok := elem.([]byte)
			if !ok {
				return nil, codec.ErrDecodingValue
			}
			v, err := unmarshalValue(elemBytes)
			if err != nil {
				return nil, err
			}
			arr[i] = v
		}
		return core.NewSafeValue(arr)
	case core.TypeIDMap:
		rawMap, ok := decoded.Data.(map[string][]byte)
		if !ok {
			return nil, codec.ErrDecodingValue
		}
		mp := make(map[*core.Value]*core.Value, len(rawMap))
		for kStr, vBytes := range rawMap {
			kVal, err := unmarshalValue([]byte(kStr))
			if err != nil {
				return nil, err
			}
			vVal, err := unmarshalValue(vBytes)
			if err != nil {
				return nil, err
			}
			mp[kVal] = vVal
		}
		return core.NewSafeValue(mp)
	case core.TypeIDTime:
		unixtime, ok := decoded.Data.(int64)
		if !ok {
			return nil, codec.ErrDecodingValue
		}

		t := time.UnixMilli(unixtime).UTC()

		return core.NewSafeValue(t)
	default:
		return core.NewSafeValue(decoded.Data)
	}
}
