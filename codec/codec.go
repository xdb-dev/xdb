package codec

import (
	"errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrDecodingValue is returned when the value cannot be decoded.
	ErrDecodingValue = errors.New("codec: decoding value failed")
	// ErrDecodingURI is returned when the URI cannot be decoded.
	ErrDecodingURI = errors.New("codec: decoding URI failed")
)

// KVEncoder is an interface for encoding XDB types for key-value storage.
type KVEncoder interface {
	EncodeURI(uri *core.URI) ([]byte, error)
	EncodeValue(value *core.Value) ([]byte, error)
	EncodeTuple(tuple *core.Tuple) ([]byte, []byte, error)
}

// KVDecoder is an interface for decoding key-value pairs from key-value storage.
type KVDecoder interface {
	DecodeURI(data []byte) (*core.URI, error)
	DecodeValue(data []byte) (*core.Value, error)
	DecodeTuple(key []byte, value []byte) (*core.Tuple, error)
}

// KVCodec is an interface for encoding and decoding XDB types to key-value pairs.
type KVCodec interface {
	KVEncoder
	KVDecoder
}
