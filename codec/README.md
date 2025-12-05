# XDB Codecs

Codecs provide low-level serialization for key-value storage backends.

## Structure

Each codec lives in its own subdirectory:

```
codec/
├── codec.go        # Core interfaces
├── json/          # JSON codec
└── msgpack/       # MessagePack codec
```

## Implementation Guide

### Encoding

Codecs must marshal XDB types to byte arrays:

- **URIs**: Encode namespace, schema, ID, and attribute components
- **Values**: Encode typed values with type information
- **Tuples**: Encode as key-value pairs (key = URI, value = Value)

### Decoding

Codecs must unmarshal byte arrays back to XDB types:

- Preserve type information
- Handle missing or corrupted data gracefully
- Return descriptive errors

## Available Codecs

- **json**: Human-readable JSON encoding
- **msgpack**: Efficient binary encoding using MessagePack
