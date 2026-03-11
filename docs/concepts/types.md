---
title: Types
description: Built-in type system with typed value accessors and cross-database type mapping.
package: core, types
---

# Types

XDB has a built-in type system that maps to both Go types and database-specific types. Every [Value](tuples.md) in XDB carries its type, enabling type-safe access and cross-database portability.

## Supported Types

| Type ID     | Go Type             | PostgreSQL         | SQLite    | Description                  |
| ----------- | ------------------- | ------------------ | --------- | ---------------------------- |
| `STRING`    | `string`            | `TEXT`             | `TEXT`    | UTF-8 string                 |
| `INTEGER`   | `int64`             | `BIGINT`           | `INTEGER` | 64-bit signed integer        |
| `UNSIGNED`  | `uint64`            | `BIGINT`           | `INTEGER` | 64-bit unsigned integer      |
| `FLOAT`     | `float64`           | `DOUBLE PRECISION` | `REAL`    | 64-bit floating point        |
| `BOOLEAN`   | `bool`              | `BOOLEAN`          | `INTEGER` | True or false                |
| `TIME`      | `time.Time`         | `TIMESTAMPTZ`      | `INTEGER` | Date and time in UTC         |
| `JSON`      | `json.RawMessage`   | `JSONB`            | `TEXT`    | Arbitrary JSON data          |
| `BYTES`     | `[]byte`            | `BYTEA`            | `BLOB`    | Binary data                  |
| `ARRAY`     | `[]*Value`          | `[]T`              | `TEXT`    | Array of typed values        |

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

The `types` package provides a codec system for mapping XDB types to database-specific types. Each database backend registers its own mappings:

```go
codec := types.New("sqlite")
codec.Register(types.Mapping{
    Type:     core.TypeString,
    TypeName: "TEXT",
    Encode:   func(v *core.Value) (driver.Value, error) { ... },
    Decode:   func(t core.Type, src any) (*core.Value, error) { ... },
})

// Convert between XDB and database values
dbVal, err := codec.Encode(xdbValue)
xdbVal, err := codec.Decode(core.TypeString, dbVal)

// Get database type name
name, err := codec.TypeName(core.TypeString) // "TEXT"
```

This abstraction allows XDB to work with multiple databases without changing application code.

## Related Concepts

- [Tuples](tuples.md) — Tuples carry typed values
- [Schemas](schemas.md) — Field definitions reference type IDs
- [Encoding](encoding.md) — Type conversion during JSON serialization
- [Stores](stores.md) — Type codec used by database backends
