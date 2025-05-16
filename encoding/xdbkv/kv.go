// Package xdbkv provides functions for type-aware encoding and decoding
// of tuples to key-value pairs and vice versa.
package xdbkv

import (
	"fmt"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/xdb-dev/xdb/types"
)

type value struct {
	TypeID   types.TypeID `msgpack:"t"`
	Value    any          `msgpack:"v"`
	Repeated bool         `msgpack:"r"`
}

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

	var v value
	err := msgpack.Unmarshal(flatvalue, &v)
	if err != nil {
		return nil, err
	}

	switch v.Value.(type) {
	case []any:
		switch v.TypeID {
		case types.TypeString:
			v.Value = castArray[string](v.Value.([]any))
		case types.TypeInteger:
			v.Value = castArray[int64](v.Value.([]any))
		case types.TypeFloat:
			v.Value = castArray[float64](v.Value.([]any))
		case types.TypeBoolean:
			v.Value = castArray[bool](v.Value.([]any))
		}
	}

	return types.NewValue(v.Value), nil
}

func castArray[T any](v []any) []T {
	arr := make([]T, len(v))
	for i, v := range v {
		arr[i] = v.(T)
	}
	return arr
}
