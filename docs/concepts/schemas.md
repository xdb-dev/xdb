---
title: Schemas
description: Structure definitions with flexible, strict, and dynamic validation modes.
package: schema
---

# Schemas

A **Schema** defines the structure of [Records](records.md) and groups them together. Schemas control which fields a record may contain, what types those fields must have, and whether unknown fields are allowed.

## Schema Definition

A schema definition (`Def`) contains:

| Component  | Type                    | Description                            |
| ---------- | ----------------------- | -------------------------------------- |
| **URI**    | `*core.URI`             | Schema location (NS + Schema)          |
| **Fields** | `map[string]FieldDef`   | Field name to definition mapping       |
| **Mode**   | `Mode`                  | Validation behavior                    |

### Field Definitions

Each field has a type and a required flag:

```go
schema.FieldDef{
    Type:     core.TIDString,  // Expected value type
    Required: true,            // Whether the field must be present
}
```

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

Only declared fields are accepted. Values must match the declared type. Unknown fields produce `ErrUnknownField`.

```json
{
    "uri": "xdb://com.example/users",
    "mode": "strict",
    "fields": {
        "name":  { "type": "STRING",  "required": true  },
        "email": { "type": "STRING",  "required": true  },
        "age":   { "type": "INTEGER", "required": false }
    }
}
```

### Dynamic Mode

Like strict, declared fields are type-checked. But unknown fields are accepted and their types are inferred from the data, rather than being rejected.

## Validation

Schemas validate records at write time through the store layer.

```go
err := schema.ValidateTuples(def, tuples)
err := schema.ValidateRecords(def, records)
```

Validation checks:
1. **Field existence** — In strict and dynamic modes, unknown fields are flagged (strict rejects, dynamic accepts).
2. **Type matching** — The value's type must match the field's declared type. Mismatches produce `ErrTypeMismatch`.
3. **Required fields** — Fields marked `required: true` must be present.

### Errors

| Error                | Meaning                                   |
| -------------------- | ----------------------------------------- |
| `ErrUnknownField`    | Field not declared in schema (strict mode) |
| `ErrTypeMismatch`    | Value type does not match field type       |
| `ErrSchemaViolation` | Store-level schema violation               |

## JSON Representation

Schema definitions are stored and transmitted as JSON. The strict mode example above shows the format. Schemas are identified by their [URI](uris.md), which combines namespace and schema name (e.g., `xdb://com.example/posts`).

## Related Concepts

- [Records](records.md) — The data validated by schemas
- [Types](types.md) — The type identifiers used in field definitions
- [Namespaces](namespaces.md) — How schemas are organized
- [Stores](stores.md) — Where schemas are persisted
