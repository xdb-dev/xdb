---
title: Errors
description: Sentinel errors, the [xdb/pkg] message prefix, and the error tag vocabulary that reaches callers as JSON-RPC error data.
package: core, rpc, cmd/xdb/cli/output
---

# Errors

Match an XDB error with `errors.Is` against a sentinel, and read its tags for structured detail. The message is written for a human; its exact wording is not a contract.

## Sentinels

The shared sentinels live in `core`, so that the store, the RPC layer, and the drivers can all return them without importing each other:

```go
core.ErrNotFound
core.ErrAlreadyExists
core.ErrSchemaViolation
core.ErrAttrNotFound
core.ErrConflict
core.ErrUniqueViolation
core.ErrInvalidFilter
core.ErrInvalidURI
core.ErrUnknownType
core.ErrUnsupportedValue
core.ErrTypeMismatch
core.ErrNotImplemented
```

Match with `errors.Is`:

```go
record, err := db.GetRecord(ctx, uri)
if errors.Is(err, core.ErrNotFound) {
    // ...
}
```

`ErrAttrNotFound` and `ErrConflict` are deliberately standalone and do **not** wrap `ErrNotFound`. An attribute typo must not surface as a missing resource, and a failed compare-and-swap is not an absence.

## Message prefixes

Every error message opens with `[xdb/<pkg>]`, naming the package that detected the fault:

```
[xdb/store] GetTuple requires an attr-level URI, got xdb://ns/posts/1
[xdb/api] schemas.create xdb://ns/posts: schema exists with a different definition
[xdb/core] not found
```

The prefix applies everywhere, so a message can be traced to a package without a stack trace. Match with `errors.Is` instead of on the prefix text.

## Tags

XDB attaches structured data to an error with `xerrors.Wrap`, from `github.com/gojekfarm/xtools/errors`. That package is always imported under the `xerrors` alias, and only `Wrap` and `ErrorTags` are used from it. Every other `errors` function comes from the standard library:

```go
import (
    "errors"                                     // New, Is, As, Join, Unwrap
    xerrors "github.com/gojekfarm/xtools/errors" // Wrap, ErrorTags
)

return xerrors.Wrap(schema.ErrTypeMismatch,
    "field", "age",
    "expected", "INTEGER",
    "got", "STRING",
)
```

A call site using `xerrors.` is therefore attaching structured data, and one using plain `errors.` is not. The linter pins the alias; the rest is convention.

### The vocabulary

The tags reach callers as the `data` object of a JSON-RPC error, so the key set is part of the public API. Only these keys are permitted:

| Key | Meaning |
|-----|---------|
| `field` | The attribute or struct field at fault |
| `member` | A name inside `field`, for object-valued attributes |
| `pointer` | A JSON Pointer locating the fault in the source document |
| `reason` | A short machine-readable cause, in snake_case |
| `fix` | One sentence telling the caller what to do instead |
| `type` | A type name, or a comma-separated list of them |
| `expected` | The value or type required |
| `got` | The value or type supplied |
| `from` | The prior value, for a genuine transition |
| `to` | The new value, for a genuine transition |
| `valid` | A comma-separated list of the permitted values |
| `ref` | An unresolved `$ref` |
| `keys` | A comma-separated list of offending keys in the source |
| `keyword` | The source-format keyword at fault, e.g. `oneOf` |
| `cycle` | A rendered type cycle |
| `index` | The position of the offending element |
| `mode` | A schema mode name |
| `message` | A protobuf message name |
| `number` | A protobuf field number |
| `oneof` | A protobuf oneof name |
| `parts` | The components of a malformed URI |
| `conflict` | A colliding declaration |

Use `expected`/`got` for a mismatch and `from`/`to` only for a transition such as schema evolution. Adding a key means adding it to this table and to the `Error tags` section of the `core` package doc first.

### Reading tags in Go

```go
if tags := core.ErrorTags(err); tags != nil {
    fmt.Println(tags["field"], tags["expected"], tags["got"])
}
```

### Constraints on `xerrors.Wrap`

`Wrap` panics on an odd number of attrs, so the pairs must balance. It also edits an existing `*xerrors.ErrorTags` found in the chain **in place** rather than wrapping it. Tag an error once, at the point the fault is detected, and never re-wrap one that already carries tags.

## On the wire

`rpc.MapError` maps a sentinel to a JSON-RPC code and puts the tags in `error.data`:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "error": {
    "code": -32002,
    "message": "[xdb/core] schema violation: field \"age\": cannot decode as INTEGER [field=age, expected=INTEGER, got=STRING, reason=decode_failed]",
    "data": {
      "field": "age",
      "expected": "INTEGER",
      "got": "STRING",
      "reason": "decode_failed"
    }
  }
}
```

| Code | Sentinel |
|------|----------|
| `-32000` | `core.ErrNotFound` |
| `-32001` | `core.ErrAlreadyExists` |
| `-32002` | `core.ErrSchemaViolation`, `core.ErrUnknownType`, `schema.ErrInvalidMode` |
| `-32003` | `core.ErrConflict` |
| `-32004` | `core.ErrNotImplemented` |
| `-32005` | `core.ErrUniqueViolation` |
| `-32602` | `core.ErrInvalidURI`, `core.ErrInvalidFilter` |

The message may still repeat the tags as text, and **their order there is not stable between runs**. Read `data` for anything a program depends on.

## In the CLI

The CLI renders every error, in every output format, as one envelope. The `fix` tag becomes `hint`; the rest become `details`:

```json
{
  "code": "SCHEMA_VIOLATION",
  "message": "field \"age\": cannot decode as INTEGER",
  "resource": "records",
  "action": "put",
  "uri": "xdb://ns/posts/1",
  "hint": "send age as an integer",
  "details": {
    "field": "age",
    "expected": "INTEGER",
    "got": "STRING",
    "reason": "decode_failed"
  }
}
```

Exit codes are derived from `code`: `1` for an application error, `2` for invalid arguments, `3` for a connection failure, `4` for an internal error.

## Related Concepts

- [Schemas](schemas.md): validation modes and what a violation means
- [Versioning](versioning.md): where `ErrConflict` comes from
- [Stores](stores.md): which verb returns which sentinel
