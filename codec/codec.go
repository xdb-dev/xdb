package codec

import (
	"errors"

	"github.com/xdb-dev/xdb/core"
)

// KeyValueCodec is an interface for encoding and decoding key-value pairs.
type KeyValueCodec interface {
	MarshalKey(key *core.Key) ([]byte, error)
	UnmarshalKey(data []byte) (*core.Key, error)
	MarshalValue(value *core.Value) ([]byte, error)
	UnmarshalValue(data []byte) (*core.Value, error)
}

var (
	// ErrDecodingValue is returned when the value cannot be decoded.
	ErrDecodingValue = errors.New("encoding/xdbkv: decoding value failed")
	// ErrDecodingKey is returned when the key cannot be decoded.
	ErrDecodingKey = errors.New("encoding/xdbkv: decoding key failed")
)
