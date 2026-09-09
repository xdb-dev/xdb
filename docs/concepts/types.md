---
title: Types
description: Built-in type system with typed value accessors and the SQLite column mapping.
package: core
---

# Types

Every XDB `Value` carries type metadata. Use it to read typed values in Go and map values to SQLite columns.

## Supported Types

The user-facing type names are lowercase. Internally, XDB stores type identifiers as uppercase constants (`TID`). `TID.Lower()` returns the lowercase form for JSON output and CLI display.

| Type       | Go Type           | SQLite    | Description             |
| ---------- | ----------------- | --------- | ----------------------- |
| `string`   | `string`          | `TEXT`    | UTF-8 string            |
| `integer`  | `int64`           | `INTEGER` | 64-bit signed integer   |
| `unsigned` | `uint64`          | `INTEGER` | 64-bit unsigned integer |
| `float`    | `float64`         | `REAL`    | 64-bit floating point   |
| `boolean`  | `bool`            | `INTEGER` | True or false           |
| `time`     | `time.Time`       | `INTEGER` | Date and time in UTC    |
| `json`     | `json.RawMessage` | `TEXT`    | Arbitrary JSON data     |
| `bytes`    | `[]byte`          | `BLOB`    | Binary data             |
| `array`    | `[]*Value`        | `TEXT`    | Array of typed values   |

`core.ValueTypes` is the ordered list of user-facing types. From the [CLI](../../cmd/xdb/cli/CONTEXT.md), you declare types in schema field definitions, for example `{"fields":{"age":{"type":"integer"}}}`. Filter predicates use the declared types. Run `xdb describe --value-types` for the live list.

## Type Identifiers

A `TID` (Type ID) is a string constant that identifies a type:

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

A `Value` is a typed container. It holds the data and its type metadata.

### Creating Values

Typed constructors are preferred, because they do not use reflection:

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

The dynamic constructors use reflection:

```go
v := core.NewValue("hello")          // panics on an unsupported type
v, err := core.NewSafeValue("hello") // returns an error instead
```

### Accessing Values

Use the `As*` methods to read a value with its type. Each method returns `(T, error)`:

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

If the value does not have the requested type, the method returns `ErrTypeMismatch`.

A nil `*Value` is an attribute that is explicitly set to null. The `As*`
methods on a nil `*Value` return the zero value with no error. The same `As*`
methods on a nil [Tuple](tuples.md) return `ErrAttrNotFound`. A nil tuple
means that the attribute is absent, which is different from an explicit null.
See [Tuples](tuples.md#typed-value-accessors).

### Inspecting Values

```go
value.Type()   // Type — type metadata
value.IsNil()  // bool — true if the value is nil
```

Use `As*` methods for type-safe access. `Unwrap()` returns a raw `any` value without a type guarantee.

## Array Types

Arrays carry an element type:

```go
arrType := core.NewArrayType(core.TIDString) // ARRAY<STRING>
arrType.ID()           // TIDArray
arrType.ElemTypeID()   // TIDString
```

In [Schema](schemas.md) definitions, every array field must declare its
element type with the `elem_type` JSON property. In Go, the element type is
part of the `Type` of the field, built with `core.NewArrayType`. The element
type is required in all modes. It is immutable after the field exists. See
[Schemas -> Array fields](schemas.md#array-fields).

## SQLite Type Mapping

The SQLite driver is the only driver that maps XDB types to database column types. The mapping lives in `store/xdbsqlite/internal/sql`:

- `SQLiteTypeName` returns the column type from the table above. A `strict` or `dynamic` schema gets a column table with one column per field. A `flexible` schema and schema-free records get a key-value table. The key-value table stores each value in its native SQLite storage class, with `_type` and `_elem` columns that record the XDB type.

- The `Value` type implements `driver.Valuer` and `sql.Scanner`. It converts a `*core.Value` to a SQL parameter on write and back to a `*core.Value` on read. A `boolean` is stored as `0` or `1`. A `time` is stored as Unix milliseconds. A `json` value is stored as text. An `array` is stored as a JSON array in text form.

The other drivers (memory, filesystem, redis) have no column types. See [Drivers](drivers.md).

## Related Concepts

- [Tuples](tuples.md): Tuples carry typed values

- [Schemas](schemas.md): Field definitions reference type IDs

- [Encoding](encoding.md): Type conversion during JSON serialization

- [Stores](stores.md): The store facade that the drivers sit behind
