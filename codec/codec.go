package codec

import (
	"errors"

	"github.com/xdb-dev/xdb/types"
)

// KeyValueCodec is an interface for encoding and decoding key-value pairs.
type KeyValueCodec interface {
	MarshalKey(key *types.Key) ([]byte, error)
	UnmarshalKey(data []byte) (*types.Key, error)
	MarshalValue(value *types.Value) ([]byte, error)
	UnmarshalValue(data []byte) (*types.Value, error)
}

var (
	// ErrDecodingValue is returned when the value cannot be decoded.
	ErrDecodingValue = errors.New("encoding/xdbkv: decoding value failed")
	// ErrDecodingKey is returned when the key cannot be decoded.
	ErrDecodingKey = errors.New("encoding/xdbkv: decoding key failed")
)
