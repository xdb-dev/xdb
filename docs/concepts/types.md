---
title: Types
description: Built-in type system with typed value accessors and cross-database type mapping.
package: core, types
---

# Types

XDB has a built-in type system that maps to both Go types and database-specific types. Every [Value](tuples.md) in XDB carries its type, enabling type-safe access and cross-database portability.

## Supported Types

The canonical user-facing type names are lowercase. Internally, type identifiers are stored as uppercase constants (`TID`), but `TID.Lower()` returns the lowercase form for JSON output and CLI display.

| Type       | Go Type           | PostgreSQL         | SQLite    | Description             |
| ---------- | ----------------- | ------------------ | --------- | ----------------------- |
| `string`   | `string`          | `TEXT`             | `TEXT`    | UTF-8 string            |
| `integer`  | `int64`           | `BIGINT`           | `INTEGER` | 64-bit signed integer   |
| `unsigned` | `uint64`          | `BIGINT`           | `INTEGER` | 64-bit unsigned integer |
| `float`    | `float64`         | `DOUBLE PRECISION` | `REAL`    | 64-bit floating point   |
| `boolean`  | `bool`            | `BOOLEAN`          | `INTEGER` | True or false           |
| `time`     | `time.Time`       | `TIMESTAMPTZ`      | `INTEGER` | Date and time in UTC    |
| `json`     | `json.RawMessage` | `JSONB`            | `TEXT`    | Arbitrary JSON data     |
| `bytes`    | `[]byte`          | `BYTEA`            | `BLOB`    | Binary data             |
| `array`    | `[]*Value`        | `[]T`              | `TEXT`    | Array of typed values   |

The ordered list of user-facing types is available as `core.ValueTypes`.

## Type Identifiers

Types are identified by `TID` (Type ID), a string constant:

```go
core.TIDString    // "STRING"
core.TIDInteger   // "INTEGER"
core.TIDUnsigned  // "UNSIGNED"
core.TIDFloat     // "FLOAT"
core.TIDBoolean   // "BOOLEAN"
core.TIDTime      // "TIME"
core.TIDJSON      // "JSON"
core.TIDBytes     // "BYTES"
core.TIDArray     // "ARRAY"
core.TIDUnknown   // "UNKNOWN"
```

## Values

A `Value` is a typed container that holds data and its type metadata.

### Creating Values

Typed constructors (preferred — no reflection):

```go
core.StringVal("hello")
core.IntVal(42)
core.UintVal(100)
core.FloatVal(3.14)
core.BoolVal(true)
core.TimeVal(time.Now())
core.JSONVal(json.RawMessage(`{"key":"val"}`))
core.BytesVal([]byte{0x01, 0x02})
core.ArrayVal(core.StringVal("a"), core.StringVal("b"))
```

Dynamic constructor (uses reflection):

```go
v := core.NewValue("hello")          // panics on unsupported type
v, err := core.NewSafeValue("hello") // safe version
```

### Accessing Values

Use `As*` methods for type-safe extraction. Each returns `(T, error)`:

```go
s, err := value.AsStr()    // string
n, err := value.AsInt()    // int64
u, err := value.AsUint()   // uint64
f, err := value.AsFloat()  // float64
b, err := value.AsBool()   // bool
t, err := value.AsTime()   // time.Time
j, err := value.AsJSON()   // json.RawMessage
bs, err := value.AsBytes() // []byte
a, err := value.AsArray()  // []*Value
```

If the value does not match the requested type, `ErrTypeMismatch` is returned.

All accessors are **nil-safe** — calling them on a nil value returns the zero value for the type without error.

### Must Accessors

For cases where you are certain of the type, `Must*` methods panic on mismatch:

```go
s := value.MustStr()   // panics if not STRING
n := value.MustInt()   // panics if not INTEGER
```

### Inspecting Values

```go
value.Type()   // Type — type metadata
value.IsNil()  // bool — true if value is nil
```

> **Important:** Avoid `Unwrap()`. Always use `As*` methods for type-safe access. `Unwrap()` returns the raw `any` value without type guarantees.

## Array Types

Arrays carry an element type:

```go
arrType := core.NewArrayType(core.TIDString) // ARRAY<STRING>
arrType.ID()       // TIDArray
arrType.ElemType() // TIDString
```

In [Schema](schemas.md) definitions, array fields specify the element type.

## Type Codec

Each database store defines a codec that maps XDB types to database-specific representations. A codec provides two encoding paths:

- **ToDriver / FromDriver** — converts `*core.Value` to/from `driver.Value` for SQL column storage
- **ToBytes / FromBytes** — converts `*core.Value` to/from `[]byte` for KV storage

The SQLite store's codec is defined in `store/xdbsqlite/internal/sql/codec.go`.

This is how XDB defines types once and maps them to SQLite, Postgres, Redis, or the filesystem.

## Related Concepts

- [Tuples](tuples.md) — Tuples carry typed values
- [Schemas](schemas.md) — Field definitions reference type IDs
- [Encoding](encoding.md) — Type conversion during JSON serialization
- [Stores](stores.md) — Type codec used by database backends
