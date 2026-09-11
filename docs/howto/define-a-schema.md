---
title: Define a schema
description: Declare fields and types, select a validation mode, and add indexes and constraints.
package: schema
read_when:
  - You create the schema for a new kind of record
  - A write fails with SCHEMA_VIOLATION or UNIQUE_VIOLATION
---

# Define a schema

A schema groups records and controls the validation of their fields. A namespace groups schemas by domain, application, or tenant.

<figure class="frame">
  <svg id="fig-nest" role="img" aria-label="Nested boxes: a namespace contains a schema, which contains a record, which is a list of tuples."></svg>
  <figcaption>resource hierarchy</figcaption>
</figure>

Each resource has a URI. The depth of the URI selects the resource:

| Resource | URI |
| --- | --- |
| Namespace | `xdb://com.example` |
| Schema | `xdb://com.example/users` |
| Record | `xdb://com.example/users/u-1` |
| Tuple | `xdb://com.example/users/u-1#email` |

## Declare the fields

Give each field a type. These markers are optional:

- `required`: each full write of a record must include the field.
- `indexed`: makes equality and `in` lookups faster.
- `unique`: a second record cannot have the same value.

```bash
xdb schemas create xdb://com.example/users --json '{
  "mode": "strict",
  "fields": {
    "email": { "type": "string", "required": true, "unique": true },
    "age":   { "type": "integer", "indexed": true },
    "tags":  { "type": "array", "elem_type": "string" }
  }
}'
```

The types are `string`, `integer`, `unsigned`, `float`, `boolean`, `time`, `json`, `bytes`, and `array`. An array declares the type of its elements with `elem_type`. An array of objects can also declare the fields of its items. [Types](../concepts/types.md) describes each type.

XDB adds the `_version` and `_updated` system fields to each schema.

## Select a mode

The mode controls a field that the schema does not declare. The store type-checks the declared fields in each mode.

| Mode | Undeclared field |
| --- | --- |
| `strict` | Rejected |
| `flexible` | Stored as it is. The schema does not record it. |
| `dynamic` | Stored. The schema adds the field with the type of the value. |

A `flexible` schema with no fields accepts any data:

```bash
xdb schemas create xdb://com.example/events --json '{"mode":"flexible"}'
```

## Read the errors

A write that breaks a rule fails with `SCHEMA_VIOLATION`. The `details` object gives the field:

```bash
xdb records create xdb://com.example/users/u-1 --json '{"email":"ada@example.com","nick":"ada"}'
```

```json
{
  "code": "SCHEMA_VIOLATION",
  "message": "[xdb/core] schema violation: [xdb/schema] unknown field [field=nick]",
  "resource": "records",
  "action": "create",
  "uri": "xdb://com.example/users/u-1",
  "hint": "run xdb describe --uri <schema-uri> to inspect the schema",
  "details": { "field": "nick" }
}
```

A missing required field gives `missing required field` with the same shape. A duplicate value in a `unique` field fails with `UNIQUE_VIOLATION`.

## Know where indexes apply

`indexed` and `unique` apply only to scalar fields. SQLite enforces them on `strict` and `dynamic` schemas: `indexed` creates an index, and `unique` rejects a duplicate value. The memory, filesystem, and Redis backends, and SQLite `flexible` schemas, keep the markers but do not enforce them.

To enforce uniqueness, use a `strict` or `dynamic` schema on SQLite.

## Change a schema

`xdb schemas update` patches a schema. You can add fields, and you can add or remove a field that has `indexed` or `unique`. The type of a field, the element type of an array, and the mode cannot change.

An update can include the revision that you read. If the stored revision is different, the update fails with `CONFLICT`. A revision of 0 skips this check.

To delete a schema and its records, pass `--force` and `--cascade`:

```bash
xdb schemas delete xdb://com.example/users --force --cascade
```

CAUTION: `--cascade` deletes each record in the schema.

[Schemas](../concepts/schemas.md) gives the full rules.
