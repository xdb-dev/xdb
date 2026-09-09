---
title: Bring Your Own Types
description: Import protobuf, JSON Schema, and Go struct types into a schema.Def, with type mappings, null handling, and unsupported constructs.
package: encoding/xdbjson, encoding/xdbproto, encoding/xdbstruct
---

# Bring Your Own Types

XDB imports protobuf messages, JSON Schema documents, and Go structs into a [schema.Def](schemas.md). Use an imported definition to store and query data without maintaining a separate XDB schema.

The source types remain authoritative:

1. Importers produce a `Def` from the source type. XDB never regenerates source from a stored schema.

2. Adapters preserve data through a write and read: import a `User`, write it, and read it back as a `User`. Source details such as integer widths, enum names, and proto field numbers remain in opaque field annotations.

Each adapter under `encoding/` imports schema definitions and converts values between its source format and XDB records.

| Source      | Package              | Schema path                    | Data path              | CLI                                   |
| ----------- | -------------------- | ------------------------------ | ---------------------- | ------------------------------------- |
| Go structs  | `encoding/xdbstruct` | `Def[T]`                       | `Marshal`, `Unmarshal` | in-process only (Go embed)            |
| Protobuf    | `encoding/xdbproto`  | `ImportFiles`, `ImportMessage` | `Marshal`, `Unmarshal` | `xdb schemas import user.proto`       |
| JSON Schema | `encoding/xdbjson`   | `ImportSchema`                 | `Marshal`, `Unmarshal` | `xdb schemas import user.schema.json` |

The CLI imports one protobuf or JSON Schema file per command. To use Go structs, embed XDB and call `xdbstruct` in your application.

## Nested Objects and Arrays

All importers use the same nested-data representation:

- A single nested object (`*Profile`, a proto sub-message, a nested JSON Schema `object`) flattens to dotted attributes: `profile.name`, `profile.city`. Filters can address these attributes.

- An array of objects (`[]Order`, `repeated Order`, `items: {type: object}`) uses `Field.Items`. Each element validates against the element field set, but the elements stay opaque. Filters cannot reach inside array elements.

`Items` is for arrays only. A single object never uses it.

## Protobuf to schema.Def

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
| nested message (single)                           | dotted attributes                | `address` -> `address.city`, `address.zip`                    |
| `repeated <scalar/enum>`                          | `array<scalar>`                  |                                                              |
| `repeated <message>`                              | `array<json>` + `Items`          | object array                                                 |

Every field records `Annotations["proto.number"]`. The field number is what makes a proto rename reliably detectable (see [Renames](#renames-are-proto-only)).

Documented rejections (each error names the offending item and the fix):

- a file with no proto package and no `WithNamespace` -> `ErrNoNamespace`.

- `oneof` -> `ErrOneof`. Unions are unsupported. Flatten the union into optional fields.

- a recursive message (`Node` that refers to `Node`) -> `ErrRecursive`. Pass `WithAllowJSON("pkg.Node")` (`--allow-json pkg.Node`) to store it as JSON.

## JSON Schema to schema.Def

`encoding/xdbjson` targets a documented subset of draft 2020-12. The root must describe an object. The namespace comes from `WithNS` (`--ns`). The schema name comes from `WithSchema`, the document `title`, or the `$id` filename, in that order.

| JSON Schema construct                    | schema.Def result                        | Notes                                             |
| ---------------------------------------- | ---------------------------------------- | ------------------------------------------------- |
| `type: object`                           | schema                                   | root                                              |
| `properties`                             | fields                                   | nested objects flatten to dotted attributes       |
| nested `object` property                 | dotted attributes                        | `profile` -> `profile.name`                        |
| `items: {type: object}`                  | `array<json>` + `Items`                  | object array                                      |
| `items: {type: <scalar>}`                | `array<scalar>`                          |                                                   |
| `required: [...]`                        | `Field.Required`                         | only for single (non-flattened) fields            |
| `additionalProperties: false`            | `Mode = strict`                          |                                                   |
| `additionalProperties: true`/absent      | `Mode = flexible`                        |                                                   |
| `additionalProperties: {typed}`          | member -> `json`, root -> error            | `Annotations`                                     |
| `format: date-time`                      | `time`                                   |                                                   |
| `description`                            | `Field.Description` / `Def.Description`  | carried through                                   |
| `enum`/`const`/`pattern`/`min`/`max`     | annotations only                         | XDB does not enforce constraints                  |
| `$ref` (same document)                   | resolved and walked                      |                                                   |
| `allOf` of disjoint objects              | merged                                   |                                                   |

Documented rejections (each error names the JSON pointer to the offending node, where one exists):

- input that is not valid JSON -> `ErrInvalidJSON`.

- no `WithNS` -> `ErrMissingNamespace`. No schema name from `WithSchema`, `title`, or `$id` -> `ErrMissingSchema`.

- `anyOf`/`oneOf` -> `ErrUnion` (unions are unsupported).

- a property name that contains `.`, or that is not a single attribute segment -> `ErrInvalidKey`. There is no escaping.

- two definitions for the same field, from overlapping `allOf` branches or from a nested-object flatten -> `ErrConflict`.

- a cross-document `$ref` -> `ErrCrossDocument`. A cyclic `$ref` -> `ErrCyclicRef`. Pass `WithOpaqueJSON("#/$defs/Node")` to store a cyclic node as JSON.

- an unresolved `$ref` -> `ErrUnresolvedRef`.

- arrays of arrays, an untyped array, `allOf` inside an array element, or any other unmapped construct -> `ErrUnsupported`.

## Go Structs to schema.Def

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
| map / interface with `xdb:",json"` | `json`                           | explicit opt-in. Stores recursive values as JSON       |

Documented rejections (each error names the field and the fix):

- a type that is not a struct -> `ErrNotStruct`.

- a map or interface without the `json` opt-in -> `ErrUnsupported`.

- a channel or function field -> `ErrUnsupported`.

- a recursive type (`User → Manager → User`) -> `ErrRecursive`, unless the recursive field opts into `json`.

## Null, Absent, and Zero

XDB distinguishes *absent* (no tuple for the attribute) from *null* (a tuple whose value is null). An explicit null satisfies a `Required` field. Absence does not. Each source produces these as follows:

| Source        | Absent (no tuple)                                        | Null                                    | Zero value                                  |
| ------------- | -------------------------------------------------------- | --------------------------------------- | ------------------------------------------- |
| Protobuf      | unset field, or a proto3 implicit-zero scalar            | not distinct in proto3 (use a wrapper)  | scalar zero is treated as absent            |
| Go structs    | a `nil` pointer field (also a `nil` embedded pointer)    |:                                       | a non-pointer zero marshals as its value    |
| JSON Schema   | a property absent from the document                      | an explicit JSON `null`                 | the literal value written                   |

A wrapper message (`google.protobuf.Int32Value`) or a Go pointer adds presence when the zero value must be distinguishable from absence.

## CLI: Import and Diff

```
xdb schemas import ./api/user.proto --ns com.example
xdb schemas import ./schemas/user.schema.json --ns com.example
xdb schemas diff   ./api/user.proto --ns com.example        # drift check
xdb schemas diff   ./api/user.proto --ns com.example --check # CI: non-zero on drift
```

`import` is create-or-update. It runs the importer, shows the delta (new schema, added, changed, and removed fields, suspected renames), and applies the delta through `schemas.create` or `schemas.update`. `--dry-run` prints the delta only. `--yes` skips confirmation for scripting. `diff` is the same walk without a write. With `--check`, `diff` exits non-zero on any drift, so CI fails on schema divergence instead of a strict write in production.

### Renames Are Proto-Only

Only proto has field numbers, so only a proto rename is detected reliably. A field whose `proto.number` matches an existing field under a different name is a rename. `xdbproto.CheckRename(existing, updated)` finds such a field and returns `xdbproto.ErrRename`, which names the number, the old name, and the new name. A Go caller can use it to refuse the update. The CLI uses it as the signal for a proto rename. It applies the rename without acknowledgment, as a removal of the old field and an addition of the new field. There is no tuple migration.

A Go-struct or JSON-Schema rename cannot be distinguished from remove-old plus add-new. Applied silently, it orphans the stored tuples of the old attribute.

For a non-proto source, `import` refuses a suspected rename (matched by a type-plus-position heuristic), unless you acknowledge it with `--rename old:new` (repeatable) or `--yes`. `diff` shows it as a `rename?` warning. Acknowledgment permits the removal and addition; it does not migrate stored tuples.

## Describe a Stored Schema

`xdb describe --uri xdb://ns/schema` shows what agents read. At the schema level: `description`, `mode`, `revision`, and source `annotations`. For each field: the type, the `required` flag, the description, the annotations, and the element schema (`items`).

## Related Concepts

- [Schemas](schemas.md): The `Def` that importers produce, and its modes

- [Types](types.md): The XDB value types that source types map onto

- [Encoding](encoding.md): The JSON data path that `xdbjson` and the CLI use

- [Filters](filters.md): Why dotted attributes are filterable and array elements are not
