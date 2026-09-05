---
title: Encoding
description: JSON encoding and decoding of records with automatic dot-notation nesting.
package: encoding/xdbjson
---

# Encoding

The `encoding/xdbjson` package converts JSON to XDB [Records](records.md) and back. It translates between flat [Tuple](tuples.md) attributes (with dot notation) and nested JSON objects.

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

The encoder **unfolds** dot-separated attributes into nested objects. The decoder **flattens** nested objects back into dot-separated attributes.

## Encoder

The encoder converts records to JSON.

```go
encoder := xdbjson.New()

// Compact JSON
data, err := encoder.FromRecord(record)

// Indented JSON
data, err := encoder.FromRecord(record, xdbjson.WithIndent("", "  "))

// Field projection
data, err := encoder.FromRecord(record, xdbjson.WithFields("name", "email"))
```

`WithFields` limits the output to the named fields. The ID field is always included.

### Options

```go
encoder := xdbjson.New(
    xdbjson.WithIDField("_id"),          // JSON field for the record ID (default: "_id")
    xdbjson.WithNSField("_ns"),          // JSON field for the namespace (default: "_ns")
    xdbjson.WithSchemaField("_schema"),  // JSON field for the schema (default: "_schema")
    xdbjson.WithIncludeNS(),             // Include the namespace in the output
    xdbjson.WithIncludeSchema(),         // Include the schema in the output
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
// With a default NS and Schema (used when the JSON does not carry them)
decoder := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("posts"))

// Parse JSON into a new record
record, err := decoder.ToRecord(jsonData)

// Parse JSON into an existing record
err := decoder.ToExistingRecord(jsonData, record)
```

### Numbers

The decoder reads JSON numbers with `UseNumber`, so no precision is lost through `float64`. Without a schema, a number with a fractional part or an exponent becomes a `float`. Every other number becomes an `integer`. A number that does not fit in an `int64` becomes a `float`.

### Schema-aware Decoding

When you give the decoder a schema definition with `WithDef()`, the decoder converts each declared field to its declared type:

- `integer`, `unsigned`, and `float` fields are converted from JSON numbers.
- `time` fields are parsed from RFC 3339 strings.
- `bytes` fields are decoded from base64 strings.
- `json` fields keep their JSON value as-is. Their nested keys are not flattened.
- `array` fields convert every element to the declared `elem_type`.
- Object arrays (`ARRAY<JSON>` with `items`) convert each element member to the type that `items` declares.

If a declared field cannot be decoded as its declared type, the decoder returns an error that wraps `core.ErrSchemaViolation`. The error names the field and the expected type. Undeclared attributes get their types from the number rules above. An undeclared attribute that XDB cannot type, for example an empty or mixed JSON array, is dropped in the same way as a null.

`WithNumberInference()` is deprecated and has no effect. The decoder always infers numbers as described above.

```go
decoder := xdbjson.NewDecoder(
    xdbjson.WithNS("com.example"),
    xdbjson.WithSchema("posts"),
    xdbjson.WithDef(schemaDef),  // enables type conversion
)
```

### Resolution Order

The decoder resolves the record identity (ID, NS, Schema) in this order:

1. **JSON fields** — the values in the JSON data (`_id`, `_ns`, `_schema`).
2. **Option defaults** — the values from `WithNS` and `WithSchema`.

If neither source gives a required value, the decoder returns an error.

### Errors

| Error                     | Cause                                                        |
| ------------------------- | ------------------------------------------------------------ |
| `ErrInvalidJSON`          | The JSON did not parse                                       |
| `ErrMissingID`            | No ID field in the JSON                                      |
| `ErrEmptyID`              | The ID field is present but empty                            |
| `ErrMissingNamespace`     | No namespace in the JSON and no default                      |
| `ErrMissingSchema`        | No schema in the JSON and no default                         |
| `ErrNilRecord`            | A nil record was passed to `FromRecord` or `ToExistingRecord` |
| `core.ErrSchemaViolation` | A declared field cannot be decoded as its declared type      |

## Related Concepts

- [Records](records.md) — The data that is encoded
- [Tuples](tuples.md) — Dot-notation attributes
- [Types](types.md) — Type conversions during encoding
- [Stores](stores.md) — The drivers use encoding for persistence
