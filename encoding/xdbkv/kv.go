// Package xdbkv provides functions for type-aware encoding and decoding
// of tuples to key-value pairs and vice versa.
package xdbkv

import (
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
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

type value struct {
	TypeID   types.TypeID `msgpack:"t"`
	Repeated bool         `msgpack:"r"`
	Value    any          `msgpack:"v"`
}

// EncodeValue encodes a types.Value to []byte.
func EncodeValue(v *types.Value) ([]byte, error) {
	vv := value{
		TypeID:   v.TypeID(),
		Value:    v.Unwrap(),
		Repeated: v.Repeated(),
	}

	return msgpack.Marshal(vv)
}

// DecodeValue decodes a []byte to a types.Value.
func DecodeValue(flatvalue []byte) (*types.Value, error) {
	var vv value
	err := msgpack.Unmarshal(flatvalue, &vv)
	if err != nil {
		return nil, err
	}

	switch vv.TypeID {
	case types.TypeString:
		arr, ok := vv.Value.([]any)
		if ok {
			vv.Value = x.CastArray(arr, cast.ToString)
		} else {
			vv.Value = cast.ToString(vv.Value)
		}
	case types.TypeInteger:
		arr, ok := vv.Value.([]any)
		if ok {
			vv.Value = x.CastArray(arr, cast.ToInt64)
		} else {
			vv.Value = cast.ToInt64(vv.Value)
		}
	case types.TypeFloat:
		arr, ok := vv.Value.([]any)
		if ok {
			vv.Value = x.CastArray(arr, cast.ToFloat64)
		} else {
			vv.Value = cast.ToFloat64(vv.Value)
		}
	case types.TypeBoolean:
		arr, ok := vv.Value.([]any)
		if ok {
			vv.Value = x.CastArray(arr, cast.ToBool)
		} else {
			vv.Value = cast.ToBool(vv.Value)
		}
	case types.TypeBytes:
		arr, ok := vv.Value.([]any)
		if ok {
			vv.Value = x.CastArray(arr, x.ToBytes)
		} else {
			vv.Value = x.ToBytes(vv.Value)
		}
	}

	return types.NewValue(vv.Value), nil
}
