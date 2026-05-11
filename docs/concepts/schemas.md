---
title: Schemas
description: Structure definitions with flexible, strict, and dynamic validation modes.
package: schema
---

# Schemas

A **Schema** defines the structure of [Records](records.md) and groups them together. Schemas control which fields a record may contain, what types those fields must have, and whether unknown fields are allowed.

## Schemas in the CLI

Schemas are both a resource (`xdb schemas <action>`) and a type reference used by other actions:

- Discover a live schema: `xdb describe --uri xdb://ns/schema`
- Project fields on reads: `xdb records list xdb://ns/schema --fields id,title`
- Create/update from JSON: `xdb schemas create xdb://ns/schema --json '{"fields":{...}}'`

See the [CLI reference](../../cmd/xdb/cli/CONTEXT.md) for the full grammar.

## Schema Definition

A schema definition (`Def`) contains:

| Component  | Type                    | Description                            |
| ---------- | ----------------------- | -------------------------------------- |
| **URI**    | `*core.URI`             | Schema location (NS + Schema)          |
| **Fields** | `map[string]FieldDef`   | Field name to definition mapping       |
| **Mode**   | `Mode`                  | Validation behavior                    |

### Field Definitions

Each field has a type, an optional element type (for arrays), and a required flag:

```go
schema.FieldDef{
    Type:     core.TIDString,  // Expected value type
    Required: true,            // Whether the field must be present
}

// Array fields must also declare the element type:
schema.FieldDef{
    Type:     core.TIDArray,
    ElemType: core.TIDString,  // ARRAY<STRING>
}
```

#### Array fields

When `Type` is `core.TIDArray`, `ElemType` must be set. This rule applies in
**every mode**, including `flexible` — `ElemType` is part of the field's
declared shape, not a validation toggle. Schemas that omit it are rejected at
`CreateSchema` / `UpdateSchema` with `ErrInvalidField` (wrapped as
`store.ErrSchemaViolation`).

Both `Type` and `ElemType` are **immutable** once a field exists. `UpdateSchema`
rejects changes with `ErrImmutableField`. Adding new fields and removing
existing fields are still allowed.

## Modes

Schemas support three validation modes:

| Mode         | Unknown Fields | Type Checking | Use Case                          |
| ------------ | -------------- | ------------- | --------------------------------- |
| **flexible** | Allowed        | None          | Schemaless data, rapid prototyping |
| **strict**   | Rejected       | Enforced      | Production data with fixed shapes  |
| **dynamic**  | Auto-inferred  | Enforced      | Evolving data with type safety     |

### Flexible Mode

No validation is performed. Any fields with any types are accepted. This is the default when creating a schema without a definition file.

```bash
xdb make-schema xdb://com.example/events
```

### Strict Mode

Only declared fields are accepted. Values must match the declared type. Unknown fields produce `ErrUnknownField`. For array fields, the value's element type must match the declared `elem_type`; mismatches produce `ErrTypeMismatch`.

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

### Dynamic Mode

Like strict, declared fields are type-checked. But unknown fields are accepted and their types are inferred from the data, rather than being rejected. When the inferred type is `array`, the element type is captured from the value and persisted as part of the field — subsequent writes must use the same element type.

## Validation

Schemas are validated at two boundaries:

1. **Schema declaration** — when a schema is created or updated, the
   declaration itself is checked for well-formedness and compatibility.
2. **Record writes** — values are validated against the schema's field
   definitions at write time through the store layer.

### Declaration checks

`CreateSchema` and `UpdateSchema` reject malformed or incompatible schemas:

```go
err := def.Validate()                    // well-formedness
err := schema.ValidateUpdate(old, new)   // compatibility with existing
```

- **Well-formedness** — every `array` field must declare an `elem_type`.
  Violations produce `ErrInvalidField`.
- **Immutability** — on update, an existing field's `Type` and `ElemType`
  cannot change. Violations produce `ErrImmutableField`. Adding new fields
  and removing existing fields are allowed.

Stores wrap these as `store.ErrSchemaViolation`.

### Record write checks

```go
err := schema.ValidateTuples(def, tuples)
err := schema.ValidateRecords(def, records)
```

1. **Field existence** — In strict and dynamic modes, unknown fields are flagged (strict rejects, dynamic infers).
2. **Type matching** — The value's type must match the field's declared type, including the element type for arrays. Mismatches produce `ErrTypeMismatch`.
3. **Required fields** — Fields marked `required: true` must be present.

### Errors

| Error                | Meaning                                              |
| -------------------- | ---------------------------------------------------- |
| `ErrUnknownField`    | Field not declared in schema (strict mode)           |
| `ErrTypeMismatch`    | Value type or array element type does not match      |
| `ErrInvalidField`    | Field declaration is malformed (e.g. array missing `elem_type`) |
| `ErrImmutableField`  | Update would change an existing field's `Type` or `ElemType` |
| `ErrSchemaViolation` | Store-level wrapper around any of the above          |

## JSON Representation

Schema definitions are stored and transmitted as JSON. The strict mode example above shows the format. Schemas are identified by their [URI](uris.md), which combines namespace and schema name (e.g., `xdb://com.example/posts`).

## Related Concepts

- [Records](records.md) — The data validated by schemas
- [Types](types.md) — The type identifiers used in field definitions
- [Namespaces](namespaces.md) — How schemas are organized
- [Stores](stores.md) — Where schemas are persisted
