// Package json provides JSON encoding and decoding for XDB values.
package json

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/core"
)

// Codec implements the KVCodec interface using JSON encoding.
type Codec struct{}

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
func (c *Codec) DecodeTuple(key, value []byte) (*core.Tuple, error) {
	uri, err := c.DecodeURI(key)
	if err != nil {
		return nil, err
	}

	val, err := c.DecodeValue(value)
	if err != nil {
		return nil, err
	}

	return core.NewTuple(uri.Path(), uri.Attr().String(), val), nil
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

// value is the msgpack wire format for a core.Value.
type value struct {
	TypeID core.TID `json:"t"`
	Data   any      `json:"d"`
}

func marshalValue(v *core.Value) ([]byte, error) {
	if v == nil || v.IsNil() {
		return json.Marshal(value{TypeID: core.TIDUnknown, Data: nil})
	}

	switch v.Type().ID() {
	case core.TIDArray:
		arr := v.Unwrap().([]*core.Value)
		data := make([][]byte, len(arr))

		for i, elem := range arr {
			b, err := marshalValue(elem)
			if err != nil {
				return nil, err
			}
			data[i] = b
		}

		return json.Marshal(value{TypeID: core.TIDArray, Data: data})
	case core.TIDMap:
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

		return json.Marshal(value{TypeID: core.TIDMap, Data: data})
	case core.TIDTime:
		unixtime := v.Unwrap().(time.Time).UnixMilli()

		return json.Marshal(value{TypeID: core.TIDTime, Data: unixtime})
	default:
		return json.Marshal(value{TypeID: v.Type().ID(), Data: v.Unwrap()})
	}
}

func convertToBytes(data any) ([]byte, error) {
	if b, ok := data.([]byte); ok {
		return b, nil
	}

	if s, ok := data.(string); ok {
		return base64.StdEncoding.DecodeString(s)
	}

	return nil, codec.ErrDecodingValue
}

func unmarshalArray(rawArr []any) (*core.Value, error) {
	arr := make([]*core.Value, len(rawArr))
	for i, elem := range rawArr {
		elemBytes, err := convertToBytes(elem)
		if err != nil {
			return nil, err
		}

		v, err := unmarshalValue(elemBytes)
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return core.NewSafeValue(arr)
}

func unmarshalMap(rawMap map[string]any) (*core.Value, error) {
	mp := make(map[*core.Value]*core.Value, len(rawMap))
	for kStr, vAny := range rawMap {
		kVal, err := unmarshalValue([]byte(kStr))
		if err != nil {
			return nil, err
		}

		vBytes, err := convertToBytes(vAny)
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
}

func unmarshalTime(data any) (*core.Value, error) {
	var unixtime int64

	switch n := data.(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return nil, err
		}
		unixtime = i
	case int64:
		unixtime = n
	case float64:
		unixtime = int64(n)
	default:
		return nil, codec.ErrDecodingValue
	}

	t := time.UnixMilli(unixtime).UTC()
	return core.NewSafeValue(t)
}

func unmarshalValue(b []byte) (*core.Value, error) {
	if len(b) == 0 {
		return nil, nil
	}

	var decoded value
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&decoded); err != nil {
		return nil, err
	}

	if decoded.TypeID == core.TIDUnknown || decoded.Data == nil {
		return nil, nil
	}

	return unmarshalTypedData(decoded.TypeID, decoded.Data)
}

func unmarshalTypedData(tid core.TID, data any) (*core.Value, error) {
	switch tid {
	case core.TIDArray:
		rawArr, ok := data.([]any)
		if !ok {
			return nil, codec.ErrDecodingValue
		}
		return unmarshalArray(rawArr)
	case core.TIDMap:
		rawMap, ok := data.(map[string]any)
		if !ok {
			return nil, codec.ErrDecodingValue
		}
		return unmarshalMap(rawMap)
	case core.TIDTime:
		return unmarshalTime(data)
	case core.TIDInteger:
		return unmarshalInt64(data)
	case core.TIDUnsigned:
		return unmarshalUint64(data)
	case core.TIDFloat:
		return unmarshalFloat64(data)
	case core.TIDBytes:
		if s, ok := data.(string); ok {
			decodedBytes, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return nil, err
			}
			return core.NewSafeValue(decodedBytes)
		}
		return core.NewSafeValue(data)
	default:
		return core.NewSafeValue(data)
	}
}

func unmarshalInt64(data any) (*core.Value, error) {
	switch n := data.(type) {
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return nil, err
		}
		return core.NewSafeValue(i)
	case float64:
		return core.NewSafeValue(int64(n))
	default:
		return core.NewSafeValue(data)
	}
}

func unmarshalUint64(data any) (*core.Value, error) {
	switch n := data.(type) {
	case json.Number:
		i, err := strconv.ParseUint(n.String(), 10, 64)
		if err != nil {
			return nil, err
		}
		return core.NewSafeValue(i)
	case float64:
		return core.NewSafeValue(uint64(n))
	default:
		return core.NewSafeValue(data)
	}
}

func unmarshalFloat64(data any) (*core.Value, error) {
	switch n := data.(type) {
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return nil, err
		}
		return core.NewSafeValue(f)
	case float64:
		return core.NewSafeValue(n)
	default:
		return core.NewSafeValue(data)
	}
}
