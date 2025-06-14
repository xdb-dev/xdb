// Package xdbkv provides functions for type-aware encoding and decoding
// of tuples to key-value pairs and vice versa.
package xdbkv

import (
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/xdb-dev/xdb/types"
)

var (
	ErrTypeMismatch = errors.New("encoding/xdbkv: type mismatch")
)

// EncodeTuple encodes a tuple to a key-value pair.
func EncodeTuple(tuple *types.Tuple) ([]byte, []byte, error) {
	flatkey := EncodeKey(tuple.Key())

	flatvalue, err := EncodeValue(tuple.Value())
	if err != nil {
		return nil, nil, err
	}

	return flatkey, flatvalue, nil
}

// DecodeTuple decodes a key-value pair to a tuple.
func DecodeTuple(flatkey, flatvalue []byte) (*types.Tuple, error) {
	key, err := DecodeKey(flatkey)
	if err != nil {
		return nil, err
	}

	v, err := DecodeValue(flatvalue)
	if err != nil {
		return nil, err
	}

	return types.NewTuple(key.Kind(), key.ID(), key.Attr(), v), nil
}

// EncodeKey encodes a types.Key to []byte.
func EncodeKey(key interface {
	Kind() string
	ID() string
	Attr() string
}) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", key.Kind(), key.ID(), key.Attr()))
}

// DecodeKey decodes a []byte to a types.Key.
func DecodeKey(key []byte) (*types.Key, error) {
	parts := strings.Split(string(key), ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid key: %s", string(key))
	}

	return types.NewKey(parts...), nil
}

// EncodeValue encodes a types.Value to []byte.
func EncodeValue(v types.Value) ([]byte, error) {
	switch v := v.(type) {
	case types.Bool:
		return msgpack.Marshal(&boolExt{v})
	case types.Int64:
		return msgpack.Marshal(&int64Ext{v})
	case types.Uint64:
		return msgpack.Marshal(&uint64Ext{v})
	case types.Float64:
		return msgpack.Marshal(&float64Ext{v})
	case types.String:
		return msgpack.Marshal(&stringExt{v})
	case types.Bytes:
		return msgpack.Marshal(&bytesExt{v})
	case types.Time:
		return msgpack.Marshal(&timeExt{v})
	case *types.Array:
		return msgpack.Marshal(&arrayExt{v})
	default:
		return msgpack.Marshal(v)
	}
}

// DecodeValue decodes a []byte to a types.Value.
func DecodeValue(flatvalue []byte) (types.Value, error) {
	var v any
	err := msgpack.Unmarshal(flatvalue, &v)
	if err != nil {
		return nil, err
	}

	switch vv := v.(type) {
	case *boolExt:
		return types.Bool(vv.Bool), nil
	case *int64Ext:
		return types.Int64(vv.Int64), nil
	case *uint64Ext:
		return types.Uint64(vv.Uint64), nil
	case *float64Ext:
		return types.Float64(vv.Float64), nil
	case *stringExt:
		return types.String(vv.String), nil
	case *bytesExt:
		return types.Bytes(vv.Bytes), nil
	case *timeExt:
		return types.Time(vv.Time), nil
	case *arrayExt:
		return vv.Array, nil
	default:
		return nil, fmt.Errorf("unsupported value type: %T", v)
	}
}
