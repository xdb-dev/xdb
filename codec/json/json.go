// Package json provides JSON encoding and decoding for XDB values.
package json

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/core"
)

// Codec implements the KVCodec interface using JSON encoding.
type Codec struct {
}

// New creates a new JSON codec.
func New() *Codec {
	return &Codec{}
}

// EncodeTuple encodes a [core.Tuple] to key-value pair.
func (c *Codec) EncodeTuple(tuple *core.Tuple) ([]byte, []byte, error) {
	encodedURI, err := c.EncodeURI(tuple.URI())
	if err != nil {
		return nil, nil, err
	}

	encodedValue, err := c.EncodeValue(tuple.Value())
	if err != nil {
		return nil, nil, err
	}

	return encodedURI, encodedValue, nil
}

// DecodeTuple decodes a key-value pair to a [core.Tuple].
func (c *Codec) DecodeTuple(key []byte, value []byte) (*core.Tuple, error) {
	decodedURI, err := c.DecodeURI(key)
	if err != nil {
		return nil, err
	}

	decodedValue, err := c.DecodeValue(value)
	if err != nil {
		return nil, err
	}

	return core.NewTuple(decodedURI.Repo(), decodedURI.ID(), decodedURI.Attr(), decodedValue), nil
}

// EncodeURI encodes a [core.URI] to []byte.
func (c *Codec) EncodeURI(uri *core.URI) ([]byte, error) {
	return []byte(uri.String()), nil
}

// DecodeURI decodes a []byte to a [core.URI].
func (c *Codec) DecodeURI(data []byte) (*core.URI, error) {
	if len(data) == 0 {
		return nil, nil
	}

	return core.ParseURI(string(data))
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
		rawMap, ok := decoded.Data.(map[string]any)
		if !ok {
			return nil, codec.ErrDecodingValue
		}
		mp := make(map[*core.Value]*core.Value, len(rawMap))
		for kStr, vAny := range rawMap {
			kVal, err := unmarshalValue([]byte(kStr))
			if err != nil {
				return nil, err
			}

			var vBytes []byte
			// Handle both []byte and base64 string cases
			if vBytes, ok = vAny.([]byte); !ok {
				if vStr, strOk := vAny.(string); strOk {
					// Decode base64 string back to bytes
					var err error
					vBytes, err = base64.StdEncoding.DecodeString(vStr)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, codec.ErrDecodingValue
				}
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
