// Package json provides JSON encoding and decoding for XDB values.
package json

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"encoding/json/v2"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/core"
)

// Codec implements the KVCodec interface using JSON encoding.
type Codec struct {
	idSep   string
	attrSep string
}

// New creates a new JSON codec.
func New() *Codec {
	return &Codec{
		idSep:   "/",
		attrSep: ".",
	}
}

// NewWithSep creates a new JSON codec with the given
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
	TypeID core.TypeID `json:"t"`
	Data   any         `json:"d"`
}

func marshalValue(v *core.Value) ([]byte, error) {
	if v == nil || v.IsNil() {
		return json.Marshal(value{TypeID: core.TypeIDUnknown, Data: nil})
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

		return json.Marshal(value{TypeID: core.TypeIDArray, Data: data})
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

		return json.Marshal(value{TypeID: core.TypeIDMap, Data: data})
	case core.TypeIDTime:
		unixtime := v.Unwrap().(time.Time).UnixMilli()

		return json.Marshal(value{TypeID: core.TypeIDTime, Data: unixtime})
	default:
		return json.Marshal(value{TypeID: v.Type().ID(), Data: v.Unwrap()})
	}
}

func unmarshalValue(b []byte) (*core.Value, error) {
	if len(b) == 0 {
		return nil, nil
	}

	var decoded value
	if err := json.Unmarshal(b, &decoded); err != nil {
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
			var elemBytes []byte
			var ok bool

			// Handle both []byte and base64 string cases
			if elemBytes, ok = elem.([]byte); !ok {
				if elemStr, strOk := elem.(string); strOk {
					// Decode base64 string back to bytes
					var err error
					elemBytes, err = base64.StdEncoding.DecodeString(elemStr)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, codec.ErrDecodingValue
				}
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
		var unixtime int64
		var ok bool

		// Handle both int64 and float64 cases (JSON unmarshals numbers as float64)
		if unixtime, ok = decoded.Data.(int64); !ok {
			if f, floatOk := decoded.Data.(float64); floatOk {
				unixtime = int64(f)
			} else {
				return nil, codec.ErrDecodingValue
			}
		}

		t := time.UnixMilli(unixtime).UTC()

		return core.NewSafeValue(t)
	case core.TypeIDInteger:
		// JSON unmarshals numbers as float64, so we need to convert back to int64
		if f, ok := decoded.Data.(float64); ok {
			return core.NewSafeValue(int64(f))
		}
		return core.NewSafeValue(decoded.Data)
	case core.TypeIDUnsigned:
		// JSON unmarshals numbers as float64, so we need to convert back to uint64
		if f, ok := decoded.Data.(float64); ok {
			return core.NewSafeValue(uint64(f))
		}
		return core.NewSafeValue(decoded.Data)
	case core.TypeIDBytes:
		// JSON unmarshals []byte as base64 string, so we need to decode it back
		if s, ok := decoded.Data.(string); ok {
			decodedBytes, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return nil, err
			}
			return core.NewSafeValue(decodedBytes)
		}
		return core.NewSafeValue(decoded.Data)
	default:
		return core.NewSafeValue(decoded.Data)
	}
}
