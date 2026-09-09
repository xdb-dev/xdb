---
title: Encoding
description: JSON Schema import and JSON encoding of records with automatic dot-notation nesting.
package: encoding/xdbjson
---

# Encoding

The `encoding/xdbjson` package converts JSON documents to XDB [records](records.md) and imports JSON Schema definitions. Both operations use the same option set.

Pass an imported definition to the data decoder with `WithDef` to decode declared fields using their schema types.

## Overview

```
Record (flat tuples)              JSON (nested objects)
┌─────────────────────┐           ┌─────────────────────┐
│ _id       = "123"   │           │ {                   │
│ title     = "Hello" │ Unmarshal │   "_id": "123",     │
│ author.id = "u-1"   │ ───────→  │   "title": "Hello", │
│ author.name = "Bob" │           │   "author": {       │
└─────────────────────┘  Marshal  │     "id": "u-1",    │
                        ←───────  │     "name": "Bob"   │
                                  │   }                 │
                                  │ }                   │
                                  └─────────────────────┘
```

`Unmarshal` unfolds dot-separated attributes into nested objects. `Marshal` flattens nested objects back into dot-separated attributes.

> Direction convention. In XDB, `Marshal` produces a record and `Unmarshal` consumes one. This is the inverse of `encoding/json`, and it holds for every format adapter: `xdbjson`, `xdbproto`, and `xdbstruct` all marshal *into* the XDB data model and unmarshal *out of* it.

## Encoding Records to JSON

```go
// Compact JSON
data, err := xdbjson.Unmarshal(record)

// Indented JSON
data, err := xdbjson.Unmarshal(record, xdbjson.WithIndent("", "  "))

// Field projection
data, err := xdbjson.Unmarshal(record, xdbjson.WithFields("name", "email"))
```

`WithFields` limits the output to the named fields. The ID field is always included.

### Options

```go
data, err := xdbjson.Unmarshal(record,
    xdbjson.WithIDField("_id"),          // JSON field for the record ID (default: "_id")
    xdbjson.WithNSField("_ns"),          // JSON field for the namespace (default: "_ns")
    xdbjson.WithSchemaField("_schema"),  // JSON field for the schema (default: "_schema")
    xdbjson.WithIncludeNS(),             // Include the namespace in the output
    xdbjson.WithIncludeSchema(),         // Include the schema in the output
)
```

Of the identity fields, only the ID is emitted by default. Use `WithIncludeNS` and `WithIncludeSchema` to include the namespace and schema.

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

## Decoding JSON to Records

`Marshal` builds a new record. The defaults apply when the document carries no namespace or schema field:

```go
record, err := xdbjson.Marshal(jsonData,
    xdbjson.WithNS("com.example"),
    xdbjson.WithSchema("posts"),
)
```

`MarshalInto` populates a record the caller already owns:

```go
record := core.NewRecord("com.example", "posts", "p1")

err := xdbjson.MarshalInto(jsonData, record)
```

`Marshal` takes no URI, unlike `xdbproto.Marshal` and `xdbstruct.Marshal`, because a JSON document carries its own identity. `MarshalInto` keeps the identity of the record you pass and ignores any metadata fields in the document, so it is the one to use when the caller owns the URI.

### Numbers

The decoder reads JSON numbers with `UseNumber`, so no precision is lost through `float64`. Without a schema, a number with a fractional part or an exponent becomes a `float`. Every other number becomes an `integer`. A number that does not fit in an `int64` becomes a `float`.

### Schema-Aware Decoding

When you pass a schema definition with `WithDef()`, each declared field is converted to its declared type:

- `integer`, `unsigned`, and `float` fields are converted from JSON numbers.

- `time` fields are parsed from RFC 3339 strings.

- `bytes` fields are decoded from base64 strings.

- `json` fields keep their JSON value as-is. Their nested keys are not flattened.

- `array` fields convert every element to the declared `elem_type`.

- Object arrays (`ARRAY<JSON>` with `items`) convert each element member to the type that `items` declares.

If a declared field cannot be decoded as its declared type, the decoder returns an error that wraps `core.ErrSchemaViolation`. The error names the field and the expected type. Undeclared attributes get their types from the number rules above. An undeclared attribute that XDB cannot type, for example an empty or mixed JSON array, is dropped in the same way as a null.

```go
record, err := xdbjson.Marshal(jsonData,
    xdbjson.WithNS("com.example"),
    xdbjson.WithSchema("posts"),
    xdbjson.WithDef(schemaDef),  // enables type conversion
)
```

### Resolution Order

`Marshal` resolves the record identity (ID, NS, Schema) in this order:

1. JSON fields: the values in the JSON data (`_id`, `_ns`, `_schema`).

2. Option defaults: the values from `WithNS` and `WithSchema`.

If neither source gives a required value, `Marshal` returns an error.

## Importing a JSON Schema

`ImportSchema` parses a JSON Schema document (a documented subset of draft 2020-12) into a `schema.Def`:

```go
def, err := xdbjson.ImportSchema(doc, xdbjson.WithNS("com.example"))
```

The namespace comes from `WithNS` and is required: the importer never derives one from the document. The schema name comes from `WithSchema`, the document `title`, or the `$id` filename, in that order. `WithOpaqueJSON` imports a named `$ref` pointer as an opaque JSON field, which allows a cyclic `$ref` to be stored as JSON.

See [Bring Your Own Types](bring-your-own-types.md) for the full type mapping, the two nesting representations, and the list of rejected constructs.

## Errors

Record encoding and decoding return these errors:

| Error                     | Cause                                                   |
| ------------------------- | ------------------------------------------------------- |
| `ErrInvalidJSON`          | The document did not parse                              |
| `ErrMissingID`            | No ID field in the document                             |
| `ErrEmptyID`              | The ID field is present but empty                       |
| `ErrMissingNamespace`     | No namespace in the document and no `WithNS`            |
| `ErrMissingSchema`        | No schema in the document and no `WithSchema`           |
| `ErrNilRecord`            | A nil record was passed to `Unmarshal` or `MarshalInto` |
| `core.ErrSchemaViolation` | A declared field cannot be decoded as its declared type |

`ImportSchema` shares `ErrInvalidJSON`, returns `ErrMissingNamespace` when `WithNS` was not given, and returns `ErrMissingSchema` when no name resolves from `WithSchema`, `title`, or `$id`. Every unsupported construct has its own error that names the JSON pointer to the offending node: `ErrInvalidKey`, `ErrUnion`, `ErrConflict`, `ErrCrossDocument`, `ErrCyclicRef`, `ErrUnresolvedRef`, and `ErrUnsupported`.

## Related Concepts

- [Records](records.md): The data that is encoded

- [Tuples](tuples.md): Dot-notation attributes

- [Types](types.md): Type conversions during encoding

- [Bring Your Own Types](bring-your-own-types.md): The schema import path, and the sibling proto and Go-struct adapters

- [Stores](stores.md): The drivers use encoding for persistence
