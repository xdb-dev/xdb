package codec

import (
	"errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrDecodingValue is returned when the value cannot be decoded.
	ErrDecodingValue = errors.New("codec: decoding value failed")
	// ErrDecodingKey is returned when the key cannot be decoded.
	ErrDecodingKey = errors.New("codec: decoding key failed")
)

// KVEncoder is an interface for encoding XDB types to key-value pairs.
type KVEncoder interface {
	EncodeID(id core.ID) ([]byte, error)
	EncodeAttr(attr core.Attr) ([]byte, error)
	EncodeKey(key *core.Key) ([]byte, []byte, error)
	EncodeValue(value *core.Value) ([]byte, error)
}

// KVDecoder is an interface for decoding key-value pairs to XDB types.
type KVDecoder interface {
	DecodeID(data []byte) (core.ID, error)
	DecodeAttr(data []byte) (core.Attr, error)
	DecodeKey(id []byte, attr []byte) (*core.Key, error)
	DecodeValue(data []byte) (*core.Value, error)
}

// KVCodec is an interface for encoding and decoding XDB types to key-value pairs.
type KVCodec interface {
	KVEncoder
	KVDecoder
}
