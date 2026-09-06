---
title: Schemas
description: Structure definitions with strict, flexible, and dynamic validation modes.
package: schema
---

# Schemas

A **Schema** defines the structure of [Records](records.md) and groups them. A schema controls which fields a record can contain, which types those fields must have, and what happens to undeclared fields.

## Schemas in the CLI

Schemas are a resource (`xdb schemas <action>`) and also a type reference that other actions use:

- Show a live schema: `xdb describe --uri xdb://ns/schema`
- Project fields on reads: `xdb records list xdb://ns/schema --fields _id,title`
- Create or update from JSON: `xdb schemas create xdb://ns/schema --json '{"fields":{...}}'`

See the [CLI reference](../../cmd/xdb/cli/CONTEXT.md) for the full grammar.

## Schema Definition

A schema definition (`Def`) contains:

| Component       | Type                      | Description                                      |
| --------------- | ------------------------- | ------------------------------------------------ |
| **URI**         | `*core.URI`               | Schema location (NS + Schema)                    |
| **Fields**      | `map[string]schema.Field` | Field name to field definition                   |
| **Mode**        | `Mode`                    | How undeclared fields are handled                |
| **Description** | `string`                  | Free-text description of the schema              |
| **Annotations** | `map[string]string`       | Arbitrary key-value metadata                     |
| **Revision**    | `int64`                   | Increments on each update. Drives the update CAS |

### Field Definitions

A `Field` has these members:

| Member          | Type                | Description                                                     |
| --------------- | ------------------- | --------------------------------------------------------------- |
| **Type**        | `core.Type`         | The value type. For arrays, it also carries the element type    |
| **Required**    | `bool`              | The field must be present on a full-record write                |
| **Indexed**     | `bool`              | The field has an index. See [Indexed and unique fields](#indexed-and-unique-fields) |
| **Unique**      | `bool`              | The field has a unique constraint. See [Indexed and unique fields](#indexed-and-unique-fields) |
| **Items**       | `map[string]Field`  | The element object schema of an `ARRAY<JSON>` field. See [Object arrays](#object-arrays) |
| **Description** | `string`            | Free-text description of the field                              |
| **Annotations** | `map[string]string` | Arbitrary key-value metadata                                    |

In JSON, the members are `type`, `elem_type`, `required`, `indexed`, `unique`, `items`, `description`, and `annotations`.

```go
// A scalar field.
schema.Field{
    Type:     core.NewType(core.TIDString), // expected value type
    Required: true,                         // must be present
}

// An array field. The element type is part of the Type.
schema.Field{
    Type: core.NewArrayType(core.TIDString), // ARRAY<STRING>
}
```

#### Array fields

The element type of an array field is part of its `Type`, built with
`core.NewArrayType`. Every array field must declare one. In JSON, use the
`elem_type` property. This rule applies in **every mode**, including
`flexible`. The element type is part of the declared shape of the field, not
a validation option. `CreateSchema` and `UpdateSchema` reject a definition
that omits it with `ErrInvalidField`.

The type of a field, including the element type of an array, is
**immutable** after the field exists. The **mode** of a schema is also
immutable. `UpdateSchema` rejects a type change with `ErrImmutableField` and
a mode change with `ErrImmutableMode`. You can add new fields and remove
existing fields.

#### Object arrays

An `ARRAY<JSON>` field can declare `Items`: the schema of each element
object. Each element must be a JSON object. Its members type-check against
`Items` with the same rules as top-level fields, including `Required`.
`Items` is valid only on `ARRAY<JSON>` fields. A single nested object does
not use `Items`. It flattens to dotted attributes, for example
`profile.name`.

#### Indexed and unique fields

A scalar field can be marked `indexed` to make equality and `in` lookups
faster, or `unique` to declare that its values must not repeat. Both are
declared per field:

```go
schema.Field{Type: core.NewType(core.TIDString), Indexed: true} // lookup key
schema.Field{Type: core.NewType(core.TIDString), Unique: true}  // unique constraint
```

In JSON: `{"type": "string", "indexed": true}` or `{"type": "string", "unique": true}`.

Both markers are backend capabilities, not store policy. XDB stores them
on every backend, and the backend applies them if it can.

On SQLite, a `strict` or `dynamic` schema gets a real index on the column
table (`CREATE INDEX`, or `CREATE UNIQUE INDEX` for `unique`). There
`indexed` makes lookups faster, and a duplicate write on a `unique` field
fails with `core.ErrUniqueViolation`. Everywhere else — memory,
filesystem, redis, and the SQLite key-value tables of a `flexible`
schema — both markers are stored declarations that change no behavior. A
duplicate value is accepted.

Declare `unique` for the intent, and pick a backend that materializes it
if you need the guarantee.

Rules:

- **Scalar only.** `indexed` and `unique` are rejected on `ARRAY` and `JSON`
  fields with `ErrInvalidField`. Those values are serialized, and the filter
  pushdown cannot compare them by equality.
- **Fixed at creation.** Like `type`, the markers are immutable.
  `UpdateSchema` rejects a change to them with `ErrImmutableField`. You can
  add a new indexed or unique field, and you can remove one.
- **Omitted on a patch keeps the marker.** A `schemas update` patch replaces
  each field it names. The markers are the exception: a patch that omits
  `indexed` or `unique` keeps the stored value, so you can edit the
  description or the required flag of a marked field without restating the
  marker. A patch that gives the key a new value is still rejected.
- **Present values only.** Where `unique` is materialized, it constrains
  present values only. Many records can omit the field, because NULL values
  are distinct.

## Modes

The mode of a schema controls only undeclared fields. Declared fields are type-checked in every mode.

| Mode         | Declared fields | Undeclared fields                     | Use case                          |
| ------------ | --------------- | ------------------------------------- | --------------------------------- |
| **strict**   | Type-checked    | Rejected with `ErrUnknownField`       | Production data with fixed shapes |
| **flexible** | Type-checked    | Accepted and stored as-is             | Semi-structured data              |
| **dynamic**  | Type-checked    | Type inferred and added to the schema | Evolving data with type safety    |

`strict` is the default. A definition without a `mode` is created as `strict`.

### Strict Mode

Only declared fields are accepted. Values must match the declared type. An undeclared field produces `ErrUnknownField`. For an array field, the element type of the value must match the declared `elem_type`. A mismatch produces `ErrTypeMismatch`.

```json
{
    "uri": "xdb://com.example/users",
    "mode": "strict",
    "fields": {
        "name":  { "type": "string",  "required": true  },
        "email": { "type": "string",  "required": true  },
        "age":   { "type": "integer", "required": false },
        "tags":  { "type": "array",   "elem_type": "string" }
    }
}
```

### Flexible Mode

Declared fields are type-checked, the same as in `strict` mode. Undeclared fields are accepted and stored as-is. The schema does not record them. A `flexible` schema with no fields accepts any data.

```bash
xdb schemas create xdb://com.example/events --json '{"mode":"flexible"}'
```

On SQLite, a `flexible` schema uses a key-value table instead of a column table. See [Indexed and unique fields](#indexed-and-unique-fields).

### Dynamic Mode

Declared fields are type-checked, the same as in `strict` mode. An undeclared field is not rejected. Its type is inferred from the data and added to the schema. When the inferred type is `array`, the element type is taken from the value and persisted as part of the field. Later writes must use the same element type.

## Validation

Schemas are validated at two boundaries:

1. **Schema declaration** — when a schema is created or updated, the
   declaration is checked for well-formedness and compatibility.
2. **Record writes** — at write time, the store layer validates values
   against the field definitions.

Every stored definition also carries the `_version` and `_updated`
[system fields](versioning.md). The store stamps them. They are not part of
what you declare. Schema import strips them, and a definition that you send
back on update is stamped again.

### Declaration checks

`CreateSchema` and `UpdateSchema` reject a malformed or incompatible definition:

```go
err := def.Validate()                    // well-formedness
err := schema.ValidateUpdate(old, new)   // compatibility with the existing definition
```

`def.Validate()` applies these well-formedness rules:

- **Mode** — the mode must be `strict`, `flexible`, or `dynamic`. An empty or
  unknown mode produces `ErrInvalidMode`.
- **Reserved names** — a top-level field name cannot start with `_`. That
  prefix belongs to the [system fields](versioning.md). A violation produces
  `ErrInvalidField`. The rule is top-level only. The `Items` of an
  object-array field are a separate namespace inside a JSON value, so an
  element field named `_id` is legal.
- **Field names** — every field name must parse as an attribute path. A
  violation produces `ErrInvalidField`.
- **Path prefixes** — a field name cannot be a path prefix of another field.
  For example, `author` and `author.name` cannot both be fields. A violation
  produces `ErrInvalidField`.
- **Array element type** — every `array` field must declare an element type
  (`elem_type` in JSON). A violation produces `ErrInvalidField`.
- **Indexed and unique** — `indexed` and `unique` are valid only on scalar
  fields. A violation produces `ErrInvalidField`.
- **Items** — `items` is valid only on `ARRAY<JSON>` fields, and its fields
  are validated with the same rules. A violation produces `ErrInvalidField`.

`schema.ValidateUpdate(old, new)` applies the compatibility rules:

- **Immutability** — the type of an existing field, including the element
  type of an array, cannot change. Its `indexed` and `unique` markers cannot
  change. The mode of the schema cannot change. Violations produce
  `ErrImmutableField` and `ErrImmutableMode`. You can add new fields and
  remove existing fields.
- **Revision CAS** — an update carries the `Revision` it is based on. A stale
  base is rejected with `core.ErrConflict`. See [Stores](stores.md).

The store wraps the declaration errors in `core.ErrSchemaViolation`.
`core.ErrConflict` is returned as-is.

### Record write checks

```go
err := schema.ValidateTuples(def, tuples)
err := schema.CheckRequired(def, tuples)
```

1. **Field existence** — `ValidateTuples` handles undeclared fields by mode. `strict` rejects them. `flexible` ignores them. A `dynamic` schema uses `schema.EvolveDynamic` instead, which infers the new fields.
2. **Type matching** — the type of the value must match the declared type of the field, including the element type for arrays. A mismatch produces `ErrTypeMismatch`. An explicit null carries no type, so it satisfies any declared type. A `dynamic` schema infers no field from a null, and adds the field when a typed value arrives.
3. **Required fields** — `CheckRequired` makes sure that every field marked `required: true` has a tuple. The store calls it on writes that carry the full attribute set of a record: `create`, `upsert`, and a patch that creates a new record.

### Errors

| Error                     | Meaning                                                                          |
| ------------------------- | -------------------------------------------------------------------------------- |
| `ErrInvalidMode`          | The mode is empty or not one of `strict`, `flexible`, `dynamic`                  |
| `ErrUnknownField`         | The field is not declared in the schema (`strict` mode)                          |
| `ErrTypeMismatch`         | The value type or the array element type does not match                          |
| `ErrMissingRequired`      | A `required` field has no tuple on a full-record write                           |
| `ErrInvalidField`         | The field declaration is malformed. See the well-formedness rules                |
| `ErrImmutableField`       | The update changes the type, element type, `indexed`, or `unique` of a field     |
| `ErrImmutableMode`        | The update changes the mode of the schema                                        |
| `core.ErrConflict`        | The base revision of the update is stale (CAS failure). Returned as-is           |
| `core.ErrUniqueViolation` | A write duplicates the value of a `unique` field (SQLite column tables only)     |
| `core.ErrSchemaViolation` | The store-level wrapper around the `schema` errors above                         |

## JSON Representation

Schema definitions are stored and transmitted as JSON. The `strict` mode example above shows the format. A schema is identified by its [URI](uris.md), which combines the namespace and the schema name, for example `xdb://com.example/posts`.

## Related Concepts

- [Records](records.md) — The data that schemas validate
- [Types](types.md) — The type identifiers used in field definitions
- [Namespaces](namespaces.md) — How schemas are organized
- [Stores](stores.md) — Where schemas are persisted
