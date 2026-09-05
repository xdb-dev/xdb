---
name: schema-evolution
description: "Evolve schemas safely with patch updates and revision CAS"
category: recipe
---

# Schema Evolution

A schema update is a patch: fields are added or replaced, never
removed. Every update increments the `revision` of the schema.

## Inspect the current definition

```bash
xdb schemas get --uri xdb://myapp/todos
xdb describe --uri xdb://myapp/todos
```

## Add a field

```bash
xdb schemas update --uri xdb://myapp/todos --json '{
  "fields": {
    "priority": {"type": "integer"}
  }
}'
```

Existing records stay readable. The new field is absent until a record
sets it. Add a field as optional first. If you mark a field
`"required": true`, writes that omit the field fail.

## Safe rollout order

1. Run `schemas update` to add the new optional field.
2. Backfill the existing records with `xdb records update` or a bulk import.
3. When every record carries the field, mark the field required.

## Concurrency: revision CAS

`schemas get` returns the current `revision`. If you pass this
`revision` in an update, and another update changed the schema in the
meantime, the update fails with CONFLICT. If you omit `revision`, the
update is unconditional:

```bash
xdb schemas update --uri xdb://myapp/todos --json '{
  "revision": 3,
  "fields": {"priority": {"type": "integer"}}
}'
```

## Conflicts on create

If you create a schema again with an identical definition, the create
succeeds and is idempotent. If the definition differs, the create fails
with CONFLICT. To change a schema, use `schemas update`.

## Modes

`xdb describe --schema-format` documents the three validation modes.
`strict` (the default) rejects undeclared fields. `flexible` accepts
undeclared fields as-is. `dynamic` infers undeclared fields and adds
them to the schema. Declared fields type-check in every mode.
