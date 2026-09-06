---
title: Bring Your Own Types
description: Import protobuf, JSON Schema, and Go struct types into a schema.Def. The mapping tables, null handling, and documented rejections per source.
package: encoding/xdbjson, encoding/xdbproto, encoding/xdbstruct
---

# Bring Your Own Types

You already have types: protobuf messages, JSON Schemas, Go structs. XDB imports these types into a [schema.Def](schemas.md). You can then store and query the data without a hand-written schema, and without a second source of truth.

Two rules anchor the design:

1. **The IR is one-way.** Proto files, JSON Schemas, and Go types stay the source of truth. `Def` is a projection that an importer produces. XDB never regenerates source from a stored schema.
2. **The contract is data round-trip, not schema round-trip.** Import a `User`, write a `User`, and read the same `User` back. Filter or list it in between. Source-type details that do not affect the data round-trip (`int32` against `int64`, enum names, proto field numbers) are preserved in opaque per-field annotations, not modeled.

There is one adapter per source format, under `encoding/`. Each owns both halves: the schema path that builds a `Def`, and the data path that moves values in and out of a record.

| Source      | Package              | Schema path                    | Data path              | CLI                                   |
| ----------- | -------------------- | ------------------------------ | ---------------------- | ------------------------------------- |
| Go structs  | `encoding/xdbstruct` | `Def[T]`                       | `Marshal`, `Unmarshal` | in-process only (Go embed)            |
| Protobuf    | `encoding/xdbproto`  | `ImportFiles`, `ImportMessage` | `Marshal`, `Unmarshal` | `xdb schemas import user.proto`       |
| JSON Schema | `encoding/xdbjson`   | `ImportSchema`                 | `Marshal`, `Unmarshal` | `xdb schemas import user.schema.json` |

The CLI imports proto and JSON Schema **files**, one file per command. The Go-struct path is in-process: you embed XDB and call `xdbstruct`. It is not a CLI file import.

## Two nesting representations

Every importer shares one model for nested data. The model decides what you can filter:

- A **single nested object** (`*Profile`, a proto sub-message, a nested JSON Schema `object`) **flattens to dotted attributes**: `profile.name`, `profile.city`. These are first-class attributes. Filters can address them.
- An **array of objects** (`[]Order`, `repeated Order`, `items: {type: object}`) uses `Field.Items`. Each element validates against the element field set, but the elements stay opaque. Filters cannot reach inside array elements.

`Items` is for arrays only. A single object never uses it.

## Protobuf → schema.Def

`encoding/xdbproto` walks `protoreflect` descriptors. No code generation is needed. The proto package becomes the namespace, unless `WithNamespace` (`--ns`) overrides it.

| Proto construct                                   | schema.Def field type            | Notes                                                        |
| ------------------------------------------------- | -------------------------------- | ------------------------------------------------------------ |
| `bool`                                            | `boolean`                        |                                                              |
| `int32`/`sint32`/`sfixed32`/`int64`/…             | `integer`                        | width in `Annotations["proto.type"]`                         |
| `uint32`/`fixed32`/`uint64`/`fixed64`             | `unsigned`                       |                                                              |
| `float`/`double`                                  | `float`                          |                                                              |
| `string`                                          | `string`                         |                                                              |
| `bytes`                                           | `bytes`                          |                                                              |
| `enum`                                            | `string` (value name)            | `Annotations["proto.enum"]`                                  |
| `google.protobuf.Timestamp`                       | `time`                           |                                                              |
| wrapper (`Int32Value`, `StringValue`, …)          | the wrapped scalar               | adds presence                                                |
| `Duration`/`Struct`/`Value`/`ListValue`/`Any`     | `json`                           | `Annotations["proto.type"]`                                  |
| `map<K, V>`                                       | `json`                           | always JSON, no opt-in. `Annotations["proto.map"]` records the key and value types |
| nested message (single)                           | dotted attributes                | `address` → `address.city`, `address.zip`                    |
| `repeated <scalar/enum>`                          | `array<scalar>`                  |                                                              |
| `repeated <message>`                              | `array<json>` + `Items`          | object array                                                 |

Every field records `Annotations["proto.number"]`. The field number is what makes a proto rename reliably detectable (see [Renames](#renames-are-proto-only)).

**Documented rejections** (each error names the offending item and the fix):

- a file with no proto package and no `WithNamespace` → `ErrNoNamespace`.
- `oneof` → `ErrOneof`. Unions are a non-goal. Flatten the union into optional fields.
- a recursive message (`Node` that refers to `Node`) → `ErrRecursive`. Pass `WithAllowJSON("pkg.Node")` (`--allow-json pkg.Node`) to store it as JSON.

## JSON Schema → schema.Def

`encoding/xdbjson` targets a documented subset of draft 2020-12. The root must describe an object. The namespace comes from `WithNS` (`--ns`). The schema name comes from `WithSchema`, the document `title`, or the `$id` filename, in that order.

| JSON Schema construct                    | schema.Def result                        | Notes                                             |
| ---------------------------------------- | ---------------------------------------- | ------------------------------------------------- |
| `type: object`                           | schema                                   | root                                              |
| `properties`                             | fields                                   | nested objects flatten to dotted attributes       |
| nested `object` property                 | dotted attributes                        | `profile` → `profile.name`                        |
| `items: {type: object}`                  | `array<json>` + `Items`                  | object array                                      |
| `items: {type: <scalar>}`                | `array<scalar>`                          |                                                   |
| `required: [...]`                        | `Field.Required`                         | only for single (non-flattened) fields            |
| `additionalProperties: false`            | `Mode = strict`                          |                                                   |
| `additionalProperties: true`/absent      | `Mode = flexible`                        |                                                   |
| `additionalProperties: {typed}`          | member → `json`, root → error            | `Annotations`                                     |
| `format: date-time`                      | `time`                                   |                                                   |
| `description`                            | `Field.Description` / `Def.Description`  | carried through                                   |
| `enum`/`const`/`pattern`/`min`/`max`     | annotations only                         | XDB does not enforce constraints                  |
| `$ref` (same document)                   | resolved and walked                      |                                                   |
| `allOf` of disjoint objects              | merged                                   |                                                   |

**Documented rejections** (each error names the JSON pointer to the offending node, where one exists):

- input that is not valid JSON → `ErrInvalidJSON`.
- no `WithNS` → `ErrMissingNamespace`. No schema name from `WithSchema`, `title`, or `$id` → `ErrMissingSchema`.
- `anyOf`/`oneOf` → `ErrUnion` (unions are a non-goal).
- a property name that contains `.`, or that is not a single attribute segment → `ErrInvalidKey`. There is no escaping.
- two definitions for the same field, from overlapping `allOf` branches or from a nested-object flatten → `ErrConflict`.
- a cross-document `$ref` → `ErrCrossDocument`. A cyclic `$ref` → `ErrCyclicRef`. Pass `WithOpaqueJSON("#/$defs/Node")` to store a cyclic node as JSON.
- an unresolved `$ref` → `ErrUnresolvedRef`.
- arrays of arrays, an untyped array, `allOf` inside an array element, or any other unmapped construct → `ErrUnsupported`.

## Go structs → schema.Def

`encoding/xdbstruct` reflects over a Go type. Fields carry an `xdb` struct tag that obeys the `encoding/json` conventions (`xdb:"name,required"`, `xdb:"-"`, `xdb:"attrs,json"`).

| Go type                            | schema.Def field type            | Notes                                             |
| ---------------------------------- | -------------------------------- | ------------------------------------------------- |
| `bool`/`int*`/`uint*`/`float*`     | matching scalar                  |                                                   |
| `string`                           | `string`                         |                                                   |
| `time.Time`                        | `time`                           |                                                   |
| `[]byte`                           | `bytes`                          |                                                   |
| `json.RawMessage`                  | `json`                           |                                                   |
| named scalar (`type UserID string`)| underlying scalar                | `Annotations["go.type"]` restores it on Unmarshal |
| nested `struct` / `*struct`        | dotted attributes                | a pointer is nullable                             |
| `[]Struct`                         | `array<json>` + `Items`          | object array                                      |
| `[]scalar`                         | `array<scalar>`                  |                                                   |
| embedded (anonymous) struct        | promoted fields                  | Go promotion rules                                |
| map / interface with `xdb:",json"` | `json`                           | explicit opt-in. Escape hatch for recursion       |

**Documented rejections** (each error names the field and the fix):

- a type that is not a struct → `ErrNotStruct`.
- a map or interface **without** the `json` opt-in → `ErrUnsupported`.
- a channel or function field → `ErrUnsupported`.
- a recursive type (`User → Manager → User`) → `ErrRecursive`, unless the recursive field opts into `json`.

## Null, absent, and zero

XDB distinguishes *absent* (no tuple for the attribute) from *null* (a tuple whose value is null). An explicit null satisfies a `Required` field. Absence does not. Each source produces these as follows:

| Source        | Absent (no tuple)                                        | Null                                    | Zero value                                  |
| ------------- | -------------------------------------------------------- | --------------------------------------- | ------------------------------------------- |
| Protobuf      | unset field, or a proto3 implicit-zero scalar            | not distinct in proto3 (use a wrapper)  | scalar zero is treated as absent            |
| Go structs    | a `nil` pointer field (also a `nil` embedded pointer)    | —                                       | a non-pointer zero marshals as its value    |
| JSON Schema   | a property absent from the document                      | an explicit JSON `null`                 | the literal value written                   |

A wrapper message (`google.protobuf.Int32Value`) or a Go pointer adds presence when the zero value must be distinguishable from absence.

## CLI: import and diff

```
xdb schemas import ./api/user.proto --ns com.example
xdb schemas import ./schemas/user.schema.json --ns com.example
xdb schemas diff   ./api/user.proto --ns com.example        # drift check
xdb schemas diff   ./api/user.proto --ns com.example --check # CI: non-zero on drift
```

`import` is create-or-update. It runs the importer, shows the delta (new schema, added, changed, and removed fields, suspected renames), and applies the delta through `schemas.create` or `schemas.update`. `--dry-run` prints the delta only. `--yes` skips confirmation for scripting. `diff` is the same walk without a write. With `--check`, `diff` exits non-zero on any drift, so CI fails on schema divergence instead of a strict write in production.

### Renames are proto-only

Only proto has field numbers, so only a proto rename is detected **reliably**. A field whose `proto.number` matches an existing field under a different name is a rename. `xdbproto.CheckRename(existing, updated)` finds such a field and returns `xdbproto.ErrRename`, which names the number, the old name, and the new name. A Go caller can use it to refuse the update. The CLI uses it as the signal for a proto rename. It applies the rename without acknowledgment, as a removal of the old field and an addition of the new field. There is no tuple migration.

A Go-struct or JSON-Schema rename cannot be distinguished from remove-old plus add-new. Applied silently, it orphans the stored tuples of the old attribute.

Guard: for a **non-proto** source, `import` **refuses** a suspected rename (matched by a type-plus-position heuristic), unless you acknowledge it with `--rename old:new` (repeatable) or `--yes`. `diff` shows it as a `rename?` warning. XDB never silently removes and re-adds a heuristic rename.

## Describe a stored schema

`xdb describe --uri xdb://ns/schema` shows what agents read. At the schema level: `description`, `mode`, `revision`, and source `annotations`. For each field: the type, the `required` flag, the description, the annotations, and the element schema (`items`).

## Related Concepts

- [Schemas](schemas.md) — The `Def` that importers produce, and its modes
- [Types](types.md) — The XDB value types that source types map onto
- [Encoding](encoding.md) — The JSON data path that `xdbjson` and the CLI use
- [Filters](filters.md) — Why dotted attributes are filterable and array elements are not
