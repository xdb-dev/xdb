---
title: Encoding
description: JSON encoding and decoding of records with automatic dot-notation nesting.
package: encoding/xdbjson
---

# Encoding

The `encoding/xdbjson` package handles bidirectional conversion between JSON and XDB [Records](records.md). It translates between flat [Tuple](tuples.md) attributes (with dot notation) and nested JSON objects.

## Overview

```
Record (flat tuples)              JSON (nested objects)
┌─────────────────────┐           ┌─────────────────────┐
│ _id       = "123"   │           │ {                   │
│ title     = "Hello" │  encode   │   "_id": "123",     │
│ author.id = "u-1"   │ ───────→  │   "title": "Hello", │
│ author.name = "Bob" │           │   "author": {       │
└─────────────────────┘  decode   │     "id": "u-1",    │
                        ←───────  │     "name": "Bob"   │
                                  │   }                 │
                                  │ }                   │
                                  └─────────────────────┘
```

Dot-separated attributes are **unfolded** into nested objects during encoding, and **flattened** back during decoding.

## Encoder

The encoder converts records to JSON.

```go
encoder := xdbjson.New()

// Compact JSON
data, err := encoder.FromRecord(record)

// Pretty-printed JSON
data, err := encoder.FromRecord(record, xdbjson.WithIndent("", "  "))

// Field projection
data, err := encoder.FromRecord(record, xdbjson.WithFields("name", "email"))
```

### Options

```go
encoder := xdbjson.New(
    xdbjson.WithIDField("_id"),          // JSON field for record ID (default: "_id")
    xdbjson.WithNSField("_ns"),          // JSON field for namespace (default: "_ns")
    xdbjson.WithSchemaField("_schema"),  // JSON field for schema (default: "_schema")
    xdbjson.WithIncludeNS(),             // Include namespace in output
    xdbjson.WithIncludeSchema(),         // Include schema in output
)
```

### Output

```json
{
  "_id": "post-123",
  "title": "Hello World",
  "author": {
    "id": "user-001",
    "name": "Alice"
  },
  "tags": ["go", "xdb"],
  "created": "2024-01-15T10:30:00Z"
}
```

### Type Conversions (Encode)

| XDB Type   | JSON Representation     |
| ---------- | ----------------------- |
| `string`   | String                  |
| `integer`  | Number                  |
| `unsigned` | Number                  |
| `float`    | Number                  |
| `boolean`  | Boolean                 |
| `time`     | String (RFC 3339)       |
| `json`     | Inline JSON             |
| `bytes`    | String (base64-encoded) |
| `array`    | Array                   |

## Decoder

The decoder parses JSON into records.

```go
// With default NS and Schema (used when not present in JSON)
decoder := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("posts"))

// Parse JSON to a new record
record, err := decoder.ToRecord(jsonData)

// Parse JSON into an existing record
err := decoder.ToExistingRecord(jsonData, record)
```

### Schema-aware Decoding

When a schema definition is provided via `WithDef()`, the decoder coerces JSON values to match field types. For example, JSON numbers (always `float64` in Go) are converted to `int64` for `integer` fields and `uint64` for `unsigned` fields. Without a schema, values keep their JSON-native types.

```go
decoder := xdbjson.NewDecoder(
    xdbjson.WithNS("com.example"),
    xdbjson.WithSchema("posts"),
    xdbjson.WithDef(schemaDef),  // enables type coercion
)
```

### Resolution Order

The decoder resolves record identity (ID, NS, Schema) using:

1. **JSON fields** — Values found in the JSON data (e.g., `_id`, `_ns`, `_schema`).
2. **Options defaults** — Values provided in the decoder options.

If neither source provides a required field, an error is returned.

### Errors

| Error                 | Cause                                   |
| --------------------- | --------------------------------------- |
| `ErrInvalidJSON`      | JSON parsing failed                     |
| `ErrMissingID`        | No ID field in JSON and no default      |
| `ErrEmptyID`          | ID field is present but empty           |
| `ErrMissingNamespace` | No NS in JSON and no default            |
| `ErrMissingSchema`    | No schema in JSON and no default        |
| `ErrNilRecord`        | Nil record passed to `ToExistingRecord` |

## Related Concepts

- [Records](records.md) — The data being encoded
- [Tuples](tuples.md) — Dot-notation attributes
- [Types](types.md) — Type conversions during encoding
- [Stores](stores.md) — Storage backends use encoding for persistence
