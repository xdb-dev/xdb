---
title: Bring Your Own Types
description: Import protobuf, JSON Schema, and Go struct types into a schema.Def — the mapping tables, null handling, and documented rejections per source.
package: schema
---

# Bring Your Own Types

You already have types: protobuf messages, JSON Schemas, Go structs. XDB imports
those types into a [schema.Def](schemas.md) so you can store and query the data
without hand-writing a schema or giving up your source of truth.

Two rules anchor the design:

1. **The IR is one-way.** Proto files, JSON Schemas, and Go types stay the source
   of truth; `Def` is a projection produced by an importer. XDB never regenerates
   source from a stored schema.
2. **The contract is data round-trip, not schema round-trip.** Import a `User`,
   write a `User`, read the same `User` back — and filter/list it in between.
   Source-type fidelity that does not affect data round-trip (`int32` vs `int64`,
   enum names, proto field numbers) is preserved in opaque per-field
   annotations, not modeled.

There is one importer per source format:

| Source        | Package                                                 | Entry point                        | CLI                          |
| ------------- | ------------------------------------------------------- | ---------------------------------- | ---------------------------- |
| Go structs    | `encoding/xdbstruct`                                    | `Def[T]`, `Marshal`, `Unmarshal`   | in-process only (Go embed)   |
| Protobuf      | `schema/protoimport`                                    | `ImportFiles`, `ImportMessage`     | `xdb schemas import *.proto` |
| JSON Schema   | `schema/jsonschemaimport`                               | `Import`                           | `xdb schemas import *.json`  |

The CLI imports proto and JSON Schema **files**. The Go-struct path is
in-process (you embed XDB and call `xdbstruct`), not a CLI file import.

## Two nesting representations

Every importer shares one model for nested data, and it is worth stating up front
because it decides what you can filter:

- A **single nested object** (`*Profile`, a proto sub-message, a nested JSON
  Schema `object`) **flattens to dotted attributes** — `profile.name`,
  `profile.city`. These are first-class attributes: filters can address them.
- An **array of objects** (`[]Order`, `repeated Order`, `items: {type: object}`)
  uses `Field.Items`: the element object validates against the element field set,
  but the elements stay opaque. Filters cannot reach inside array elements (yet).

`Items` is arrays-only. A single object never uses it.

## Protobuf → schema.Def

`schema/protoimport` walks `protoreflect` descriptors (no codegen). The proto
package becomes the namespace unless `WithNamespace` (`--ns`) overrides it.

| Proto construct                                   | schema.Def field type            | Notes                                                   |
| ------------------------------------------------- | -------------------------------- | ------------------------------------------------------- |
| `bool`                                            | `boolean`                        |                                                         |
| `int32`/`sint32`/`sfixed32`/`int64`/…             | `integer`                        | width in `Annotations["proto.type"]`                    |
| `uint32`/`fixed32`/`uint64`/`fixed64`             | `unsigned`                       |                                                         |
| `float`/`double`                                  | `float`                          |                                                         |
| `string`                                          | `string`                         |                                                         |
| `bytes`                                           | `bytes`                          |                                                         |
| `enum`                                            | `string` (value name)            | `Annotations["proto.enum"]`                             |
| `google.protobuf.Timestamp`                       | `time`                           |                                                         |
| wrapper (`Int32Value`, `StringValue`, …)          | the wrapped scalar               | adds presence                                           |
| `Duration`/`Struct`/`Value`/`ListValue`/`Any`     | `json`                           | `Annotations["proto.type"]`                             |
| `map<K, V>`                                        | `json`                           | `Annotations["proto.map"]`; opt in explicitly           |
| nested message (single)                           | dotted attributes                | `address` → `address.city`, `address.zip`               |
| `repeated <scalar/enum>`                          | `array<scalar>`                  |                                                         |
| `repeated <message>`                              | `array<json>` + `Items`          | object array                                            |

Every field records `Annotations["proto.number"]` — the field number is what
makes proto renames reliably detectable (see [Renames](#renames-are-proto-only)).

**Documented rejections** (each names the field and the fix):

- `oneof` → `ErrOneof`. Unions are a non-goal; flatten into optional fields.
- a recursive message (`Node` referring to `Node`) → `ErrRecursive`. Escape with
  `WithAllowJSON("pkg.Node")` (`--allow-json pkg.Node`) to store it as JSON.

## JSON Schema → schema.Def

`schema/jsonschemaimport` targets a documented subset of draft 2020-12. The root
must describe an object. The namespace comes from `WithNamespace` (`--ns`); the
schema name from `WithSchemaName`, the document `title`, or the `$id` filename.

| JSON Schema construct                    | schema.Def result                        | Notes                                             |
| ---------------------------------------- | ---------------------------------------- | ------------------------------------------------- |
| `type: object`                           | schema                                   | root                                              |
| `properties`                             | fields                                   | nested objects flatten to dotted keys             |
| nested `object` property                 | dotted attributes                        | `profile` → `profile.name`                        |
| `items: {type: object}`                  | `array<json>` + `Items`                  | object array                                      |
| `items: {type: <scalar>}`                | `array<scalar>`                          |                                                   |
| `required: [...]`                        | `Field.Required`                         | only for single (non-flattened) fields            |
| `additionalProperties: false`            | `Mode = strict`                          |                                                   |
| `additionalProperties: true`/absent      | `Mode = flexible`                        |                                                   |
| `additionalProperties: {typed}`          | member → `json`; root → error            | `Annotations`                                     |
| `format: date-time`                      | `time`                                   |                                                   |
| `description`                            | `Field.Description` / `Def.Description`  | carried through                                   |
| `enum`/`const`/`pattern`/`min`/`max`     | annotations only                         | XDB does not enforce constraints                  |
| `$ref` (same document)                   | resolved and walked                      |                                                   |
| `allOf` of disjoint objects              | merged                                   |                                                   |

**Documented rejections** (each names the JSON pointer to the offending node):

- `anyOf`/`oneOf` → `ErrUnion` (unions are a non-goal).
- a property name containing `.`, or one that is not a single attribute segment
  → `ErrInvalidKey`. No escaping in v1.
- a cross-document `$ref` → `ErrCrossDocument`; a cyclic `$ref` → `ErrCyclicRef`
  (escape with `WithJSON("#/$defs/Node")`).
- an unresolved `$ref` → `ErrUnresolvedRef`.
- arrays of arrays, an untyped array, `allOf` inside an array element, or any
  other unmapped construct → `ErrUnsupported`.

## Go structs → schema.Def

`encoding/xdbstruct` reflects over a Go type. Fields carry an `xdb` struct tag
that follows `encoding/json` conventions (`xdb:"name,required"`, `xdb:"-"`,
`xdb:"attrs,json"`).

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
| map / interface with `xdb:",json"` | `json`                           | explicit opt-in; escape hatch for recursion       |

**Documented rejections** (each names the field and the fix):

- a map or interface **without** the `json` opt-in.
- a channel or function field.
- a recursive type (`User → Manager → User`), unless the recursive field opts
  into `json`.

## Null, absent, and zero

XDB distinguishes *absent* (no tuple for the attribute) from *null* (a tuple
whose value is null). A `Required` field is satisfied by an explicit null but not
by absence. How each source produces these:

| Source        | Absent (no tuple)                                        | Null                                    | Zero value                                  |
| ------------- | -------------------------------------------------------- | --------------------------------------- | ------------------------------------------- |
| Protobuf      | unset field, or a proto3 implicit-zero scalar            | not distinct in proto3 (use a wrapper)  | scalar zero is treated as absent            |
| Go structs    | a `nil` pointer field (also a `nil` embedded pointer)    | —                                       | a non-pointer zero marshals as its value    |
| JSON Schema   | a property absent from the document                      | an explicit JSON `null`                 | the literal value written                   |

A wrapper message (`google.protobuf.Int32Value`) or a Go pointer is how you add
presence when the zero value must be distinguishable from absence.

## CLI: import and diff

```
xdb schemas import ./api/user.proto --ns com.example
xdb schemas import ./schemas/user.schema.json --ns com.example
xdb schemas diff   ./api/user.proto --ns com.example        # drift check
xdb schemas diff   ./api/user.proto --ns com.example --check # CI: non-zero on drift
```

`import` is create-or-update: it runs the importer, shows the delta (new schema,
added/changed/removed fields, suspected renames), and applies it via
`schemas.create`/`schemas.update`. `--dry-run` prints the delta only; `--yes`
skips confirmation for scripting. `diff` is the same walk without writing;
`--check` exits non-zero on any drift so CI fails on schema divergence instead of
failing a strict write in production.

### Renames are proto-only

Only proto has field numbers, so only a proto rename is **reliably** detected: a
field whose `proto.number` matches an existing field under a different name is a
rename. A Go-struct or JSON-Schema rename is indistinguishable from
remove-old + add-new, and applying it silently orphans the old attribute's stored
tuples (there is no tuple migration in v1).

Guard: for a **non-proto** source, a suspected rename (matched by a type+position
heuristic) is **refused** by `import` unless you acknowledge it with `--rename
old:new` (repeatable) or `--yes`. `diff` surfaces it as a `rename?` warning. XDB
never silently drops and re-adds a heuristic rename.

## Describe a stored schema

`xdb describe --uri xdb://ns/schema` surfaces what agents read: the schema
`description`, `mode`, `revision`, and source `annotations`, plus each field's
type, `required` flag, description, annotations, and element schema (`items`).
