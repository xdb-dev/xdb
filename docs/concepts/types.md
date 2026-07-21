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

The ordered list of user-facing types is available as `core.ValueTypes`. From the [CLI](../../cmd/xdb/cli/CONTEXT.md), types are declared in schema field definitions (`{"fields":{"age":{"type":"integer"}}}`) and interpreted by filter predicates; run `xdb describe --value-types` for the live list.

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
core.ArrayVal(core.TIDString, core.StringVal("a"), core.StringVal("b"))
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

A nil `*Value` (an attribute explicitly set to null) returns the zero value
with no error. The same `As*` methods on a [Tuple](tuples.md) instead return
`ErrAttrNotFound` when the tuple itself is nil — i.e. the attribute is absent,
which is distinct from an explicit null. See [Tuples](tuples.md#typed-value-accessors).

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
arrType.ID()           // TIDArray
arrType.ElemTypeID()   // TIDString
```

In [Schema](schemas.md) definitions, every array field must declare its element
type via the `elem_type` JSON property (in Go, it is folded into the field's
`Type` with `core.NewArrayType`). Element type is required in all modes and is
immutable once set — see [Schemas → Array fields](schemas.md#array-fields).

## Type Codec

Each database store defines a codec that maps XDB types to database-specific representations. The SQLite codec is a `Value` type providing two encoding paths:

- **`Value()` / `Scan()`** — implements `driver.Valuer` and `sql.Scanner`, converting `*core.Value` to/from a `driver.Value` for SQL column storage
- **`MarshalBytes()` / `UnmarshalBytes()`** — converts `*core.Value` to/from `[]byte` for KV storage

The SQLite store's codec is defined in `store/xdbsqlite/internal/sql/value.go`.

This is how XDB defines types once and maps them to SQLite, Postgres, Redis, or the filesystem.

## Related Concepts

- [Tuples](tuples.md) — Tuples carry typed values
- [Schemas](schemas.md) — Field definitions reference type IDs
- [Encoding](encoding.md) — Type conversion during JSON serialization
- [Stores](stores.md) — Type codec used by database backends
